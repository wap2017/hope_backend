package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hope_backend/dao"
	"hope_backend/models"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
)

// AI Provider types
type AIProvider string

const (
	ProviderOpenAI   AIProvider = "openai"
	ProviderDeepSeek AIProvider = "deepseek"
	ProviderClaude   AIProvider = "claude"
)

// Rate limiter for API calls
var (
	rateLimiter = make(map[int64]time.Time)
	rateMutex   sync.RWMutex
	minInterval = 10 * time.Second // Can be adjusted based on your needs
)

type SendMsg struct {
	UserID  int64  `json:"user_id"`
	ChatID  string `json:"chat_id"`
	Content string `json:"content"`
}

// Claude API structures
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const (
	MsgType_Unknow = iota
	MsgType_Text
)

const (
	MsgStatus_Send = iota
)

// Check if user can make an API call (rate limiting)
func canMakeAPICall(userID int64) bool {
	rateMutex.RLock()
	lastCall, exists := rateLimiter[userID]
	rateMutex.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastCall) >= minInterval
}

// Record API call time
func recordAPICall(userID int64) {
	rateMutex.Lock()
	rateLimiter[userID] = time.Now()
	rateMutex.Unlock()
}

// SendMessageHandler handles sending a message with multiple AI provider support
func SendMessageHandler(profileDAO *dao.UserProfileDAO) func(c *gin.Context) {
	return func(c *gin.Context) {
		var msg SendMsg
		if err := c.ShouldBindJSON(&msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		now := time.Now().UnixMicro()

		// Save user message first
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

		// Check rate limiting
		if !canMakeAPICall(msg.UserID) {
			aiRsp := "请稍等一下再发送消息，让我有时间为您提供最好的回复。谢谢您的耐心！"

			if err := dao.CreateMessage(&models.Message{
				SenderID:    1,
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
			return
		}

		// Get user info
		user, err := profileDAO.GetByID(msg.UserID)
		if err != nil {
			fmt.Printf("err:%v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}

		// Record the API call attempt
		recordAPICall(msg.UserID)

		// Process AI response in goroutine (async)
		go func() {
			// Try different AI providers with fallback logic
			aiRsp := getAIResponse(msg.Content, user.PatientName, user.RelationshipToPatient, user.IllnessCause)

			// Save AI response
			if err := dao.CreateMessage(&models.Message{
				SenderID:    1, //system的用户id固定是1
				ReceiverID:  msg.UserID,
				ChatID:      msg.ChatID,
				Content:     aiRsp,
				MsgType:     MsgType_Text,
				Status:      MsgStatus_Send,
				CreatedTime: time.Now().UnixMicro(),
				UpdatedTime: time.Now().UnixMicro(),
			}); err != nil {
				fmt.Printf("[AI Response] Failed to save AI response: %v\n", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
	}
}

// getAIResponse tries multiple providers with fallback logic
func getAIResponse(userInput, patientName, relationshipToPatient, illnessCause string) string {
	// Priority order: DeepSeek (cheapest) -> Claude Haiku -> OpenAI -> fallback

	// Try DeepSeek first (cheapest and good quality)
	start := time.Now()
	if response, err := getDeepSeekResponse(userInput, patientName, relationshipToPatient, illnessCause); err == nil {
		duration := time.Since(start)
		fmt.Printf("[AI Response] DeepSeek success in %v\n", duration)
		return response
	} else {
		duration := time.Since(start)
		fmt.Printf("[AI Response] DeepSeek failed in %v: %v\n", duration, err)
	}

	// Try Claude Haiku (good balance of cost/quality)
	start = time.Now()
	if response, err := getClaudeResponse(userInput, patientName, relationshipToPatient, illnessCause); err == nil {
		duration := time.Since(start)
		fmt.Printf("[AI Response] Claude success in %v\n", duration)
		return response
	} else {
		duration := time.Since(start)
		fmt.Printf("[AI Response] Claude failed in %v: %v\n", duration, err)
	}

	// Try OpenAI as fallback
	start = time.Now()
	if response, err := getChatGPTResponseEnhance(userInput, patientName, relationshipToPatient, illnessCause); err == nil {
		duration := time.Since(start)
		fmt.Printf("[AI Response] OpenAI success in %v\n", duration)
		return response
	} else {
		duration := time.Since(start)
		fmt.Printf("[AI Response] OpenAI failed in %v: %v\n", duration, err)
	}

	// All providers failed - return default response
	fmt.Printf("[AI Response] All providers failed, using fallback\n")
	return "抱歉，我现在暂时无法回复。请稍后再试，或者告诉我更多关于您当前情况的信息，我会尽力帮助您。"
}

// getClaudeResponse uses Claude API
func getClaudeResponse(userInput, patientName, relationshipToPatient, illnessCause string) (string, error) {
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("Claude API key not configured")
	}

	systemPrompt := "你是一位富有同情心的助手，帮助那些正在照顾抑郁症亲人的人。提供支持性、有同理心的回应，并在适当的时候提供实用的指导。请用温暖、理解的语调回复，避免过于技术性的建议。"

	contextualPrompt := fmt.Sprintf(
		"你正在回复一位照顾抑郁症患者的人。"+
			"患者姓名：%s。照顾者与患者的关系：%s。"+
			"关于患者病情的背景：%s。"+
			"请提供一个富有同情心和支持性的回复，同时认可他们所处的情况。"+
			"原始消息：%s",
		patientName, relationshipToPatient, illnessCause, userInput)

	request := ClaudeRequest{
		Model:     "claude-3-5-haiku-20241022", // Using cheaper Haiku model
		MaxTokens: 800,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: contextualPrompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", err
	}

	if claudeResp.Error.Message != "" {
		return "", fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	return claudeResp.Content[0].Text, nil
}

// getDeepSeekResponse uses DeepSeek API (compatible with OpenAI client)
func getDeepSeekResponse(userInput, patientName, relationshipToPatient, illnessCause string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DeepSeek API key not configured")
	}

	// Create OpenAI client but point to DeepSeek endpoint
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.deepseek.com"
	client := openai.NewClientWithConfig(config)

	// Create contextual prompt
	contextualPrompt := fmt.Sprintf(
		"你正在回复一位照顾抑郁症患者的人。"+
			"患者姓名：%s。照顾者与患者的关系：%s。"+
			"关于患者病情的背景：%s。"+
			"请提供一个富有同情心和支持性的回复，同时认可他们所处的情况。"+
			"原始消息：%s",
		patientName, relationshipToPatient, illnessCause, userInput)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       "deepseek-chat", // DeepSeek's main model
			MaxTokens:   800,
			Temperature: 0.7,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "你是一位富有同情心的助手，帮助那些正在照顾抑郁症亲人的人。提供支持性、有同理心的回应，并在适当的时候提供实用的指导。请用温暖、理解的语调回复，避免过于技术性的建议。",
				},
				{
					Role:    "user",
					Content: contextualPrompt,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// getChatGPTResponse - original simple function (kept for backwards compatibility)
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

// getChatGPTResponseEnhance - enhanced OpenAI function with context
func getChatGPTResponseEnhance(userInput, patientName, relationshipToPatient, illnessCause string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	client := openai.NewClient(apiKey)

	// Create contextual prompt with patient information in Chinese
	contextualPrompt := fmt.Sprintf(
		"你正在回复一位照顾抑郁症患者的人。"+
			"患者姓名：%s。照顾者与患者的关系：%s。"+
			"关于患者病情的背景：%s。"+
			"请提供一个富有同情心和支持性的回复，同时认可他们所处的情况。"+
			"原始消息：%s",
		patientName, relationshipToPatient, illnessCause, userInput)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:       "gpt-4o-mini",
			MaxTokens:   500,
			Temperature: 0.7,
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

// GetMessagesHandler fetches all messages for a chat
func GetMessagesHandler(c *gin.Context) {
	chatID := c.Query("chat_id")

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
