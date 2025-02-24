package api

import (
	"context"

	"github.com/sashabaranov/go-openai"

	"fmt"
	"hope_backend/dao"
	"hope_backend/models"
	"net/http"
	"os"
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

	// 进行chatgpt回复
	aiRsp, err := getChatGPTResponse(msg.Content)
	if err != nil {
		fmt.Printf("err=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ai resp error"})
		return
	}

	// 将AI回复插入数据库中
	if err = dao.CreateMessage(&models.Message{
		SenderID:    1, //system的用户id固定是1
		ReceiverID:  msg.UserID,
		ChatID:      msg.ChatID,
		Content:     aiRsp,
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

func getChatGPTResponse(userInput string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: userInput},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// GetMessagesHandler fetches all messages for a chat
func GetMessagesHandler(c *gin.Context) {
	chatID := c.Query("chat_id")
	// userID, err := strconv.Atoi(c.Query("user_id"))
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
	// 	return
	// }

	lastID, err := strconv.Atoi(c.Query("last_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid last ID"})
		return
	}

	messages, err := dao.GetMessages(chatID, uint(lastID), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
