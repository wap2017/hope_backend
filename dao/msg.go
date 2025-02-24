package dao

import (
	"hope_backend/config"
	"hope_backend/models"
)

// CreateMessage inserts a new message into the database
func CreateMessage(msg *models.Message) error {
	return config.DB.Create(msg).Error
}

// GetMessages retrieves messages using `id` as the offset for pagination
func GetMessages(chatID string, lastID uint, pageSize int) ([]models.Message, error) {
	var messages []models.Message
	query := config.DB.Where("chat_id = ?", chatID)

	// If lastID is provided, fetch messages with IDs greater than lastID
	if lastID > 0 {
		query = query.Where("id > ?", lastID)
	}

	err := query.Order("id").Limit(pageSize).Find(&messages).Error
	return messages, err
}
