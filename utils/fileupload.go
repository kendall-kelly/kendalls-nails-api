package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize is 10MB in bytes
	MaxFileSize = 10 * 1024 * 1024
	// AllowedImageFormat is PNG
	AllowedImageFormat = ".png"
)

var (
	// UploadDir is the directory where uploaded files are stored
	// Can be overridden for testing
	UploadDir = "./uploads"
)

// FileUploadError represents a file upload validation error
type FileUploadError struct {
	Code    string
	Message string
}

func (e *FileUploadError) Error() string {
	return e.Message
}

// ValidateImageFile validates the uploaded file format and size
func ValidateImageFile(fileHeader *multipart.FileHeader) error {
	// Check file size
	if fileHeader.Size > MaxFileSize {
		return &FileUploadError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("File size exceeds maximum allowed size of %d MB", MaxFileSize/(1024*1024)),
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != AllowedImageFormat {
		return &FileUploadError{
			Code:    "INVALID_FILE_FORMAT",
			Message: fmt.Sprintf("Only %s files are allowed", AllowedImageFormat),
		}
	}

	return nil
}

// SaveUploadedFile saves the uploaded file to the local filesystem
// Returns the relative path to the saved file
func SaveUploadedFile(fileHeader *multipart.FileHeader, uploadDir string) (filename string, err error) {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename to prevent collisions
	filename = fmt.Sprintf("%d_%s",
		fileHeader.Size,
		filepath.Base(fileHeader.Filename))

	// Full path to save the file
	fullPath := filepath.Join(uploadDir, filename)

	// Open the uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			// Log error since we're reading; not critical enough to fail the operation
			fmt.Printf("warning: failed to close source file: %v\n", closeErr)
		}
	}()

	// Create the destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close destination file: %w", closeErr)
		}
	}()

	// Copy the file
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return relative path (from uploadDir)
	return filename, nil
}

// GetImageURL returns the URL path for accessing the uploaded image
func GetImageURL(filename string) string {
	if filename == "" {
		return ""
	}
	return fmt.Sprintf("/api/v1/uploads/%s", filename)
}
