package api

import (
	"hope_backend/dao"
	"hope_backend/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type SendMsg struct {
	UserID  uint   `json:"user_id"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

const (
	MsgType_Unknow = iota
	MsgType_Text
)

const (
	MsgStatus_Send = iota
)

// SendMessageHandler handles sending a message
func SendMessageHandler(c *gin.Context) {
	var msg SendMsg
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UnixMicro()

	if err := dao.CreateMessage(&models.Message{
		SenderID:    msg.UserID,
		ReceiverID:  1, //system的用户id固定是1
		ChatID:      msg.ChatID,
		Content:     msg.Content,
		MsgType:     MsgType_Text,
		Status:      MsgStatus_Send,
		CreatedTime: now,
		UpdatedTime: now,
	}); err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

// GetMessagesHandler fetches all messages for a chat
func GetMessagesHandler(c *gin.Context) {
	chatID := c.Query("chat_id")
	userID, err := strconv.Atoi(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	lastID, err := strconv.Atoi(c.Query("last_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last ID"})
		return
	}

	messages, err := dao.GetMessages(chatID, uint(userID), uint(lastID), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
