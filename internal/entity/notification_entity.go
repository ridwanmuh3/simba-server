package entity

import (
	"gorm.io/gorm"
)

// Notification represents a persistent, broadcastable alert for a dapur.
// UserID nil = broadcast to all users in the dapur; otherwise targeted to one user.
type Notification struct {
	gorm.Model
	Type     string `gorm:"size:30;not null"`
	Title    string `gorm:"size:100;not null"`
	Message  string `gorm:"type:text;not null"`
	Level    string `gorm:"size:10;not null;default:'info'"`
	IsRead   bool   `gorm:"default:false"`
	UserID   *uint  `gorm:"index"`
	DapurID  uint   `gorm:"not null;index"`
	ActorID  *uint  `gorm:"index"`
	MetaJSON string `gorm:"type:text"`
}

func (n *Notification) TableName() string {
	return "notifications"
}
