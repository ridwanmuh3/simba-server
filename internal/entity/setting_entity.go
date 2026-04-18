package entity

import "time"

// AppSetting represents application-wide settings such as company profile and document sequences.
type AppSetting struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Key       string `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value     string `gorm:"type:text" json:"value"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
