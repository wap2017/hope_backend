package api

import (
	"net/http"
	"strconv"

	"hope_backend/dao"

	"github.com/gin-gonic/gin"
)

func UserHandler(c *gin.Context) {
	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate creating a user
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created",
		"user":    user,
	})

}

// GetUserProfileHandler returns a gin.HandlerFunc that handles requests for user profiles
func GetUserProfileHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from query parameters
		idStr := c.Query("id")
		if idStr == "" {
			// If no ID provided, try to get the current user's ID from the auth context
			userID, exists := c.Get("userID")
			if !exists {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Message: "User ID is required",
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

			// Get profile by ID from the DAO
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
						Message: "Failed to retrieve user profile",
					})
				}
				return
			}

			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    profile,
			})
			return
		}

		// If ID is provided, parse it
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid user ID format",
			})
			return
		}

		// Get profile by ID from the DAO
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
					Message: "Failed to retrieve user profile",
				})
			}
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    profile,
		})
	}
}

// UpdateProfileRequest represents the request body for updating a profile
type UpdateProfileRequest struct {
	PatientName           string `json:"patient_name" binding:"required"`
	RelationshipToPatient string `json:"relationship_to_patient" binding:"required"`
	IllnessCause          string `json:"illness_cause"`
	ChatBackground        string `json:"chat_background"`
	UserAvatar            string `json:"user_avatar"`
	UserNickname          string `json:"user_nickname" binding:"required"`
}

// UpdatePasswordRequest represents the request body for changing password
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdateMobileRequest represents the request body for updating mobile number
type UpdateMobileRequest struct {
	MobileNumber     string `json:"mobile_number" binding:"required"`
	VerificationCode string `json:"verification_code" binding:"required"`
}

// UpdateUserProfileHandler returns a handler for updating user profile information
func UpdateUserProfileHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
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
					Message: "Failed to retrieve user profile",
				})
			}
			return
		}

		// Bind request body
		var req UpdateProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Update profile fields
		profile.PatientName = req.PatientName
		profile.RelationshipToPatient = req.RelationshipToPatient
		profile.IllnessCause = req.IllnessCause
		profile.ChatBackground = req.ChatBackground
		profile.UserAvatar = req.UserAvatar
		profile.UserNickname = req.UserNickname

		// Save updated profile
		if err := profileDAO.Update(profile); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to update profile: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    profile,
			Message: "Profile updated successfully",
		})
	}
}

// UpdatePasswordHandler returns a handler for changing user password
func UpdatePasswordHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
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

		// Bind request body
		var req UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Update password
		err := profileDAO.UpdatePassword(id, req.CurrentPassword, req.NewPassword)
		if err != nil {
			if err.Error() == "current password is incorrect" {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Message: "Current password is incorrect",
				})
			} else {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Message: "Failed to update password: " + err.Error(),
				})
			}
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Password updated successfully",
		})
	}
}

// UpdateMobileNumberHandler returns a handler for updating mobile number
func UpdateMobileNumberHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
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

		// Bind request body
		var req UpdateMobileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Update mobile number with verification
		err := profileDAO.UpdateMobileNumber(id, req.MobileNumber, req.VerificationCode)
		if err != nil {
			if err.Error() == "mobile number verification failed" {
				c.JSON(http.StatusBadRequest, Response{
					Success: false,
					Message: "Mobile number verification failed",
				})
			} else {
				c.JSON(http.StatusInternalServerError, Response{
					Success: false,
					Message: "Failed to update mobile number: " + err.Error(),
				})
			}
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Mobile number updated successfully",
		})
	}
}
