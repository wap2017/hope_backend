package api

import (
	"hope_backend/dao"
	"hope_backend/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateNoteRequest is the expected request body for creating a note
type CreateNoteRequest struct {
	NoteDate string `json:"note_date" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

// UpdateNoteRequest is the expected request body for updating a note
type UpdateNoteRequest struct {
	Content string `json:"content" binding:"required"`
}

// Response is a generic JSON response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// CreateNoteHandler handles creating a new note
func CreateNoteHandler(c *gin.Context) {
	// Get user ID from context or session
	// This assumes you have middleware that sets the user ID in context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	if req.NoteDate == "" {
		// If no date provided, use today's date in the format "YYYY.M.D"
		now := time.Now()
		req.NoteDate = strconv.Itoa(now.Year()) + "." +
			strconv.Itoa(int(now.Month())) + "." +
			strconv.Itoa(now.Day())
	}

	// Check if a note already exists for this date
	existingNote, err := dao.GetNoteByUserAndDate(userID.(int), req.NoteDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to check for existing note: " + err.Error(),
		})
		return
	}

	if existingNote != nil {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Message: "A note already exists for this date",
		})
		return
	}

	note := &models.Note{
		UserID:   userID.(int),
		NoteDate: req.NoteDate,
		Content:  req.Content,
	}

	if err := dao.CreateNote(note); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to create note: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: "Note created successfully",
		Data:    note,
	})
}

// GetNoteHandler handles retrieving a note by ID
func GetNoteHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid note ID",
		})
		return
	}

	note, err := dao.GetNoteByID(noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve note: " + err.Error(),
		})
		return
	}

	if note == nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "Note not found",
		})
		return
	}

	// Check if the note belongs to the user
	if note.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, Response{
			Success: false,
			Message: "You don't have permission to access this note",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Note retrieved successfully",
		Data:    note,
	})
}

// UpdateNoteHandler handles updating an existing note
func UpdateNoteHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid note ID",
		})
		return
	}

	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Check if the note exists and belongs to the user
	note, err := dao.GetNoteByID(noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve note: " + err.Error(),
		})
		return
	}

	if note == nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "Note not found",
		})
		return
	}

	if note.UserID != userID.(int) {
		c.JSON(http.StatusForbidden, Response{
			Success: false,
			Message: "You don't have permission to update this note",
		})
		return
	}

	note.Content = req.Content
	if err := dao.UpdateNote(note); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to update note: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Note updated successfully",
		Data:    note,
	})
}

// DeleteNoteHandler handles deleting a note
func DeleteNoteHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	noteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Invalid note ID",
		})
		return
	}

	if err := dao.DeleteNote(noteID, userID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to delete note: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Note deleted successfully",
	})
}

// GetUserNotesHandler handles retrieving all notes for a user
func GetUserNotesHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	notes, err := dao.GetNotesByUserID(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve notes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Notes retrieved successfully",
		Data:    notes,
	})
}

// GetNoteByDateHandler handles retrieving a note for a specific date
func GetNoteByDateHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	date := c.Param("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Date parameter is required",
		})
		return
	}

	note, err := dao.GetNoteByUserAndDate(userID.(int), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve note: " + err.Error(),
		})
		return
	}

	if note == nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Message: "No note found for this date",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Note retrieved successfully",
		Data:    note,
	})
}

// GetNotesByDateRangeHandler handles retrieving notes within a date range
func GetNotesByDateRangeHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	startDate := c.Query("start")
	endDate := c.Query("end")

	if startDate == "" || endDate == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Both start and end date parameters are required",
		})
		return
	}

	notes, err := dao.GetNotesByDateRange(userID.(int), startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve notes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Notes retrieved successfully",
		Data:    notes,
	})
}

// GetNotesByMonthHandler handles retrieving all notes for a user for a specific month
func GetNotesByMonthHandler(c *gin.Context) {
	// Get user ID from context or session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Message: "Unauthorized: User not authenticated",
		})
		return
	}

	year := c.Param("year")
	month := c.Param("month")

	if year == "" || month == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "Both year and month parameters are required",
		})
		return
	}

	notes, err := dao.GetNotesByMonth(userID.(int), year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to retrieve notes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Notes retrieved successfully",
		Data:    notes,
	})
}
