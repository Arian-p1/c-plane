package handlers

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nextranet/gateway/c-plane/internal/context"
	"github.com/nextranet/gateway/c-plane/internal/logger"
	"github.com/nextranet/gateway/c-plane/internal/web/templates"
	"github.com/nextranet/gateway/c-plane/pkg/factory"
)

const (
	maxFileSize  = 100 * 1024 * 1024  // 100MB
	maxTotalSize = 1024 * 1024 * 1024 // 1GB
	uploadDir    = "uploads"
	allowedTypes = "firmware,config,backup,script,other"
)

// Files renders the files management page
func Files(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get files from storage
		files, err := getStoredFiles()
		if err != nil {
			logger.WebLog.Errorf("Failed to get files: %v", err)
			files = []templates.FileInfo{} // Empty list on error
		}

		// Calculate total size
		var totalSize int64
		for _, file := range files {
			totalSize += file.Size
		}

		// Get theme
		theme := c.GetString("theme")
		if theme == "" {
			theme = "dark"
		}

		// Prepare data for template
		data := templates.FilesPageData{
			BasePageData: templates.BasePageData{
				Title:       "Files",
				Theme:       theme,
				CurrentPath: "/files",
			},
			Files:     files,
			TotalSize: totalSize,
			Filters: templates.FileFilters{
				Type:   c.Query("type"),
				Search: c.Query("search"),
			},
		}

		// Render the files page
		component := templates.FilesPage(data)
		c.Header("Content-Type", "text/html; charset=utf-8")

		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			logger.WebLog.Errorf("Failed to render files page: %v", err)
			c.String(http.StatusInternalServerError, "Failed to render page")
			return
		}
	}
}

// UploadFiles handles file upload requests
func UploadFiles(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse multipart form
		err := c.Request.ParseMultipartForm(maxFileSize)
		if err != nil {
			logger.WebLog.Errorf("Failed to parse multipart form: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid file upload",
			})
			return
		}

		// Get form values
		fileType := c.PostForm("type")
		description := c.PostForm("description")

		// Validate file type
		if !isValidFileType(fileType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid file type",
			})
			return
		}

		// Get uploaded files
		form := c.Request.MultipartForm
		files := form.File["files"]

		if len(files) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No files provided",
			})
			return
		}

		// Check total size limit
		var totalSize int64
		for _, file := range files {
			totalSize += file.Size
		}

		if totalSize > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Total file size exceeds limit of %d MB", maxFileSize/(1024*1024)),
			})
			return
		}

		// Ensure upload directory exists
		uploadPath := getUploadPath()
		if err := os.MkdirAll(uploadPath, 0755); err != nil {
			logger.WebLog.Errorf("Failed to create upload directory: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create upload directory",
			})
			return
		}

		uploadedFiles := make([]templates.FileInfo, 0, len(files))

		// Process each file
		for _, file := range files {
			// Validate file
			if file.Size > maxFileSize {
				logger.WebLog.Warnf("File %s exceeds size limit", file.Filename)
				continue
			}

			// Open uploaded file
			src, err := file.Open()
			if err != nil {
				logger.WebLog.Errorf("Failed to open uploaded file %s: %v", file.Filename, err)
				continue
			}
			defer src.Close()

			// Generate unique filename
			timestamp := time.Now().Unix()
			filename := fmt.Sprintf("%d_%s", timestamp, sanitizeFilename(file.Filename))
			filePath := filepath.Join(uploadPath, filename)

			// Create destination file
			dst, err := os.Create(filePath)
			if err != nil {
				logger.WebLog.Errorf("Failed to create file %s: %v", filePath, err)
				continue
			}
			defer dst.Close()

			// Copy file content and calculate hash
			hash := md5.New()
			writer := io.MultiWriter(dst, hash)

			size, err := io.Copy(writer, src)
			if err != nil {
				logger.WebLog.Errorf("Failed to copy file content: %v", err)
				os.Remove(filePath) // Clean up on error
				continue
			}

			// Create file info
			fileInfo := templates.FileInfo{
				ID:          generateFileID(filename),
				Name:        file.Filename,
				Type:        fileType,
				Size:        size,
				Description: description,
				UploadedAt:  time.Now(),
				UploadedBy:  "admin", // TODO: Get from session/user context
				Hash:        fmt.Sprintf("%x", hash.Sum(nil)),
				MimeType:    file.Header.Get("Content-Type"),
			}

			// Save file metadata
			if err := saveFileMetadata(fileInfo, filename); err != nil {
				logger.WebLog.Errorf("Failed to save file metadata: %v", err)
				os.Remove(filePath) // Clean up on error
				continue
			}

			uploadedFiles = append(uploadedFiles, fileInfo)
			logger.WebLog.Infof("Successfully uploaded file: %s (%d bytes)", file.Filename, size)
		}

		if len(uploadedFiles) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to upload any files",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("Successfully uploaded %d files", len(uploadedFiles)),
			"files":   uploadedFiles,
		})
	}
}

// DownloadFile handles single file download requests
func DownloadFile(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("fileId")
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "File ID is required",
			})
			return
		}

		// Get file metadata
		fileInfo, err := getFileMetadata(fileID)
		if err != nil {
			logger.WebLog.Errorf("Failed to get file metadata: %v", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		// Find actual file
		filePath, err := findFileByID(fileID)
		if err != nil {
			logger.WebLog.Errorf("Failed to find file: %v", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			logger.WebLog.Errorf("File does not exist: %s", filePath)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		// Set download headers
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Name))
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Length", strconv.FormatInt(fileInfo.Size, 10))

		// Serve file
		c.File(filePath)
		logger.WebLog.Infof("File downloaded: %s", fileInfo.Name)
	}
}

