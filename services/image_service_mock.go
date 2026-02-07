package services

import (
	"fmt"
	"mime/multipart"
	"sync"

	"github.com/kendall-kelly/kendalls-nails-api/utils"
)

// MockImageService is a mock implementation of ImageService for testing
type MockImageService struct {
	uploadedImages map[string][]byte // map of image key to file content
	mu             sync.RWMutex
}

// NewMockImageService creates a new mock image service
func NewMockImageService() *MockImageService {
	return &MockImageService{
		uploadedImages: make(map[string][]byte),
	}
}

// SetAsMockForTesting sets this mock as the global image service instance for testing
func (m *MockImageService) SetAsMockForTesting() {
	SetImageService(m)
}

// UploadImage simulates uploading an image
func (m *MockImageService) UploadImage(fileHeader *multipart.FileHeader) (string, error) {
	// Validate the image file
	if err := utils.ValidateImageFile(fileHeader); err != nil {
		return "", err
	}

	// Open and read the file
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	content := make([]byte, fileHeader.Size)
	_, err = file.Read(content)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Generate mock image key
	imageKey := fmt.Sprintf("uploads/mock_%s", fileHeader.Filename)

	// Store in mock storage
	m.mu.Lock()
	m.uploadedImages[imageKey] = content
	m.mu.Unlock()

	return imageKey, nil
}

// GetImageURL simulates generating a URL for an image
func (m *MockImageService) GetImageURL(imageKey string) (string, error) {
	if imageKey == "" {
		return "", nil
	}

	// Check if image exists in mock storage
	m.mu.RLock()
	_, exists := m.uploadedImages[imageKey]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("image not found in mock storage: %s", imageKey)
	}

	// Return a mock URL
	return fmt.Sprintf("https://test-bucket.s3.us-east-1.amazonaws.com/%s?mock=true", imageKey), nil
}

// DeleteImage simulates deleting an image
func (m *MockImageService) DeleteImage(imageKey string) error {
	if imageKey == "" {
		return nil
	}

	m.mu.Lock()
	delete(m.uploadedImages, imageKey)
	m.mu.Unlock()

	return nil
}

// GetUploadedImages returns all uploaded images (for testing assertions)
func (m *MockImageService) GetUploadedImages() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	images := make(map[string][]byte, len(m.uploadedImages))
	for k, v := range m.uploadedImages {
		images[k] = v
	}
	return images
}

// ImageExists checks if an image exists in mock storage
func (m *MockImageService) ImageExists(imageKey string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.uploadedImages[imageKey]
	return exists
}

// Clear removes all images from mock storage
func (m *MockImageService) Clear() {
	m.mu.Lock()
	m.uploadedImages = make(map[string][]byte)
	m.mu.Unlock()
}
