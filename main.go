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

	r.POST("/send", api.SendMessageHandler)
	r.GET("/messages", api.GetMessagesHandler)

	// Start the server on port 8080
	r.Run(":8080")
}
