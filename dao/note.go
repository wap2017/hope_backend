package dao

import (
	"hope_backend/config"
	"hope_backend/models"
)

// CreateNote inserts a new note into the database
func CreateNote(note *models.Note) error {
	return config.DB.Create(note).Error
}

// GetNoteByID retrieves a note by its ID
func GetNoteByID(noteID int) (*models.Note, error) {
	var note models.Note
	err := config.DB.Where("note_id = ?", noteID).First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// GetNoteByUserAndDate retrieves a note by user ID and date
func GetNoteByUserAndDate(userID int64, noteDate string) (*models.Note, error) {
	var note models.Note
	err := config.DB.Where("user_id = ? AND note_date = ?", userID, noteDate).First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// GetNotesByUserID retrieves all notes for a specific user
func GetNotesByUserID(userID int64) ([]models.Note, error) {
	var notes []models.Note
	err := config.DB.Where("user_id = ?", userID).Order("note_date DESC").Find(&notes).Error
	return notes, err
}

// GetNotesByDateRange retrieves notes for a user within a date range
func GetNotesByDateRange(userID int64, startDate, endDate string) ([]models.Note, error) {
	var notes []models.Note
	err := config.DB.Where("user_id = ? AND note_date BETWEEN ? AND ?", userID, startDate, endDate).
		Order("note_date ASC").Find(&notes).Error
	return notes, err
}

// UpdateNote modifies an existing note
func UpdateNote(note *models.Note) error {
	return config.DB.Model(note).Where("note_id = ? AND user_id = ?", note.NoteID, note.UserID).
		Update("content", note.Content).Error
}

// DeleteNote removes a note from the database
func DeleteNote(noteID int, userID int64) error {
	return config.DB.Where("note_id = ? AND user_id = ?", noteID, userID).Delete(&models.Note{}).Error
}

// GetNotesByMonth retrieves all notes for a user for a specific month
func GetNotesByMonth(userID int64, year, month string) ([]models.Note, error) {
	var notes []models.Note

	// Construct date pattern for the specified month (e.g., "2023.1.%")
	datePattern := year + "." + month + ".%"

	err := config.DB.Where("user_id = ? AND note_date LIKE ?", userID, datePattern).
		Order("note_date ASC").Find(&notes).Error
	return notes, err
}
