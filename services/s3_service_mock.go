package services

import (
	"fmt"
	"mime/multipart"
	"sync"
)

// MockS3Service is a mock implementation of S3Service for testing
type MockS3Service struct {
	uploadedFiles map[string][]byte // map of S3 key to file content
	mu            sync.RWMutex
}

// NewMockS3Service creates a new mock S3 service
func NewMockS3Service() *MockS3Service {
	return &MockS3Service{
		uploadedFiles: make(map[string][]byte),
	}
}

// SetAsMockForTesting sets this mock as the global S3 service instance for testing
func (m *MockS3Service) SetAsMockForTesting() {
	SetS3Service(m)
}

// UploadFile simulates uploading a file to S3
func (m *MockS3Service) UploadFile(fileHeader *multipart.FileHeader) (string, error) {
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

	// Generate mock S3 key
	s3Key := fmt.Sprintf("uploads/mock_%s", fileHeader.Filename)

	// Store in mock storage
	m.mu.Lock()
	m.uploadedFiles[s3Key] = content
	m.mu.Unlock()

	return s3Key, nil
}

// GetPresignedURL simulates generating a presigned URL
func (m *MockS3Service) GetPresignedURL(s3Key string) (string, error) {
	if s3Key == "" {
		return "", nil
	}

	// Check if file exists in mock storage
	m.mu.RLock()
	_, exists := m.uploadedFiles[s3Key]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("file not found in mock S3: %s", s3Key)
	}

	// Return a mock presigned URL
	return fmt.Sprintf("https://test-bucket.s3.us-east-1.amazonaws.com/%s?mock=true", s3Key), nil
}

// DeleteFile simulates deleting a file from S3
func (m *MockS3Service) DeleteFile(s3Key string) error {
	if s3Key == "" {
		return nil
	}

	m.mu.Lock()
	delete(m.uploadedFiles, s3Key)
	m.mu.Unlock()

	return nil
}

// GetUploadedFiles returns all uploaded files (for testing assertions)
func (m *MockS3Service) GetUploadedFiles() map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent race conditions
	files := make(map[string][]byte, len(m.uploadedFiles))
	for k, v := range m.uploadedFiles {
		files[k] = v
	}
	return files
}

// FileExists checks if a file exists in mock storage
func (m *MockS3Service) FileExists(s3Key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.uploadedFiles[s3Key]
	return exists
}

// Clear removes all files from mock storage
func (m *MockS3Service) Clear() {
	m.mu.Lock()
	m.uploadedFiles = make(map[string][]byte)
	m.mu.Unlock()
}
