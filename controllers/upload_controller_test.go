package controllers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUploadedImage_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary upload directory
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir

	// Create a test PNG file
	testContent := []byte("fake PNG content")
	testFilename := "test_image.png"
	testPath := filepath.Join(tmpDir, testFilename)
	err := os.WriteFile(testPath, testContent, 0644)
	require.NoError(t, err)

	// Setup router and request
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	req := httptest.NewRequest("GET", "/uploads/"+testFilename, nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=86400", w.Header().Get("Cache-Control"))
	assert.Equal(t, testContent, w.Body.Bytes())
}

func TestGetUploadedImage_FileNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary upload directory
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir

	// Setup router and request
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	req := httptest.NewRequest("GET", "/uploads/nonexistent.png", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "FILE_NOT_FOUND")
	assert.Contains(t, w.Body.String(), "Image not found")
}

func TestGetUploadedImage_EmptyFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router and request
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	req := httptest.NewRequest("GET", "/uploads/", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response - Gin will handle this as a 404 because route doesn't match
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUploadedImage_DirectoryTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary upload directory
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir

	// Setup router
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	testCases := []struct {
		name           string
		filename       string
		expectedStatus int
		expectedError  string
	}{
		// Gin's router prevents path traversal by treating slashes as path separators
		// So these URLs won't match our route and get 404
		{"Parent directory traversal", "../../../etc/passwd", http.StatusNotFound, ""},
		{"Forward slash in filename", "path/to/file.png", http.StatusNotFound, ""},

		// URL-encoded slashes and backslashes within a single path param are caught by our validation
		{"Backslash in filename", "path\\to\\file.png", http.StatusBadRequest, "INVALID_FILENAME"},
		{"Dots in filename", "..file.png", http.StatusBadRequest, "INVALID_FILENAME"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/uploads/"+tc.filename, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedError != "" {
				assert.Contains(t, w.Body.String(), tc.expectedError)
			}
		})
	}
}

func TestGetUploadedImage_InvalidFileType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	testCases := []struct {
		name     string
		filename string
	}{
		{"JPEG file", "image.jpg"},
		{"JPEG file uppercase", "image.JPEG"},
		{"GIF file", "image.gif"},
		{"No extension", "image"},
		{"Text file", "document.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/uploads/"+tc.filename, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "INVALID_FILE_TYPE")
			assert.Contains(t, w.Body.String(), "Only PNG files are supported")
		})
	}
}

func TestGetUploadedImage_CaseInsensitivePNG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary upload directory
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir

	// Create a test PNG file with uppercase extension
	testContent := []byte("fake PNG content")
	testFilename := "test_image.PNG"
	testPath := filepath.Join(tmpDir, testFilename)
	err := os.WriteFile(testPath, testContent, 0644)
	require.NoError(t, err)

	// Setup router and request
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	req := httptest.NewRequest("GET", "/uploads/"+testFilename, nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Verify response - should accept .PNG (case insensitive)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, testContent, w.Body.Bytes())
}

func TestGetUploadedImage_MultipleFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create temporary upload directory
	tmpDir := t.TempDir()
	utils.UploadDir = tmpDir

	// Create multiple test PNG files
	files := map[string][]byte{
		"image1.png": []byte("first image content"),
		"image2.png": []byte("second image content"),
		"image3.png": []byte("third image content"),
	}

	for filename, content := range files {
		testPath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(testPath, content, 0644)
		require.NoError(t, err)
	}

	// Setup router
	router := gin.New()
	router.GET("/uploads/:filename", GetUploadedImage)

	// Verify each file can be retrieved independently
	for filename, expectedContent := range files {
		req := httptest.NewRequest("GET", "/uploads/"+filename, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, expectedContent, w.Body.Bytes())
	}
}
