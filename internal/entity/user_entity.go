package entity

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string `gorm:"size:30;uniqueIndex;not null"`
	Fullname   string `gorm:"size:50;not null"`
	Role       string `gorm:"size:20;not null"`
	Password   string `gorm:"size:255;not null"`
	Token      string `gorm:"size:255;index"`
	IsActive   bool   `gorm:"type:boolean;default:false"`
	LastActive time.Time
}

func (u *User) TableName() string {
	return "users"
}
