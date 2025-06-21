package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"hope_backend/dao"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PostRequest represents the request body for creating a post
type PostRequest struct {
	Content string `json:"content" binding:"required"`
}

// CommentRequest represents the request body for creating a comment
type CommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ParentID *int64 `json:"parent_id"`
}

const (
	defaultPageSize = 10
	maxPageSize     = 50
	uploadDir       = "uploads/posts"
)

// Initialize the upload directory
func init() {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Failed to create upload directory: %v\n", err)
	}
}

// CreatePostHandler handles POST requests to create a new post
func CreatePostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Parse form data
		if err := c.Request.ParseMultipartForm(100 << 20); err != nil { // 32MB max
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Failed to parse form data: " + err.Error(),
			})
			return
		}

		// Get content from form
		content := c.Request.FormValue("content")
		if content == "" {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Content is required",
			})
			return
		}

		// Get images (up to 9)
		form, _ := c.MultipartForm()
		files := form.File["images"]

		// Check image count
		if len(files) > 9 {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Maximum of 9 images allowed",
			})
			return
		}

		imagePaths := make([]string, 0, len(files))

		// Process and save each image
		for _, file := range files {
			// Generate unique filename
			ext := filepath.Ext(file.Filename)
			uniqueID := uuid.New().String()
			newFilename := uniqueID + ext
			filePath := filepath.Join(uploadDir, newFilename)
			relativePath := filepath.Join("posts", newFilename) // Store relative path in DB

			// Save the file
			if err := c.SaveUploadedFile(file, filePath); err != nil {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Message: "Failed to save image: " + err.Error(),
				})
				return
			}

			imagePaths = append(imagePaths, relativePath)
		}

		// Create the post
		post := &dao.Post{
			UserID:  userID.(int64),
			Content: content,
		}

		postID, err := postDAO.Create(post, imagePaths)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to create post: " + err.Error(),
			})
			return
		}

		// Get the created post with images
		createdPost, err := postDAO.GetByID(postID, userID.(int64))
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Post created but failed to retrieve details",
			})
			return
		}

		c.JSON(http.StatusCreated, Response{
			Success: true,
			Message: "Post created successfully",
			Data:    createdPost,
		})
	}
}

// GetPostHandler handles GET requests to retrieve a post by ID
func GetPostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Get post ID from URL parameter
		postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Get post with images
		post, err := postDAO.GetByID(postID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "post not found" {
				status = http.StatusNotFound
			}
			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    post,
		})
	}
}

// ListPostsHandler handles GET requests to list posts with pagination
func ListPostsHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Parse pagination parameters
		page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defaultPageSize)))
		if err != nil || pageSize < 1 {
			pageSize = defaultPageSize
		}
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}

		// Parse filter by user ID
		filterUserID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)

		// Get posts with pagination
		posts, total, err := postDAO.ListPosts(page, pageSize, filterUserID, userID.(int64))
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to retrieve posts: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    posts,
			Total:   total,
			Page:    page,
			Size:    pageSize,
		})
	}
}

// UpdatePostHandler handles PUT requests to update a post
func UpdatePostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Get post ID from URL parameter
		postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Get the existing post
		post, err := postDAO.GetByID(postID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "post not found" {
				status = http.StatusNotFound
			}
			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Check if user is the owner of the post
		if post.UserID != userID.(int64) {
			c.JSON(http.StatusForbidden, Response{
				Success: false,
				Message: "You do not have permission to update this post",
			})
			return
		}

		// Bind request body
		var req PostRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Update post content
		post.Content = req.Content

		if err := postDAO.Update(post); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to update post: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Post updated successfully",
			Data:    post,
		})
	}
}

// DeletePostHandler handles DELETE requests to delete a post
func DeletePostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Get post ID from URL parameter
		postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Get the post to check ownership
		post, err := postDAO.GetByID(postID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "post not found" {
				status = http.StatusNotFound
			}
			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Check if user is the owner of the post
		if post.UserID != userID.(int64) {
			c.JSON(http.StatusForbidden, Response{
				Success: false,
				Message: "You do not have permission to delete this post",
			})
			return
		}

		// Delete images from filesystem
		for _, image := range post.Images {
			// Convert DB path to filesystem path
			filePath := filepath.Join("uploads", image.ImagePath)
			// Attempt to delete file, but don't fail if unsuccessful
			_ = os.Remove(filePath)
		}

		// Delete the post and all related data
		if err := postDAO.Delete(postID); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to delete post: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Post deleted successfully",
		})
	}
}

// LikePostHandler handles POST requests to like a post
func LikePostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Get post ID from URL parameter
		postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Add like
		err = postDAO.LikePost(postID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError

			// Handle specific errors
			if err.Error() == "post not found" {
				status = http.StatusNotFound
			} else if err.Error() == "post already liked by user" {
				status = http.StatusBadRequest
			}

			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Post liked successfully",
		})
	}
}

// UnlikePostHandler handles POST requests to unlike a post
func UnlikePostHandler(postDAO *dao.PostDAO) gin.HandlerFunc {
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

		// Get post ID from URL parameter
		postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Remove like
		err = postDAO.UnlikePost(postID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError

			// Handle specific errors
			if err.Error() == "post not found" {
				status = http.StatusNotFound
			} else if err.Error() == "post not liked by user" {
				status = http.StatusBadRequest
			}

			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Post unliked successfully",
		})
	}
}
