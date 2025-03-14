package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"hope_backend/dao"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// JWT secret key - should be stored in an environment variable in production
var jwtKey = []byte("your_secret_key")

// RegisterUserRequest represents the request body for user registration
type RegisterUserRequest struct {
	MobileNumber          string `json:"mobile_number" binding:"required"`
	Password              string `json:"password" binding:"required,min=8"`
	VerificationCode      string `json:"verification_code" binding:"required"`
	PatientName           string `json:"patient_name" binding:"required"`
	RelationshipToPatient string `json:"relationship_to_patient" binding:"required"`
	IllnessCause          string `json:"illness_cause"`
	UserNickname          string `json:"user_nickname" binding:"required"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	MobileNumber string `json:"mobile_number" binding:"required"`
	Password     string `json:"password" binding:"required"`
}

// VerificationCodeRequest represents the request for sending verification codes
type VerificationCodeRequest struct {
	MobileNumber string `json:"mobile_number" binding:"required"`
}

// VerifyMobileRequest represents the request for verifying a mobile number
type VerifyMobileRequest struct {
	MobileNumber     string `json:"mobile_number" binding:"required"`
	VerificationCode string `json:"verification_code" binding:"required"`
}

// Claims structure for JWT payload
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware checks for a valid JWT token in Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Public paths that don't need authentication
		publicPaths := []string{
			"/hope/ping",
			"/hope/auth/register",
			"/hope/auth/login",
			"/hope/auth/verification-code",
			"/hope/auth/verify-mobile",
			// "/hope/user",
			"/hope/file/posts",
		}

		// Skip authentication for public paths
		requestPath := c.Request.URL.Path
		for _, path := range publicPaths {
			if strings.HasPrefix(requestPath, path) {
				c.Next()
				return
			}
		}

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, Response{
				Success: false,
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if the Auth header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, Response{
				Success: false,
				Message: "Invalid authorization format. Bearer token required",
			})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse the token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate signing algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, Response{
				Success: false,
				Message: "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user ID in context
		c.Set("userID", claims.UserID)
		c.Next()
	}
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token valid for 24 hours

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	return tokenString, err
}

// RegisterUserHandler handles user registration
func RegisterUserHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Before creating the user profile, check if mobile number is already registered
		_, err := profileDAO.GetByMobileNumber(req.MobileNumber)
		if err == nil {
			// If no error occurs, it means a profile with this mobile number already exists
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Mobile number is already registered",
			})
			return
		}
		// Only proceed if the error is "user profile not found"
		if err.Error() != "user profile not found" {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Error checking mobile number: " + err.Error(),
			})
			return
		}

		// Verify mobile number with verification code
		// This would typically be implemented with a VerificationDAO
		isVerified := verifyMobileCode(req.MobileNumber, req.VerificationCode)
		if !isVerified {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid verification code",
			})
			return
		}

		// Create new user profile
		profile := &dao.UserProfile{
			PatientName:           req.PatientName,
			RelationshipToPatient: req.RelationshipToPatient,
			IllnessCause:          req.IllnessCause,
			UserNickname:          req.UserNickname,
			MobileNumber:          req.MobileNumber,
			// Default values for other fields
			ChatBackground: "",
			UserAvatar:     "",
		}

		// Create the user with password
		userID, err := profileDAO.Create(profile, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to create user: " + err.Error(),
			})
			return
		}

		// Generate JWT token
		token, err := GenerateToken(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to generate token",
			})
			return
		}

		// Return success with token and user profile
		c.JSON(http.StatusCreated, Response{
			Success: true,
			Message: "User registered successfully",
			Data: gin.H{
				"token":   token,
				"profile": profile,
			},
		})
	}
}

// LoginHandler handles user login
func LoginHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Verify credentials
		isValid, userID, err := profileDAO.VerifyPassword(req.MobileNumber, req.Password)
		fmt.Printf("login: %+v\n", req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Error verifying credentials: " + err.Error(),
			})
			return
		}

		if !isValid {
			c.JSON(http.StatusUnauthorized, Response{
				Success: false,
				Message: "Invalid credentials",
			})
			return
		}

		// Get user profile
		profile, err := profileDAO.GetByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to retrieve user profile",
			})
			return
		}

		// Generate JWT token
		token, err := GenerateToken(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to generate token",
			})
			return
		}

		// Return success with token and user profile
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Login successful",
			Data: gin.H{
				"token":   token,
				"profile": profile,
			},
		})
	}
}

// RequestVerificationCodeHandler sends verification code to a mobile number
func RequestVerificationCodeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VerificationCodeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Generate and send verification code
		// This would typically involve sending an SMS
		code := generateVerificationCode()
		if !sendVerificationSMS(req.MobileNumber, code) {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to send verification code",
			})
			return
		}

		// Store verification code in database
		if !storeVerificationCode(req.MobileNumber, code) {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Message: "Failed to process verification code",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Verification code sent successfully",
		})
	}
}

// VerifyMobileNumberHandler verifies a mobile number with a verification code
func VerifyMobileNumberHandler(profileDAO *dao.UserProfileDAO) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VerifyMobileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid request: " + err.Error(),
			})
			return
		}

		// Verify the mobile number with the provided code
		isVerified := verifyMobileCode(req.MobileNumber, req.VerificationCode)
		if !isVerified {
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Message: "Invalid verification code",
			})
			return
		}

		// If verification successful, return success response
		c.JSON(http.StatusOK, Response{
			Success: true,
			Message: "Mobile number verified successfully",
		})
	}
}

// Helper functions - these would typically be implemented in a separate service

// verifyMobileCode checks if the verification code is valid for the mobile number
func verifyMobileCode(mobileNumber, code string) bool {
	// This is a placeholder - implement actual verification
	// In a real implementation, you would check the database for a matching code
	// that hasn't expired yet
	return true // For testing purposes
}

// generateVerificationCode creates a random verification code
func generateVerificationCode() string {
	// Generate a random 6-digit code
	// This is a placeholder - implement actual code generation
	return "123456" // For testing purposes
}

// sendVerificationSMS sends an SMS with the verification code
func sendVerificationSMS(mobileNumber, code string) bool {
	// This is a placeholder - implement actual SMS sending
	// You would typically use an SMS gateway service
	return true // For testing purposes
}

// storeVerificationCode saves the verification code to the database
func storeVerificationCode(mobileNumber, code string) bool {
	// This is a placeholder - implement actual code storage
	// Store the code in the database with an expiration time
	return true // For testing purposes
}
