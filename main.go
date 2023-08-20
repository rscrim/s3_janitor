package main

import (
	"fmt"
	"os"
	s3Access "s3_mp_janitor/aws"
	"s3_mp_janitor/config"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const allProfiles = "ALL_PROFILES"

var (
	discoverFlag bool
	profileFlag  string
	helpFlag     bool
)

func init() {
	rootCmd.Flags().BoolVarP(&discoverFlag, "discover", "d", false, "Discover and manage failed S3 multipart uploads")
	rootCmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "AWS profile to use. If not specified, it will prompt interactively.")
	rootCmd.Flags().BoolVarP(&helpFlag, "help", "h", false, "Help page for the s3-janitor tool.")
}

var rootCmd = &cobra.Command{
	Use:   "s3_mp_janitor",
	Short: "A tool to manage expiring S3 multipart uploads",
	Run: func(cmd *cobra.Command, args []string) {
		if helpFlag {
			cmd.Help() // Display help information
			return
		}

		if discoverFlag {
			discover(cmd, args)
		} else {
			menu()
		}
	},
}

// discover is the command's logic for the discover sub-command.
// It prints the selected profile or defaults to "all profiles" if none is provided.
func discover(cmd *cobra.Command, args []string) {
	// If the profile flag is not set or set to "ALL_PROFILES", let the user choose interactively
	if profileFlag == "" || profileFlag == allProfiles {
		profile, err := GetProfileChoice()
		if err != nil {
			fmt.Printf("Error selecting profile: %v\n", err)
			return
		}
		profileFlag = profile
	}

	credentials, credErr := s3Access.GetCredentialsForProfile(profileFlag)
	if credErr != nil {
		fmt.Printf("Error fetching credentials for %s: %v\n", profileFlag, credErr)
		return
	}
	// Use these credentials to create the session
	sess, err := s3Access.CreateAWSSessionWithCredentials(profileFlag, credentials) // Assuming you modify or create such a function in aws package
	if err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		return
	}

	bucket, err := GetBucketChoice(profileFlag)
	if err != nil {
		fmt.Printf("Error selecting bucket: %v\n", err)
		return
	}

	if bucket == "ALL_BUCKETS" {
		buckets, err := s3Access.ListS3Buckets(sess)
		if err != nil {
			fmt.Printf("Error retrieving buckets: %v\n", err)
			return
		}
		for _, b := range buckets {
			printFailedUploads(sess, b)
		}
	} else {
		printFailedUploads(sess, bucket)
	}
}

// menu: Entry point of the application.
// Provides a menu to the user to choose actions related to expiring S3 multipart uploads.
func menu() {
	for {
		choices := []string{
			"Select the profile and bucket to expire",
			"Select the profile and all buckets to expire",
			"Expire all profiles and all buckets",
			"Exit",
		}

		prompt := promptui.Select{
			Label: "Please choose an option",
			Items: choices,
		}

		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		switch result {
		case "Select the profile and bucket to expire":
			// Select Profile
			profile, err := GetProfileChoice()
			if err != nil {
				fmt.Printf("Error selecting profile: %v\n", err)
				continue // Takes the user back to the main menu
			}

			// Create session with profile
			credentials, credErr := s3Access.GetCredentialsForProfile(profileFlag)
			if credErr != nil {
				fmt.Printf("Error fetching credentials for profile %s: %v\n", profileFlag, credErr)
				return
			}
			// Use these credentials to create the session
			sess, err := s3Access.CreateAWSSessionWithCredentials(profileFlag, credentials) // Assuming you modify or create such a function in aws package
			if err != nil {
				fmt.Printf("Error creating session: %v\n", err)
				return
			}

			// Select Bucket
			bucket, err := GetBucketChoice(profile)
			if err != nil {
				fmt.Printf("Error selecting bucket: %v\n", err)
				continue // Takes the user back to the main menu
			}

			// Validation
			fmt.Printf("You chose profile: %s and bucket: %s\n", profile, bucket)

			// Purge in bucket
			err = s3Access.AbortFailedMultipartUploadsInBucket(sess, bucket)
			if err != nil {
				fmt.Printf("Error occurred expiring multipart uploads in bucket %v, %v\n", bucket, err)
				continue // Takes the user back to the main menu
			}

		case "Select the profile and all buckets to expire":
			profile, err := GetProfileChoice()
			if err != nil {
				fmt.Printf("Error selecting profile: %v\n", err)
				continue // Takes the user back to the main menu
			}
			fmt.Printf("You chose profile: %s. All buckets under this profile will be expired.\n", profile)
			// Further processing

		case "Expire all profiles and all buckets":
			// Further processing

		case "Exit":
			fmt.Println("Exiting...")
			return // Exits the for loop and the program
		}
	}
}

// GetProfileChoice: Retrieves all AWS profiles and presents a selection prompt to the user.
// return : (string, error) : The selected profile name and error if any.
func GetProfileChoice() (string, error) {
	profiles, err := config.RetrieveConfiguredProfiles()
	if err != nil {
		return "", err
	}

	prompt := promptui.Select{
		Label: "Please select a profile",
		Items: profiles,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

// GetBucketChoice: Retrieves all S3 buckets for a given AWS profile and presents a selection prompt to the user.
// profile : string : The selected AWS profile name.
// return : (string, error) : The selected S3 bucket name and error if any.
func GetBucketChoice(profile string) (string, error) {
	// Create a session using the selected profile
	session, err := config.EstablishConnectionUsingProfile(profile)
	if err != nil {
		return "", err
	}

	buckets, err := s3Access.ListS3Buckets(session)
	if err != nil {
		return "", err
	}

	prompt := promptui.Select{
		Label: "Please select a bucket",
		Items: buckets,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

func printFailedUploads(sess *session.Session, bucketName string) {
	// Call AbortFailedMultipartUploadsInBucket or another method to display the failed uploads
	// For now, it's a dummy print to demonstrate
	fmt.Println("Failed uploads for bucket:", bucketName)
}

// main initializes the CLI and handles command execution.
func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "--profile" || os.Args[1] == "-p") {
		fmt.Println("Error: --profile/-p flag can't be used alone. Use it with --discover/-d.")
		return
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
