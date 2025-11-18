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

	"github.com/disintegration/imaging"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Constants for file upload configuration
const (
	MaxUploadSize     = 10 << 20 // 10 MB
	UploadsBasePath   = "./uploads"
	PublicFileBaseURL = "https://hope.layu.cc/hope/static"
	ThumbnailWidth    = 800 // Width for thumbnails
	ThumbnailPrefix   = "thumb_"
)

// FileType represents supported upload file types
type FileType string

const (
	FileTypeAvatar     FileType = "avatar"
	FileTypeBackground FileType = "background"
	FileTypePost       FileType = "post"
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
		dst.Close()

		// Optimize the image (resize and compress)
		if err := optimizeImage(filePath, filePath, 1920); err != nil {
			fmt.Printf("Warning: Image optimization failed: %v\n", err)
			// Continue even if optimization fails
		}

		// Generate thumbnail
		thumbnailFilename := ThumbnailPrefix + newFilename
		thumbnailPath := filepath.Join(uploadDir, thumbnailFilename)
		if err := createThumbnail(filePath, thumbnailPath, ThumbnailWidth); err != nil {
			fmt.Printf("Warning: Thumbnail creation failed: %v\n", err)
			// Continue even if thumbnail creation fails
		}

		// Generate the public URLs
		fileURL := fmt.Sprintf("%s/%s", publicURLBase, newFilename)
		thumbnailURL := fmt.Sprintf("%s/%s", publicURLBase, thumbnailFilename)

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

		// Remove old files if they exist and are not external URLs
		if oldFileURL != "" && !strings.Contains(oldFileURL, "://") {
			cleanupOldFile(uploadDir, oldFileURL)
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

		// Return success response with the new file URLs
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: fmt.Sprintf("%s uploaded successfully", fileType),
			Data: map[string]string{
				"file_url":      fileURL,
				"thumbnail_url": thumbnailURL,
				"file_type":     string(fileType),
			},
		})
	}
}

// PostImageUploadHandler handles post image uploads (returns URLs without updating profile)
func PostImageUploadHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authenticated user ID
		userID, exists := c.Get("userID")
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

		// Set up upload directory for posts
		uploadDir := filepath.Join(UploadsBasePath, "posts")
		publicURLBase := fmt.Sprintf("%s/posts", PublicFileBaseURL)

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
		dst.Close()

		// Optimize the image (resize and compress)
		if err := optimizeImage(filePath, filePath, 1920); err != nil {
			fmt.Printf("Warning: Image optimization failed: %v\n", err)
		}

		// Generate thumbnail
		thumbnailFilename := ThumbnailPrefix + newFilename
		thumbnailPath := filepath.Join(uploadDir, thumbnailFilename)
		if err := createThumbnail(filePath, thumbnailPath, ThumbnailWidth); err != nil {
			fmt.Printf("Warning: Thumbnail creation failed: %v\n", err)
		}

		// Generate the public URLs
		fileURL := fmt.Sprintf("%s/%s", publicURLBase, newFilename)
		thumbnailURL := fmt.Sprintf("%s/%s", publicURLBase, thumbnailFilename)

		// Return success response with the URLs
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Post image uploaded successfully",
			Data: map[string]string{
				"file_url":      fileURL,
				"thumbnail_url": thumbnailURL,
				"file_type":     "post",
			},
		})
	}
}

// SetupStaticFileServer configures static file serving for uploaded files with caching
func SetupStaticFileServer(router *gin.Engine) {
	// Enable gzip compression globally
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// Create the base uploads directory if it doesn't exist
	if err := os.MkdirAll(UploadsBasePath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create uploads directory: %v\n", err)
	}

	// Create subdirectories for each file type
	fileTypes := []FileType{FileTypeAvatar, FileTypeBackground, FileTypePost}
	for _, fileType := range fileTypes {
		uploadDir := filepath.Join(UploadsBasePath, string(fileType)+"s")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create %s directory: %v\n", fileType, err)
		}

		// Serve static files for this type with cache headers
		staticPath := fmt.Sprintf("/hope/static/%ss", fileType)

		// Add cache control middleware for this path
		router.Use(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, staticPath) {
				// Cache for 1 year, files are immutable (UUID filenames)
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
				c.Header("Expires", time.Now().AddDate(1, 0, 0).Format(http.TimeFormat))
			}
			c.Next()
		})

		router.Static(staticPath, uploadDir)
	}
}

// optimizeImage resizes and compresses an image
func optimizeImage(srcPath, dstPath string, maxWidth int) error {
	// Open the source image
	img, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Resize if width exceeds maxWidth
	if width > maxWidth {
		img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
	}

	// Determine output format and save with compression
	ext := strings.ToLower(filepath.Ext(dstPath))
	switch ext {
	case ".jpg", ".jpeg":
		// Save as JPEG with quality 85
		err = imaging.Save(img, dstPath, imaging.JPEGQuality(85))
	case ".png":
		// Save as PNG with default compression
		err = imaging.Save(img, dstPath)
	case ".gif":
		// GIF - save as is (or convert to PNG for better compression)
		err = imaging.Save(img, dstPath)
	default:
		// Default to JPEG
		err = imaging.Save(img, dstPath, imaging.JPEGQuality(85))
	}

	if err != nil {
		return fmt.Errorf("failed to save optimized image: %w", err)
	}

	fmt.Printf("Optimized image: %s (%dx%d -> %dx%d)\n",
		filepath.Base(srcPath), width, height, img.Bounds().Dx(), img.Bounds().Dy())

	return nil
}

// createThumbnail creates a thumbnail version of an image
func createThumbnail(srcPath, dstPath string, maxWidth int) error {
	// Open the source image
	img, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open image for thumbnail: %w", err)
	}

	// Resize to thumbnail size maintaining aspect ratio
	thumbnail := imaging.Resize(img, maxWidth, 0, imaging.Lanczos)

	// Save thumbnail with compression
	ext := strings.ToLower(filepath.Ext(dstPath))
	switch ext {
	case ".jpg", ".jpeg":
		err = imaging.Save(thumbnail, dstPath, imaging.JPEGQuality(80))
	case ".png":
		err = imaging.Save(thumbnail, dstPath)
	case ".gif":
		err = imaging.Save(thumbnail, dstPath)
	default:
		err = imaging.Save(thumbnail, dstPath, imaging.JPEGQuality(80))
	}

	if err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	fmt.Printf("Created thumbnail: %s (%dx%d)\n",
		filepath.Base(dstPath), thumbnail.Bounds().Dx(), thumbnail.Bounds().Dy())

	return nil
}

// cleanupOldFile removes old file and its thumbnail
func cleanupOldFile(uploadDir, oldFileURL string) {
	// Extract filename from URL
	oldFilename := filepath.Base(oldFileURL)
	oldFilePath := filepath.Join(uploadDir, oldFilename)

	// Remove main file
	if err := os.Remove(oldFilePath); err != nil {
		fmt.Printf("Info: Could not remove old file %s: %v\n", oldFilePath, err)
	}

	// Remove thumbnail if exists
	thumbnailPath := filepath.Join(uploadDir, ThumbnailPrefix+oldFilename)
	if err := os.Remove(thumbnailPath); err != nil {
		// Ignore error - thumbnail might not exist
		fmt.Printf("Info: Could not remove old thumbnail %s: %v\n", thumbnailPath, err)
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
