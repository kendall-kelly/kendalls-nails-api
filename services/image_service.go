package services

import (
	"fmt"
	"mime/multipart"

	"github.com/kendall-kelly/kendalls-nails-api/utils"
)

// ImageService handles all image-related operations including upload, retrieval, and deletion
type ImageService interface {
	// UploadImage validates and uploads an image file, returns the storage key
	UploadImage(fileHeader *multipart.FileHeader) (string, error)

	// GetImageURL generates a URL for accessing an uploaded image
	GetImageURL(imageKey string) (string, error)

	// DeleteImage removes an image from storage
	DeleteImage(imageKey string) error
}

// S3ImageService implements ImageService using AWS S3 for storage
type S3ImageService struct {
	s3Service S3Interface
}

var imageServiceInstance ImageService

// InitImageService initializes the image service with S3 backend
func InitImageService(s3Service S3Interface) ImageService {
	imageServiceInstance = &S3ImageService{
		s3Service: s3Service,
	}
	return imageServiceInstance
}

// GetImageService returns the initialized image service instance
func GetImageService() ImageService {
	return imageServiceInstance
}

// SetImageService sets the image service instance (primarily for testing)
func SetImageService(service ImageService) {
	imageServiceInstance = service
}

// UploadImage validates and uploads an image file to S3
func (s *S3ImageService) UploadImage(fileHeader *multipart.FileHeader) (string, error) {
	// Validate the image file
	if err := utils.ValidateImageFile(fileHeader); err != nil {
		return "", err
	}

	// Upload to S3
	s3Key, err := s.s3Service.UploadFile(fileHeader)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return s3Key, nil
}

// GetImageURL generates a presigned URL for accessing an image
func (s *S3ImageService) GetImageURL(imageKey string) (string, error) {
	if imageKey == "" {
		return "", nil
	}

	url, err := s.s3Service.GetPresignedURL(imageKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate image URL: %w", err)
	}

	return url, nil
}

// DeleteImage deletes an image from S3
func (s *S3ImageService) DeleteImage(imageKey string) error {
	if imageKey == "" {
		return nil
	}

	if err := s.s3Service.DeleteFile(imageKey); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}
