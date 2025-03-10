package api

import (
	"net/http"
	"strconv"

	"hope_backend/dao"

	"github.com/gin-gonic/gin"
)

// CreateCommentHandler handles POST requests to create a new comment
func CreateCommentHandler(commentDAO *dao.CommentDAO) gin.HandlerFunc {
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
		postID, err := strconv.ParseInt(c.Param("postId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
			})
			return
		}

		// Bind request body
		var req CommentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Create the comment
		comment := &dao.Comment{
			PostID:   postID,
			UserID:   userID.(int64),
			ParentID: req.ParentID,
			Content:  req.Content,
		}

		commentID, err := commentDAO.Create(comment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to create comment: " + err.Error(),
			})
			return
		}

		// Get the created comment
		createdComment, err := commentDAO.GetByID(commentID, userID.(int64))
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Comment created but failed to retrieve details",
			})
			return
		}

		c.JSON(http.StatusCreated, Response{
			Success: true,
			Message: "Comment created successfully",
			Data:    createdComment,
		})
	}
}

// ListCommentsHandler handles GET requests to list comments for a post
func ListCommentsHandler(commentDAO *dao.CommentDAO) gin.HandlerFunc {
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
		postID, err := strconv.ParseInt(c.Param("postId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid post ID format",
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

		// Get comments with pagination
		comments, total, err := commentDAO.ListComments(postID, page, pageSize, userID.(int64))
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to retrieve comments: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    comments,
			Total:   total,
			Page:    page,
			Size:    pageSize,
		})
	}
}

// DeleteCommentHandler handles DELETE requests to delete a comment
func DeleteCommentHandler(commentDAO *dao.CommentDAO) gin.HandlerFunc {
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

		// Get comment ID from URL parameter
		commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid comment ID format",
			})
			return
		}

		// Get the comment to check ownership
		comment, err := commentDAO.GetByID(commentID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError
			if err.Error() == "comment not found" {
				status = http.StatusNotFound
			}
			c.JSON(status, Response{
				Success: false,
				Message: err.Error(),
			})
			return
		}

		// Check if user is the owner of the comment
		if comment.UserID != userID.(int64) {
			c.JSON(http.StatusForbidden, Response{
				Success: false,
				Message: "You do not have permission to delete this comment",
			})
			return
		}

		// Delete the comment and all its replies
		if err := commentDAO.Delete(commentID); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to delete comment: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Comment deleted successfully",
		})
	}
}

// LikeCommentHandler handles POST requests to like a comment
func LikeCommentHandler(commentDAO *dao.CommentDAO) gin.HandlerFunc {
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

		// Get comment ID from URL parameter
		commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid comment ID format",
			})
			return
		}

		// Add like
		err = commentDAO.LikeComment(commentID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError

			// Handle specific errors
			if err.Error() == "comment not found" {
				status = http.StatusNotFound
			} else if err.Error() == "comment already liked by user" {
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
			Message: "Comment liked successfully",
		})
	}
}

// UnlikeCommentHandler handles POST requests to unlike a comment
func UnlikeCommentHandler(commentDAO *dao.CommentDAO) gin.HandlerFunc {
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

		// Get comment ID from URL parameter
		commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid comment ID format",
			})
			return
		}

		// Remove like
		err = commentDAO.UnlikeComment(commentID, userID.(int64))
		if err != nil {
			status := http.StatusInternalServerError

			// Handle specific errors
			if err.Error() == "comment not found" {
				status = http.StatusNotFound
			} else if err.Error() == "comment not liked by user" {
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
			Message: "Comment unliked successfully",
		})
	}
}
