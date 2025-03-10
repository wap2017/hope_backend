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
	postDAO := dao.NewPostDAO(db)
	commentDAO := dao.NewCommentDAO(db)

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

			// Update user profile
			settingsGroup.PUT("/profile", api.UpdateUserProfileHandler(userProfileDAO))

			// Update user password
			settingsGroup.PUT("/password", api.UpdatePasswordHandler(userProfileDAO))

			// Update mobile number with verification
			settingsGroup.PUT("/mobile", api.UpdateMobileNumberHandler(userProfileDAO))
		}

		// Authentication routes (outside the settingsGroup)
		authGroup := hopeGroup.Group("/auth")
		{
			// User registration
			authGroup.POST("/register", api.RegisterUserHandler(userProfileDAO))

			// User login
			authGroup.POST("/login", api.LoginHandler(userProfileDAO))

			// Request verification code for mobile number
			authGroup.POST("/verification-code", api.RequestVerificationCodeHandler())

			// Verify mobile number
			authGroup.POST("/verify-mobile", api.VerifyMobileNumberHandler(userProfileDAO))
		}

		// Inside the hopeGroup
		// Post-related endpoints
		postsGroup := hopeGroup.Group("/posts")
		{
			// Create a new post
			postsGroup.POST("", api.CreatePostHandler(postDAO))

			// Get a post by ID
			postsGroup.GET("/:id", api.GetPostHandler(postDAO))

			// Update a post
			postsGroup.PUT("/:id", api.UpdatePostHandler(postDAO))

			// Delete a post
			postsGroup.DELETE("/:id", api.DeletePostHandler(postDAO))

			// List posts with pagination
			postsGroup.GET("", api.ListPostsHandler(postDAO))

			// Like a post
			postsGroup.POST("/:id/like", api.LikePostHandler(postDAO))

			// Unlike a post
			postsGroup.POST("/:id/unlike", api.UnlikePostHandler(postDAO))

			// Comment endpoints
			postsGroup.POST("/:id/comments", api.CreateCommentHandler(commentDAO))
			postsGroup.GET("/:id/comments", api.ListCommentsHandler(commentDAO))
		}

		// Comment-related endpoints
		commentsGroup := hopeGroup.Group("/comments")
		{
			// Delete a comment
			commentsGroup.DELETE("/:id", api.DeleteCommentHandler(commentDAO))

			// Like a comment
			commentsGroup.POST("/:id/like", api.LikeCommentHandler(commentDAO))

			// Unlike a comment
			commentsGroup.POST("/:id/unlike", api.UnlikeCommentHandler(commentDAO))
		}

		hopeGroup.Static("/file/posts", "./uploads/posts")

	}

	// Set up static file serving for uploaded files
	// r.Static("/hope/file/post", "./uploads/posts") // assuming your images are stored in ./uploads/post directory
	// r.Static("/uploads", "./uploads")

	// Start the server on port 8080
	r.Run(":8080")
}
