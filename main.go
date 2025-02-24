package main

import (
	"hope_backend/api"
	"hope_backend/config"

	"github.com/gin-gonic/gin"
)

func main() {
	config.InitDB() // Initialize DB connection

	// Create a new Gin router
	r := gin.Default()

	// Simple ping endpoint to check server status
	r.GET("/ping", api.PingHandler)
	r.POST("/user", api.UserHandler)

	// 聊天页接口
	r.POST("/send", api.SendMessageHandler)
	r.GET("/messages", api.GetMessagesHandler)

	// 日记页接口  TODO

	// 设置页接口 TODO

	// Start the server on port 8080
	r.Run(":8080")
}
