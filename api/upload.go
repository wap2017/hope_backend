package api

import (
	"fmt"
	"hope_backend/dao"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Constants for file upload configuration
const (
	MaxUploadSize     = 10 << 20 // 10 MB
	UploadsBasePath   = "./uploads"
	PublicFileBaseURL = "http://hope.ioaths.com/hope/static"
)

// FileType represents supported upload file types
type FileType string

const (
	FileTypeAvatar     FileType = "avatar"
	FileTypeBackground FileType = "background"
)

// FileUploadHandler handles user file uploads (avatar, background, etc.)
func FileUploadHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authenticated user ID
		userID, exists := c.Get("userID")
		fmt.Printf("userID:%v exists%v\n", userID, exists)
		if !exists {
			c.JSON(http.StatusUnauthorized, Response{
				Success: false,
				Message: "Authentication required",
			})
			return
		}

		// Convert interface{} to int64
		id, ok := userID.(int64)
		if !ok {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Invalid user ID in authentication context",
			})
			return
		}

		// Get existing profile
		profile, err := profileDAO.GetByID(id)
		if err != nil {
			if err.Error() == "user profile not found" {
				c.JSON(http.StatusNotFound, Response{
					Success: false,
					Message: "User profile not found",
				})
			} else {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Message: "Failed to retrieve user profile: " + err.Error(),
				})
			}
			return
		}

		// Determine file type from request parameter
		fileTypeParam := c.Query("type")
		if fileTypeParam == "" {
			fileTypeParam = "avatar" // Default to avatar if not specified
		}

		var fileType FileType
		switch fileTypeParam {
		case "avatar":
			fileType = FileTypeAvatar
		case "background":
			fileType = FileTypeBackground
		default:
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid file type. Supported types: avatar, background",
			})
			return
		}

		// Set up upload directory based on file type
		uploadDir := filepath.Join(UploadsBasePath, string(fileType)+"s")
		publicURLBase := fmt.Sprintf("%s/%ss", PublicFileBaseURL, fileType)

		// Limit the upload size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)

		// Create uploads directory if it doesn't exist
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to create upload directory: " + err.Error(),
			})
			return
		}

		// Get the file from the request
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Error retrieving file: " + err.Error(),
			})
			return
		}
		defer file.Close()

		// Validate file type
		fileExt := strings.ToLower(filepath.Ext(header.Filename))
		if !isValidImageExt(fileExt) {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid file type. Allowed types: .jpg, .jpeg, .png, .gif",
			})
			return
		}

		// Generate a unique filename
		newFilename := fmt.Sprintf("%d-%s%s", id, uuid.New().String(), fileExt)
		filePath := filepath.Join(uploadDir, newFilename)

		// Create the file on the server
		dst, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to create file: " + err.Error(),
			})
			return
		}
		defer dst.Close()

		// Copy the file content
		if _, err = file.Seek(0, 0); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Error processing file: " + err.Error(),
			})
			return
		}

		// Copy file contents to destination
		if _, err = io.Copy(dst, file); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Error copying file: " + err.Error(),
			})
			return
		}

		// Generate the public URL for the file
		fileURL := fmt.Sprintf("%s/%s", publicURLBase, newFilename)

		// Update the appropriate field in user profile based on file type
		var oldFileURL string

		switch fileType {
		case FileTypeAvatar:
			oldFileURL = profile.UserAvatar
			profile.UserAvatar = fileURL
		case FileTypeBackground:
			oldFileURL = profile.ChatBackground
			profile.ChatBackground = fileURL
		}

		// Remove old file if it exists and is not an external URL
		if oldFileURL != "" && !strings.Contains(oldFileURL, "://") {
			// Extract filename from URL
			oldFilename := filepath.Base(oldFileURL)
			oldFilePath := filepath.Join(uploadDir, oldFilename)
			// Ignore errors here - just attempt to clean up
			os.Remove(oldFilePath)
		}

		// Update the user profile
		profile.UpdatedAt = time.Now().UnixMilli()
		if err := profileDAO.Update(profile); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to update profile with new file URL: " + err.Error(),
			})
			return
		}

		// Return success response with the new file URL
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: fmt.Sprintf("%s uploaded successfully", fileType),
			Data: map[string]string{
				"file_url":  fileURL,
				"file_type": string(fileType),
			},
		})
	}
}

// SetupStaticFileServer configures static file serving for uploaded files
func SetupStaticFileServer(router *gin.Engine) {
	// Create the base uploads directory if it doesn't exist
	if err := os.MkdirAll(UploadsBasePath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create uploads directory: %v\n", err)
	}

	// Create subdirectories for each file type
	fileTypes := []FileType{FileTypeAvatar, FileTypeBackground}
	for _, fileType := range fileTypes {
		uploadDir := filepath.Join(UploadsBasePath, string(fileType)+"s")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create %s directory: %v\n", fileType, err)
		}

		// Serve static files for this type
		staticPath := fmt.Sprintf("/hope/static/%ss", fileType)
		router.Static(staticPath, uploadDir)
	}
}

// isValidImageExt checks if the file extension is an allowed image type
func isValidImageExt(ext string) bool {
	validExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}
	return validExts[ext]
}
