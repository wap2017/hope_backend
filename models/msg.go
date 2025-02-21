package models

type Message struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	SenderID    uint   `gorm:"not null" json:"sender_id"`
	ReceiverID  uint   `gorm:"not null" json:"receiver_id"`
	ChatID      string `gorm:"not null" json:"chat_id"`
	Content     string `gorm:"type:varchar(2000);not null" json:"content"`
	MsgType     uint8  `gorm:"not null;default:1" json:"msg_type"` // 1=text, 2=image, etc.
	Status      uint8  `gorm:"not null;default:0" json:"status"`   // 0=sent, 1=delivered, 2=read
	CreatedTime int64  `gorm:"autoCreateTime" json:"created_time"`
	UpdatedTime int64  `gorm:"autoUpdateTime" json:"updated_time"`
}
