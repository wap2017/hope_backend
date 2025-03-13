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
	UserID  int64  `json:"user_id"`
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
func SendMessageHandler(profileDAO *dao.UserProfileDAO) func(c *gin.Context) {
	return func(c *gin.Context) {
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

		// 查询用户信息
		// TODO

		user, err := profileDAO.GetByID(msg.UserID)
		if err != nil {
			fmt.Printf("err:%v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message"})
			return
		}

		// 进行chatgpt回复
		// aiRsp, err := getChatGPTResponse(msg.Content)
		aiRsp, err := getChatGPTResponseEnhance(msg.Content,
			user.PatientName, user.RelationshipToPatient, user.IllnessCause)
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
}

// TODO 这里要加提示词
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

// getChatGPTResponse enhances responses with patient context for more empathetic communication
func getChatGPTResponseEnhance(userInput string, patientName string, relationshipToPatient string, illnessCause string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	client := openai.NewClient(apiKey)

	// Create contextual prompt with patient information in Chinese
	contextualPrompt := fmt.Sprintf(
		"你正在回复一位照顾抑郁症患者的人。"+
			"患者姓名：%s。照顾者与患者的关系：%s。"+
			"关于患者病情的背景：%s。"+
			"请提供一个富有同情心和支持性的回复，同时认可他们所处的情况。"+
			"原始消息：%s",
		patientName, relationshipToPatient, illnessCause, userInput)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{Role: "system", Content: "你是一位富有同情心的助手，帮助那些正在照顾抑郁症亲人的人。提供支持性、有同理心的回应，并在适当的时候提供实用的指导。"},
				{Role: "user", Content: contextualPrompt},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// // getChatGPTResponse enhances responses with patient context for more empathetic communication
// func getChatGPTResponseEnhance(userInput string, patientName string, relationshipToPatient string, illnessCause string) (string, error) {
// 	apiKey := os.Getenv("OPENAI_API_KEY")
// 	client := openai.NewClient(apiKey)

// 	// Create contextual prompt with patient information
// 	contextualPrompt := fmt.Sprintf(
// 		"You are responding to someone who is caring for a person with depression. "+
// 			"Patient's name: %s. Relationship to caregiver: %s. "+
// 			"Context about the illness: %s. "+
// 			"Provide a compassionate, supportive response that acknowledges their situation. "+
// 			"Original message: %s",
// 		patientName, relationshipToPatient, illnessCause, userInput)

// 	resp, err := client.CreateChatCompletion(
// 		context.Background(),
// 		openai.ChatCompletionRequest{
// 			Model: "gpt-4o-mini",
// 			Messages: []openai.ChatCompletionMessage{
// 				{Role: "system", Content: "You are a compassionate assistant helping someone who is caring for a loved one with depression. Provide supportive, empathetic responses while offering practical guidance when appropriate."},
// 				{Role: "user", Content: contextualPrompt},
// 			},
// 		},
// 	)
// 	if err != nil {
// 		return "", err
// 	}
// 	return resp.Choices[0].Message.Content, nil
// }

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
