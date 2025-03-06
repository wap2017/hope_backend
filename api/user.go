package api

import (
	"fmt"
	"net/http"

	"errors"
	"hope_backend/dao"
	"time"

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

// AuthService handles user authentication operations
type AuthService struct {
	userProfileDAO *dao.UserProfileDAO
}

// NewAuthService creates a new AuthService
func NewAuthService(userProfileDAO *dao.UserProfileDAO) *AuthService {
	return &AuthService{userProfileDAO: userProfileDAO}
}

// Login authenticates a user and returns a session token
func (s *AuthService) Login(mobileNumber, password string) (string, error) {
	// Verify credentials
	isValid, userID, err := s.userProfileDAO.VerifyPassword(mobileNumber, password)
	if err != nil {
		return "", err
	}

	if !isValid {
		return "", errors.New("invalid credentials")
	}

	// Generate a session token (in a real app, use JWT or similar)
	token := generateSessionToken(userID)

	// Store the token (in a real app, you'd use Redis or a sessions table)
	err = storeSessionToken(token, userID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Register creates a new user account
func (s *AuthService) Register(profile *dao.UserProfile, password string) (int64, error) {
	// Check if mobile number already exists
	existing, err := s.userProfileDAO.GetByMobileNumber(profile.MobileNumber)
	if err == nil && existing != nil {
		return 0, errors.New("mobile number already registered")
	}

	// Create the user profile
	return s.userProfileDAO.Create(profile, password)
}

// generateSessionToken creates a secure token for user sessions
func generateSessionToken(userID int64) string {
	// In a real application, implement a secure token generation method
	// This is just a placeholder
	return fmt.Sprintf("token_%d_%d", userID, time.Now().UnixNano())
}

// storeSessionToken saves the session token for later validation
func storeSessionToken(token string, userID int64) error {
	// In a real application, implement token storage (Redis, database, etc.)
	// This is just a placeholder
	return nil
}
