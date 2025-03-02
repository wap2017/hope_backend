-- Notes table to store the daily notes
CREATE TABLE notes (
    note_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    note_date VARCHAR(20) NOT NULL, -- Using VARCHAR for date storage (e.g., "2023.1.18" format)
    content VARCHAR(1000) NOT NULL, -- Using VARCHAR instead of TEXT
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    -- Ensure a user can only have one note per date
    UNIQUE KEY (user_id, note_date),
    -- Add index to improve query performance
    INDEX idx_notes_user_date (user_id, note_date)
);

