package utils

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
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

func TestSaveUploadedFile_Success(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create test file
	content := []byte("fake png content for testing")
	fileHeader := createTestFileHeader("test.png", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	// Save the file
	filename, err := SaveUploadedFile(fileHeader, tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, filename)

	// Verify file was created
	fullPath := filepath.Join(tmpDir, filename)
	assert.FileExists(t, fullPath)

	// Verify file content
	savedContent, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, content, savedContent)
}

func TestSaveUploadedFile_CreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	uploadDir := filepath.Join(tmpDir, "uploads", "nested")

	// Directory should not exist yet
	_, err := os.Stat(uploadDir)
	assert.True(t, os.IsNotExist(err))

	// Create test file
	content := []byte("test content")
	fileHeader := createTestFileHeader("test.png", int64(len(content)), content)
	require.NotNil(t, fileHeader)

	// Save the file (should create directory)
	filename, err := SaveUploadedFile(fileHeader, uploadDir)
	require.NoError(t, err)
	assert.NotEmpty(t, filename)

	// Verify directory was created
	_, err = os.Stat(uploadDir)
	assert.NoError(t, err)

	// Verify file was created
	fullPath := filepath.Join(uploadDir, filename)
	assert.FileExists(t, fullPath)
}

func TestSaveUploadedFile_UniqueFilenames(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Save same filename twice - should generate different filenames based on size
	content1 := []byte("content1")
	fileHeader1 := createTestFileHeader("test.png", int64(len(content1)), content1)
	require.NotNil(t, fileHeader1)

	content2 := []byte("content2different")
	fileHeader2 := createTestFileHeader("test.png", int64(len(content2)), content2)
	require.NotNil(t, fileHeader2)

	filename1, err := SaveUploadedFile(fileHeader1, tmpDir)
	require.NoError(t, err)

	filename2, err := SaveUploadedFile(fileHeader2, tmpDir)
	require.NoError(t, err)

	// Filenames should be different due to different sizes
	assert.NotEqual(t, filename1, filename2)
}

func TestGetImageURL_WithFilename(t *testing.T) {
	url := GetImageURL("test123.png")
	assert.Equal(t, "/api/v1/uploads/test123.png", url)
}

func TestGetImageURL_EmptyFilename(t *testing.T) {
	url := GetImageURL("")
	assert.Equal(t, "", url)
}

func TestFileUploadError_Error(t *testing.T) {
	err := &FileUploadError{
		Code:    "TEST_CODE",
		Message: "Test error message",
	}

	assert.Equal(t, "Test error message", err.Error())
}
