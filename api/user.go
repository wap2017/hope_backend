package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func UserHandler(c *gin.Context) {
	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simulate creating a user
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created",
		"user":    user,
	})

}
