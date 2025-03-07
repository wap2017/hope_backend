package dao

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserProfile represents the user_profiles table structure
type UserProfile struct {
	ID                    int64  `json:"id" gorm:"primaryKey"`
	PatientName           string `json:"patient_name"`
	RelationshipToPatient string `json:"relationship_to_patient"`
	IllnessCause          string `json:"illness_cause"`
	ChatBackground        string `json:"chat_background"`
	UserAvatar            string `json:"user_avatar"`
	UserNickname          string `json:"user_nickname"`
	MobileNumber          string `json:"mobile_number" gorm:"uniqueIndex"`
	Password              string `json:"-"` // Excluded from JSON serialization
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (UserProfile) TableName() string {
	return "user_profiles"
}

// VerificationCode represents the verification_codes table structure
type VerificationCode struct {
	ID           int64  `gorm:"primaryKey"`
	MobileNumber string `gorm:"index"`
	Code         string
	ExpiresAt    int64
}

// TableName specifies the table name for GORM
func (VerificationCode) TableName() string {
	return "verification_codes"
}

// UserProfileDAO handles database operations for user profiles
type UserProfileDAO struct {
	db *gorm.DB
}

// NewUserProfileDAO creates a new UserProfileDAO
func NewUserProfileDAO(db *gorm.DB) *UserProfileDAO {
	return &UserProfileDAO{db: db}
}

// GetByID retrieves a user profile by its ID
func (dao *UserProfileDAO) GetByID(id int64) (*UserProfile, error) {
	var profile UserProfile
	result := dao.db.First(&profile, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user profile not found")
		}
		return nil, result.Error
	}

	return &profile, nil
}

// GetByMobileNumber retrieves a user profile by mobile number
func (dao *UserProfileDAO) GetByMobileNumber(mobileNumber string) (*UserProfile, error) {
	var profile UserProfile
	result := dao.db.Where("mobile_number = ?", mobileNumber).First(&profile)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user profile not found")
		}
		return nil, result.Error
	}

	return &profile, nil
}

// Create inserts a new user profile with password hashing
func (dao *UserProfileDAO) Create(profile *UserProfile, plainPassword string) (int64, error) {
	// Hash the password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	// Set timestamps
	now := time.Now().UnixMilli()
	profile.CreatedAt = now
	profile.UpdatedAt = now
	profile.Password = string(hashedPassword)

	// Create the record
	result := dao.db.Create(profile)
	if result.Error != nil {
		return 0, result.Error
	}

	return profile.ID, nil
}

// Update updates an existing user profile
func (dao *UserProfileDAO) Update(profile *UserProfile) error {
	profile.UpdatedAt = time.Now().UnixMilli()

	result := dao.db.Model(profile).Updates(map[string]interface{}{
		"patient_name":            profile.PatientName,
		"relationship_to_patient": profile.RelationshipToPatient,
		"illness_cause":           profile.IllnessCause,
		"chat_background":         profile.ChatBackground,
		"user_avatar":             profile.UserAvatar,
		"user_nickname":           profile.UserNickname,
		"updated_at":              profile.UpdatedAt,
	})

	return result.Error
}

// UpdatePassword changes a user's password
func (dao *UserProfileDAO) UpdatePassword(userID int64, currentPassword, newPassword string) error {
	// First verify the current password
	profile, err := dao.GetByID(userID)
	if err != nil {
		return err
	}

	// Check if current password matches
	err = bcrypt.CompareHashAndPassword([]byte(profile.Password), []byte(currentPassword))
	if err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update the password
	now := time.Now().UnixMilli()
	result := dao.db.Model(&UserProfile{ID: userID}).Updates(map[string]interface{}{
		"password":   string(hashedPassword),
		"updated_at": now,
	})

	return result.Error
}

// VerifyPassword checks if the provided password matches the stored hash
func (dao *UserProfileDAO) VerifyPassword(mobileNumber, password string) (bool, int64, error) {
	profile, err := dao.GetByMobileNumber(mobileNumber)
	if err != nil {
		return false, 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(profile.Password), []byte(password))
	if err != nil {
		return false, 0, nil // Password doesn't match but not an error
	}

	return true, profile.ID, nil // Password matches
}

// UpdateMobileNumber handles the special case of updating a mobile number with verification
func (dao *UserProfileDAO) UpdateMobileNumber(userID int64, newMobileNumber string, verificationCode string) error {
	// First verify the mobile number belongs to the user
	isVerified, err := dao.verifyMobileNumber(newMobileNumber, verificationCode)
	if err != nil {
		return err
	}

	if !isVerified {
		return errors.New("mobile number verification failed")
	}

	// If verification passed, update the mobile number
	now := time.Now().UnixMilli()
	result := dao.db.Model(&UserProfile{ID: userID}).Updates(map[string]interface{}{
		"mobile_number": newMobileNumber,
		"updated_at":    now,
	})

	return result.Error
}

// verifyMobileNumber checks if the mobile number belongs to the user via verification code
func (dao *UserProfileDAO) verifyMobileNumber(mobileNumber string, verificationCode string) (bool, error) {
	var count int64
	now := time.Now().UnixMilli()

	// Check if there's a valid verification code
	result := dao.db.Model(&VerificationCode{}).
		Where("mobile_number = ? AND code = ? AND expires_at > ?", mobileNumber, verificationCode, now).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	if count > 0 {
		// Delete the used verification code
		dao.db.Where("mobile_number = ? AND code = ?", mobileNumber, verificationCode).Delete(&VerificationCode{})
		return true, nil
	}

	return false, nil
}
