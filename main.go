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

	// Create a group for all /hope routes
	hopeGroup := r.Group("/hope")
	{
		hopeGroup.GET("/ping", api.PingHandler)
		hopeGroup.POST("/user", api.UserHandler)

		// Chat endpoints
		hopeGroup.POST("/send", api.SendMessageHandler)
		hopeGroup.GET("/messages", api.GetMessagesHandler)

		// Future endpoints can be added here within the group
		// Diary endpoints will go here
		// Settings endpoints will go here
	}

	// Start the server on port 8080
	r.Run(":8080")
}