// DownloadBulkFiles handles bulk file download requests
func DownloadBulkFiles(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get file IDs from form
		fileIDs := c.PostFormArray("fileIds")
		if len(fileIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No files specified",
			})
			return
		}

		// Create temporary zip file
		tempFile, err := os.CreateTemp("", "bulk_download_*.zip")
		if err != nil {
			logger.WebLog.Errorf("Failed to create temp file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create download",
			})
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// Create zip writer
		zipWriter := zip.NewWriter(tempFile)
		defer zipWriter.Close()

		addedFiles := 0

		// Add each file to zip
		for _, fileID := range fileIDs {
			fileInfo, err := getFileMetadata(fileID)
			if err != nil {
				logger.WebLog.Warnf("Failed to get metadata for file %s: %v", fileID, err)
				continue
			}

			filePath, err := findFileByID(fileID)
			if err != nil {
				logger.WebLog.Warnf("Failed to find file %s: %v", fileID, err)
				continue
			}

			// Open source file
			srcFile, err := os.Open(filePath)
			if err != nil {
				logger.WebLog.Warnf("Failed to open file %s: %v", filePath, err)
				continue
			}

			// Create file in zip
			zipFile, err := zipWriter.Create(fileInfo.Name)
			if err != nil {
				logger.WebLog.Warnf("Failed to create zip entry for %s: %v", fileInfo.Name, err)
				srcFile.Close()
				continue
			}

			// Copy file content
			_, err = io.Copy(zipFile, srcFile)
			srcFile.Close()

			if err != nil {
				logger.WebLog.Warnf("Failed to copy file content for %s: %v", fileInfo.Name, err)
				continue
			}

			addedFiles++
		}

		zipWriter.Close()

		if addedFiles == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "No files found",
			})
			return
		}

		// Set download headers
		filename := fmt.Sprintf("bulk_download_%d.zip", time.Now().Unix())
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "application/zip")

		// Get file size
		fileInfo, _ := tempFile.Stat()
		c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

		// Serve zip file
		tempFile.Seek(0, 0)
		io.Copy(c.Writer, tempFile)

		logger.WebLog.Infof("Bulk download created with %d files", addedFiles)
	}
}

// DeleteFile handles file deletion requests
func DeleteFile(appContext *context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("fileId")
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "File ID is required",
			})
			return
		}

		// Get file metadata
		fileInfo, err := getFileMetadata(fileID)
		if err != nil {
			logger.WebLog.Errorf("Failed to get file metadata: %v", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "File not found",
			})
			return
		}

		// Find and delete actual file
		filePath, err := findFileByID(fileID)
		if err == nil {
			if err := os.Remove(filePath); err != nil {
				logger.WebLog.Warnf("Failed to delete file %s: %v", filePath, err)
			}
		}

		// Delete metadata
		if err := deleteFileMetadata(fileID); err != nil {
			logger.WebLog.Errorf("Failed to delete file metadata: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to delete file",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("File '%s' deleted successfully", fileInfo.Name),
		})

		logger.WebLog.Infof("File deleted: %s", fileInfo.Name)
	}
}

// Helper functions

func getUploadPath() string {
	cfg := factory.GetConfig()
	if cfg.Web.UploadDir != "" {
		return cfg.Web.UploadDir
	}
	return uploadDir
}

func isValidFileType(fileType string) bool {
	validTypes := strings.Split(allowedTypes, ",")
	for _, validType := range validTypes {
		if fileType == validType {
			return true
		}
	}
	return false
}

func sanitizeFilename(filename string) string {
	// Remove any path separators and special characters
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, " ", "_")
	return filename
}

func generateFileID(filename string) string {
	// Simple file ID generation - in production use UUID
	return fmt.Sprintf("file_%d_%s", time.Now().UnixNano(), filename)
}

// File metadata operations (simplified - in production use proper database)
func saveFileMetadata(fileInfo templates.FileInfo, filename string) error {
	// TODO: Implement proper file metadata storage
	// For now, this is a placeholder
	return nil
}

func getFileMetadata(fileID string) (*templates.FileInfo, error) {
	// TODO: Implement proper file metadata retrieval
	// For now, create a dummy entry
	return &templates.FileInfo{
		ID:   fileID,
		Name: "example.txt",
		Type: "config",
		Size: 1024,
	}, nil
}

func deleteFileMetadata(fileID string) error {
	// TODO: Implement proper file metadata deletion
	return nil
}

func findFileByID(fileID string) (string, error) {
	// TODO: Implement proper file lookup
	// For now, return a dummy path
	uploadPath := getUploadPath()
	return filepath.Join(uploadPath, "example.txt"), nil
}

func getStoredFiles() ([]templates.FileInfo, error) {
	// TODO: Implement proper file listing from metadata storage
	// For now, return dummy data
	files := []templates.FileInfo{
		{
			ID:          "file_1",
			Name:        "firmware_v2.1.0.bin",
			Type:        "firmware",
			Size:        5242880, // 5MB
			Description: "Latest firmware update",
			UploadedAt:  time.Now().Add(-24 * time.Hour),
			UploadedBy:  "admin",
		},
		{
			ID:          "file_2",
			Name:        "config_backup.xml",
			Type:        "backup",
			Size:        102400, // 100KB
			Description: "Configuration backup",
			UploadedAt:  time.Now().Add(-2 * time.Hour),
			UploadedBy:  "admin",
		},
	}
	return files, nil
}
