package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appConfig "github.com/kendall-kelly/kendalls-nails-api/config"
)

// S3Interface defines the interface for S3 operations
type S3Interface interface {
	UploadFile(fileHeader *multipart.FileHeader) (string, error)
	GetPresignedURL(s3Key string) (string, error)
	DeleteFile(s3Key string) error
}

// S3Service handles all S3-related operations
type S3Service struct {
	client *s3.Client
	bucket string
}

var s3ServiceInstance S3Interface

// InitS3Service initializes the S3 service with AWS credentials
func InitS3Service() (S3Interface, error) {
	cfg := appConfig.GetConfig()

	// Load AWS configuration with explicit options
	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with explicit options to ensure SigV4
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		// Force the use of path-style addressing if needed
		// This can sometimes help with signature issues
		o.UsePathStyle = false
	})

	s3ServiceInstance = &S3Service{
		client: client,
		bucket: cfg.AWSS3Bucket,
	}

	return s3ServiceInstance, nil
}

// GetS3Service returns the initialized S3 service instance
func GetS3Service() S3Interface {
	return s3ServiceInstance
}

// SetS3Service sets the S3 service instance (primarily for testing)
func SetS3Service(service S3Interface) {
	s3ServiceInstance = service
}

// UploadFile uploads a file to S3 and returns the S3 key
func (s *S3Service) UploadFile(fileHeader *multipart.FileHeader) (string, error) {
	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("warning: failed to close file: %v", closeErr)
		}
	}()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate unique S3 key (path in bucket)
	// Format: uploads/{timestamp}_{filename}
	timestamp := time.Now().Unix()
	filename := filepath.Base(fileHeader.Filename)
	s3Key := fmt.Sprintf("uploads/%d_%s", timestamp, filename)

	// Determine content type
	contentType := "image/png" // Since we only allow PNG files

	// Upload to S3 with proper settings
	_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s3Key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
		// Note: ACL is not set here - bucket permissions should handle access
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return s3Key, nil
}

// GetPresignedURL generates a presigned URL for accessing a private S3 object
// The URL expires after 1 hour
func (s *S3Service) GetPresignedURL(s3Key string) (string, error) {
	if s3Key == "" {
		return "", nil
	}

	// Create a presign client
	presignClient := s3.NewPresignClient(s.client)

	// Generate presigned URL valid for 1 hour using PresignGetObject
	ctx := context.TODO()
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	log.Printf("Generated presigned URL for key %s", s3Key)
	return request.URL, nil
}

// DeleteFile deletes a file from S3
func (s *S3Service) DeleteFile(s3Key string) error {
	if s3Key == "" {
		return nil
	}

	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}
