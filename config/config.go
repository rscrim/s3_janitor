package config

import (
	"os"
	"path/filepath"
	"s3_mp_janitor/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"gopkg.in/ini.v1"
)

// ReadAWSConfigFile: Reads the AWS configuration file, typically located at `~/.aws/config`.
// return : ([]aws.ProfileConfig, error) : A list of parsed AWS profiles and an error if any.
func ReadAWSConfigFile() ([]aws.ProfileConfig, error) {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".aws", "config")
	cfg, err := ini.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	var profiles []aws.ProfileConfig
	for _, section := range cfg.Sections() {
		profile := aws.ProfileConfig{
			Name:   section.Name(),
			Region: section.Key("region").String(),
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// RetrieveConfiguredProfiles: Retrieves the profiles (or accounts) listed in the AWS configuration.
// return : []string : A list of profile names from the AWS configuration.
func RetrieveConfiguredProfiles() ([]string, error) {
	profiles, err := ReadAWSConfigFile()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(profiles))
	for i, profile := range profiles {
		names[i] = profile.Name
	}

	return names, nil
}

// EstablishConnectionUsingProfile: Uses a specified profile from the AWS configuration to establish a connection or session.
// profileName : string : The name of the AWS profile to use.
// return : (*session.Session, error) : The AWS session established for the profile and error if any.
func EstablishConnectionUsingProfile(profileName string) (*session.Session, error) {
	sessOpts := session.Options{
		Profile: profileName,
	}
	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return nil, err
	}
	return sess, nil
}
