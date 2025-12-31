package entity

import (
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ItemCode   string  `gorm:"size:30;uniqueIndex;index;not null"`
	Name       string  `gorm:"size:90;not null"`
	Category   string  `gorm:"size:30;not null"`
	Quantity   int     `gorm:"not null"`
	UnitPrice  float64 `gorm:"not null"`
	TotalPrice float64 `gorm:"not null"`
	CreatedBy  string  `gorm:"size:50;not null"`
}

func (i *Item) TableName() string {
	return "items"
}
