package main

import (
	"hope_backend/api"
	"hope_backend/config"
	"hope_backend/dao"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.InitDB() // Initialize DB connection

	// Initialize DAOs
	userProfileDAO := dao.NewUserProfileDAO(db)

	// Create a new Gin router
	r := gin.Default()

	r.Use(api.AuthMiddleware())

	// Create a group for all /hope routes
	hopeGroup := r.Group("/hope")
	{

		//测试路由
		hopeGroup.GET("/ping", api.PingHandler)
		hopeGroup.POST("/user", api.UserHandler)

		// 消息页路由
		hopeGroup.POST("/send", api.SendMessageHandler)
		hopeGroup.GET("/messages", api.GetMessagesHandler)

		// 笔记页面相关接口
		notesGroup := hopeGroup.Group("/notes")
		{
			// Create a new note
			notesGroup.POST("", api.CreateNoteHandler)

			// Get a specific note by ID
			notesGroup.GET("/:id", api.GetNoteHandler)

			// Update a note
			notesGroup.PUT("/:id", api.UpdateNoteHandler)

			// Delete a note
			notesGroup.DELETE("/:id", api.DeleteNoteHandler)

			// Get all notes for the current user
			notesGroup.GET("", api.GetUserNotesHandler)

			// Get a note for a specific date
			notesGroup.GET("/date/:date", api.GetNoteByDateHandler)

			// Get notes within a date range
			notesGroup.GET("/range", api.GetNotesByDateRangeHandler)

			// Get notes for a specific month
			notesGroup.GET("/month/:year/:month", api.GetNotesByMonthHandler)
		}

		// Future endpoints can be added here within the group
		// Diary endpoints will go here
		// Settings endpoints will go here
		// Settings page related APIs
		settingsGroup := hopeGroup.Group("/user")
		{
			// Get user profile
			settingsGroup.GET("/profile", api.GetUserProfileHandler(userProfileDAO))
		}
	}

	// Start the server on port 8080
	r.Run(":8080")
}
