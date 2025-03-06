CREATE TABLE user_profiles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    patient_name VARCHAR(100) NOT NULL COMMENT '患者昵称 (Patient nickname)',
    relationship_to_patient VARCHAR(50) NOT NULL COMMENT '与患者关系 (Relationship to patient)',
    illness_cause VARCHAR(200) COMMENT '患病主要原因 (Main cause of illness)',
    chat_background VARCHAR(255) COMMENT '聊天背景 (Chat background image path)',
    user_avatar VARCHAR(255) COMMENT '用户头像 (User avatar image path)',
    user_nickname VARCHAR(100) COMMENT '用户昵称 (User nickname)',
    mobile_number VARCHAR(20) NOT NULL COMMENT '绑定手机 (Bound mobile number)',
    password VARCHAR(255) NOT NULL COMMENT 'Hashed password',
    created_at BIGINT NOT NULL COMMENT 'Creation timestamp in milliseconds since epoch',
    updated_at BIGINT NOT NULL COMMENT 'Last update timestamp in milliseconds since epoch'
);

-- Index for faster queries by mobile number
CREATE INDEX idx_mobile_number ON user_profiles(mobile_number);
