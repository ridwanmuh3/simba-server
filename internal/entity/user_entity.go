package entity

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string `gorm:"column:email;size:30;uniqueIndex;not null"`
	Fullname   string `gorm:"column:fullname;size:50;not null"`
	Role       string `gorm:"column:role;size:20;not null"`
	Password   string `gorm:"column:password;size:255;not null"`
	Token      string `gorm:"column:token;size:255;index"`
	IsActive   bool   `gorm:"column:is_active;type:boolean;default:false"`
	LastActive time.Time
}

func (u *User) TableName() string {
	return "users"
}
