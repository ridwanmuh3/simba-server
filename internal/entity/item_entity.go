package entity

import (
	"time"

	"gorm.io/gorm"
)

type Item struct {
	ID           string  `gorm:"size:30;primaryKey;index;not null"`
	Name         string  `gorm:"size:90;uniqueIndex;not null"`
	Category     string  `gorm:"size:30;not null"`
	InitialStock int     `gorm:"not null"`
	Stock        int     `gorm:"not null"`
	MeasureUnit  string  `gorm:"size:30;not null"`
	UnitPrice    float64 `gorm:"not null"`
	TotalPrice   float64 `gorm:"not null"`
	ModifiedBy   string  `gorm:"size:50;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (i *Item) TableName() string {
	return "items"
}
