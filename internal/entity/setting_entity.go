package entity

import "time"

// AppSetting stores per-dapur configuration as key-value pairs.
type AppSetting struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Key       string `gorm:"type:varchar(100);uniqueIndex:uidx_setting_key_dapur;not null" json:"key"`
	Value     string `gorm:"type:text" json:"value"`
	DapurID   uint   `gorm:"not null;uniqueIndex:uidx_setting_key_dapur" json:"dapur_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
