package utils

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestFileHeader creates a mock multipart.FileHeader for testing
func createTestFileHeader(filename string, size int64, content []byte) *multipart.FileHeader {
	// Create a buffer to write our multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create form file
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/png")
	part, _ := writer.CreatePart(h)
	part.Write(content)
	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(int64(len(content)) + 1024)
	defer form.RemoveAll()

	if len(form.File["file"]) > 0 {
		fileHeader := form.File["file"][0]
		// Override size for testing purposes
		fileHeader.Size = size
		return fileHeader
	}

	return nil
}

func TestValidateImageFile_Success(t *testing.T) {
	// Test with valid PNG file under size limit
	content := []byte("fake png content")
	fileHeader := createTestFileHeader("test.png", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.NoError(t, err)
}

func TestValidateImageFile_FileTooLarge(t *testing.T) {
	// Test with file exceeding size limit (11MB)
	content := []byte("fake png content")
	fileHeader := createTestFileHeader("large.png", 11*1024*1024, content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.Error(t, err)

	fileErr, ok := err.(*FileUploadError)
	require.True(t, ok, "Error should be of type FileUploadError")
	assert.Equal(t, "FILE_TOO_LARGE", fileErr.Code)
	assert.Contains(t, fileErr.Message, "File size exceeds maximum allowed size")
}

func TestValidateImageFile_InvalidFormat_JPG(t *testing.T) {
	// Test with JPG file (not allowed)
	content := []byte("fake jpg content")
	fileHeader := createTestFileHeader("test.jpg", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.Error(t, err)

	fileErr, ok := err.(*FileUploadError)
	require.True(t, ok, "Error should be of type FileUploadError")
	assert.Equal(t, "INVALID_FILE_FORMAT", fileErr.Code)
	assert.Contains(t, fileErr.Message, "Only .png files are allowed")
}

func TestValidateImageFile_InvalidFormat_JPEG(t *testing.T) {
	// Test with JPEG file (not allowed)
	content := []byte("fake jpeg content")
	fileHeader := createTestFileHeader("test.jpeg", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.Error(t, err)

	fileErr, ok := err.(*FileUploadError)
	require.True(t, ok, "Error should be of type FileUploadError")
	assert.Equal(t, "INVALID_FILE_FORMAT", fileErr.Code)
	assert.Contains(t, fileErr.Message, "Only .png files are allowed")
}

func TestValidateImageFile_InvalidFormat_GIF(t *testing.T) {
	// Test with GIF file (not allowed)
	content := []byte("fake gif content")
	fileHeader := createTestFileHeader("test.gif", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.Error(t, err)

	fileErr, ok := err.(*FileUploadError)
	require.True(t, ok, "Error should be of type FileUploadError")
	assert.Equal(t, "INVALID_FILE_FORMAT", fileErr.Code)
}

func TestValidateImageFile_InvalidFormat_NoExtension(t *testing.T) {
	// Test with file without extension
	content := []byte("fake content")
	fileHeader := createTestFileHeader("testfile", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.Error(t, err)

	fileErr, ok := err.(*FileUploadError)
	require.True(t, ok, "Error should be of type FileUploadError")
	assert.Equal(t, "INVALID_FILE_FORMAT", fileErr.Code)
}

func TestValidateImageFile_CaseInsensitive(t *testing.T) {
	// Test with uppercase extension
	content := []byte("fake png content")
	fileHeader := createTestFileHeader("test.PNG", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	err := ValidateImageFile(fileHeader)
	assert.NoError(t, err, "Validation should be case-insensitive")
}

func TestFileUploadError_Error(t *testing.T) {
	err := &FileUploadError{
		Code:    "TEST_CODE",
		Message: "Test error message",
	}

	assert.Equal(t, "Test error message", err.Error())
}
