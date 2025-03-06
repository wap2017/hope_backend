package dao

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserProfile represents the user_profiles table structure
type UserProfile struct {
	ID                    int64  `json:"id"`
	PatientName           string `json:"patient_name"`
	RelationshipToPatient string `json:"relationship_to_patient"`
	IllnessCause          string `json:"illness_cause"`
	ChatBackground        string `json:"chat_background"`
	UserAvatar            string `json:"user_avatar"`
	UserNickname          string `json:"user_nickname"`
	MobileNumber          string `json:"mobile_number"`
	Password              string `json:"-"` // Excluded from JSON serialization
	CreatedAt             int64  `json:"created_at"`
	UpdatedAt             int64  `json:"updated_at"`
}

// UserProfileDAO handles database operations for user profiles
type UserProfileDAO struct {
	db *sql.DB
}

// NewUserProfileDAO creates a new UserProfileDAO
func NewUserProfileDAO(db *sql.DB) *UserProfileDAO {
	return &UserProfileDAO{db: db}
}

// GetByID retrieves a user profile by its ID
func (dao *UserProfileDAO) GetByID(id int64) (*UserProfile, error) {
	query := `
		SELECT id, patient_name, relationship_to_patient, illness_cause,
			   chat_background, user_avatar, user_nickname, mobile_number,
			   password, created_at, updated_at
		FROM user_profiles
		WHERE id = ?
	`

	row := dao.db.QueryRow(query, id)

	var profile UserProfile
	err := row.Scan(
		&profile.ID,
		&profile.PatientName,
		&profile.RelationshipToPatient,
		&profile.IllnessCause,
		&profile.ChatBackground,
		&profile.UserAvatar,
		&profile.UserNickname,
		&profile.MobileNumber,
		&profile.Password,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user profile not found")
		}
		return nil, err
	}

	return &profile, nil
}

// GetByMobileNumber retrieves a user profile by mobile number
func (dao *UserProfileDAO) GetByMobileNumber(mobileNumber string) (*UserProfile, error) {
	query := `
		SELECT id, patient_name, relationship_to_patient, illness_cause,
			   chat_background, user_avatar, user_nickname, mobile_number,
			   password, created_at, updated_at
		FROM user_profiles
		WHERE mobile_number = ?
	`

	row := dao.db.QueryRow(query, mobileNumber)

	var profile UserProfile
	err := row.Scan(
		&profile.ID,
		&profile.PatientName,
		&profile.RelationshipToPatient,
		&profile.IllnessCause,
		&profile.ChatBackground,
		&profile.UserAvatar,
		&profile.UserNickname,
		&profile.MobileNumber,
		&profile.Password,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user profile not found")
		}
		return nil, err
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

	query := `
		INSERT INTO user_profiles (
			patient_name, relationship_to_patient, illness_cause,
			chat_background, user_avatar, user_nickname, mobile_number,
			password, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().UnixMilli()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	result, err := dao.db.Exec(
		query,
		profile.PatientName,
		profile.RelationshipToPatient,
		profile.IllnessCause,
		profile.ChatBackground,
		profile.UserAvatar,
		profile.UserNickname,
		profile.MobileNumber,
		string(hashedPassword),
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	profile.ID = id
	profile.Password = string(hashedPassword)
	return id, nil
}

// Update updates an existing user profile
func (dao *UserProfileDAO) Update(profile *UserProfile) error {
	query := `
		UPDATE user_profiles
		SET patient_name = ?,
			relationship_to_patient = ?,
			illness_cause = ?,
			chat_background = ?,
			user_avatar = ?,
			user_nickname = ?,
			updated_at = ?
		WHERE id = ?
	`

	profile.UpdatedAt = time.Now().UnixMilli()

	_, err := dao.db.Exec(
		query,
		profile.PatientName,
		profile.RelationshipToPatient,
		profile.IllnessCause,
		profile.ChatBackground,
		profile.UserAvatar,
		profile.UserNickname,
		profile.UpdatedAt,
		profile.ID,
	)

	return err
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
	query := `
		UPDATE user_profiles
		SET password = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now().UnixMilli()
	_, err = dao.db.Exec(query, string(hashedPassword), now, userID)
	return err
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
	query := `
		UPDATE user_profiles
		SET mobile_number = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now().UnixMilli()

	_, err = dao.db.Exec(query, newMobileNumber, now, userID)
	return err
}

// verifyMobileNumber checks if the mobile number belongs to the user via verification code
func (dao *UserProfileDAO) verifyMobileNumber(mobileNumber string, verificationCode string) (bool, error) {
	// Implementation remains the same as before
	query := `
		SELECT EXISTS(
			SELECT 1 FROM verification_codes
			WHERE mobile_number = ? AND code = ? AND expires_at > ?
		)
	`

	var exists bool
	err := dao.db.QueryRow(query, mobileNumber, verificationCode, time.Now().UnixMilli()).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists {
		_, _ = dao.db.Exec("DELETE FROM verification_codes WHERE mobile_number = ? AND code = ?",
			mobileNumber, verificationCode)
	}

	return exists, nil
}
