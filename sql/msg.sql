-- 用户表
CREATE TABLE users (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    username     VARCHAR(255) NOT NULL UNIQUE,  -- User's unique name
    password     VARCHAR(255) NOT NULL,         -- Hashed password
    avatar       VARCHAR(500) DEFAULT '',       -- Profile picture URL
    email        VARCHAR(255) UNIQUE,           -- Optional email
    phone        VARCHAR(20) UNIQUE,            -- Optional phone number
    status       TINYINT NOT NULL DEFAULT 1,    -- 1=Active, 0=Inactive, 2=Banned
    created_time   BIGINT NOT NULL,               -- Unix timestamp for creation time
    updated_time   BIGINT NOT NULL,    
    INDEX idx_username (username),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 创建消息表
CREATE TABLE messages (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    sender_id     BIGINT NOT NULL, 
    receiver_id   BIGINT NOT NULL, 
    chat_id       VARCHAR(50) NOT NULL,
    content       VARCHAR(2000) NOT NULL,  -- Replacing TEXT with VARCHAR(2000)
    msg_type      TINYINT NOT NULL DEFAULT 1, -- Using TINYINT instead of ENUM (1=text, 2=image, 3=video, etc.)
    status        TINYINT NOT NULL DEFAULT 0, -- Using TINYINT instead of ENUM (0=sent, 1=delivered, 2=read)
    created_time BIGINT NOT NULL,
    updated_time BIGINT NOT NULL,
    INDEX idx_sender_receiver (sender_id, receiver_id),
    INDEX idx_chat (chat_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;



