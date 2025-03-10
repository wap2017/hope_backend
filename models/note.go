package models

// Note represents a daily note in the system
type Note struct {
	NoteID    int    `json:"note_id" db:"note_id"`
	UserID    int64  `json:"user_id" db:"user_id"`
	NoteDate  string `json:"note_date" db:"note_date"`
	Content   string `json:"content" db:"content"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
}
