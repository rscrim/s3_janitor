package s3Access

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// LoadAWSConfigForProfile loads the AWS configuration for the given profile from the local AWS config and credentials files.
func LoadAWSConfigForProfile(ctx context.Context, profile string) (aws.Config, error) {
	if profile == "" {
		profile = "default"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config for profile %s: %v", profile, err)
	}

	return cfg, nil
}

// GetCredentialsForProfile retrieves the AWS credentials for the given profile from the loaded AWS configuration.
func GetCredentialsForProfile(ctx context.Context, profile string) (*aws.Credentials, error) {
	cfg, err := LoadAWSConfigForProfile(ctx, profile)
	if err != nil {
		return nil, err
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials for profile %s: %v", profile, err)
	}

	return &creds, nil
}

// ListS3Buckets: Lists all the S3 buckets associated with the AWS account of the given session.
// sess : *session.Session : The active AWS session.
// return : ([]string, error) : A list of all S3 bucket names and error if any.
func ListS3Buckets(sess *session.Session) ([]string, error) {
	s3Svc := s3.New(sess, &aws.Config{Region: aws.String("ap-southeast-2")})

	result, err := s3Svc.ListBuckets(nil)
	if err != nil {
		return nil, err
	}

	bucketNames := make([]string, len(result.Buckets))
	for i, b := range result.Buckets {
		bucketNames[i] = *b.Name
	}

	return bucketNames, nil
}

// AbortFailedMultipartUploadsInAllBuckets: Aborts or deletes any failed multipart uploads in all S3 buckets associated with the AWS account.
// sess : *session.Session : The active AWS session.
// return : error : Any error that occurred during the process.
func AbortFailedMultipartUploadsInAllBuckets(sess *session.Session) error {
	s3Svc := s3.New(sess)

	// List all buckets
	buckets, err := s3Svc.ListBuckets(nil)
	if err != nil {
		return fmt.Errorf("error listing buckets: %v", err)
	}

	// Loop over each bucket and abort in-progress multipart uploads
	for _, bucket := range buckets.Buckets {
		err := AbortFailedMultipartUploadsInBucket(sess, *bucket.Name)
		if err != nil {
			return fmt.Errorf("error aborting multipart uploads in bucket %s: %v", *bucket.Name, err)
		}
	}

	return nil
}

// AbortFailedMultipartUploadsInBucket: Aborts or deletes any failed multipart uploads in a given S3 bucket.
// sess : *session.Session : The active AWS session.
// bucketName : string : The name of the S3 bucket to process.
// return : error : Any error that occurred during the process.
func AbortFailedMultipartUploadsInBucket(sess *session.Session, bucketName string) error {
	s3Svc := s3.New(sess)

	// List all in-progress multipart uploads in the given bucket
	uploads, err := s3Svc.ListMultipartUploads(&s3.ListMultipartUploadsInput{
		Bucket: &bucketName,
	})

	if err != nil {
		return fmt.Errorf("error listing multipart uploads for bucket %s: %v", bucketName, err)
	}

	// Loop over each multipart upload and abort them
	for _, upload := range uploads.Uploads {
		_, err := s3Svc.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   &bucketName,
			Key:      upload.Key,
			UploadId: upload.UploadId,
		})

		if err != nil {
			return fmt.Errorf("error aborting multipart upload %s for key %s: %v", *upload.UploadId, *upload.Key, err)
		}
	}

	return nil
}
