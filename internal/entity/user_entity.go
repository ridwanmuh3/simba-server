package entity

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username         string `gorm:"size:30;uniqueIndex;not null"`
	Fullname         string `gorm:"size:50;not null"`
	Role             string `gorm:"size:20;not null"`
	Password         string `gorm:"size:255;not null"`
	Token            string `gorm:"size:64;index"`
	TokenExpiresAt   time.Time
	RefreshToken     string `gorm:"size:64;index"`
	RefreshExpiresAt time.Time
	IsActive         bool `gorm:"type:boolean;default:false"`
	LastActive       time.Time
}

func (u *User) TableName() string {
	return "users"
}

type UserStats struct {
	Total      int64 `gorm:"column:total"`
	SuperAdmin int64 `gorm:"column:super_admin"`
	Admin      int64 `gorm:"column:admin"`
	Active     int64 `gorm:"column:active"`
}
