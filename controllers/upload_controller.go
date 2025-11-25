package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/utils"
)

// GetUploadedImage handles GET /api/v1/uploads/:filename - serves uploaded PNG images
func GetUploadedImage(c *gin.Context) {
	filename := c.Param("filename")

	// Validate filename is not empty
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Filename is required",
			},
		})
		return
	}

	// Security: Prevent directory traversal attacks
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILENAME",
				"message": "Invalid filename",
			},
		})
		return
	}

	// Validate file extension is PNG
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILE_TYPE",
				"message": "Only PNG files are supported",
			},
		})
		return
	}

	// Construct full file path
	filePath := filepath.Join(utils.UploadDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FILE_NOT_FOUND",
				"message": "Image not found",
			},
		})
		return
	}

	// Serve the file with appropriate headers
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	c.File(filePath)
}
