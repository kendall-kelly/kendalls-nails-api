package utils

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize is 10MB in bytes
	MaxFileSize = 10 * 1024 * 1024
	// AllowedImageFormat is PNG
	AllowedImageFormat = ".png"
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
