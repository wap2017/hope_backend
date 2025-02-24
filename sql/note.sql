CREATE TABLE notes (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    note_date DATE NOT NULL,
    content VARCHAR(500) NOT NULL,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_time TIMESTAMP NULL DEFAULT NULL,
    INDEX idx_user_date (user_id, note_date),
    INDEX idx_soft_delete (deleted_at)
);
