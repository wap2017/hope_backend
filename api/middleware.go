package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks for a valid authentication token and sets the user ID in the context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		//just for test
		c.Set("userID", 2)
		c.Next()
		return

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")

		// Check if the header is empty or doesn't start with "Bearer "
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Authorization header is required or invalid",
			})
			c.Abort()
			return
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Here you would validate the token using your authentication system
		// This is just a placeholder for your actual token validation logic
		userID, valid := validateToken(token)

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set the user ID in the context for use in handlers
		c.Set("userID", userID)

		// Continue to the next middleware or handler
		c.Next()
	}
}

// validateToken is a placeholder function for your actual token validation logic
// It should return the user ID and a boolean indicating if the token is valid
func validateToken(token string) (int, bool) {
	// Replace this with your actual token validation logic
	// For now, we'll just return a dummy user ID if the token isn't empty
	if token != "" {
		// This is just a placeholder - in a real app, you would:
		// 1. Verify the token signature
		// 2. Check if the token is expired
		// 3. Extract the user ID from the token claims
		return 1, true // Assuming user ID 1 for illustration
	}
	return 0, false
}
