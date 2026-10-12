package entity

import (
	"gorm.io/gorm"
)

type ActivityLog struct {
	gorm.Model
	Type        string `gorm:"size:30;not null"`
	Title       string `gorm:"size:100;not null"`
	Description string `gorm:"size:100;not null"`
	ActionBy    string `gorm:"size:100;not null"`
	DapurID     uint   `gorm:"not null;index"`
}

func (l *ActivityLog) TableName() string {
	return "activity_logs"
}
