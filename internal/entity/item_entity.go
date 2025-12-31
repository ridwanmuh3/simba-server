package entity

import (
	"gorm.io/gorm"
)

type Item struct {
	gorm.Model
	ItemCode   string  `gorm:"column:item_code;size:30;uniqueIndex;index;not null"`
	Name       string  `gorm:"column:name;size:90;not null"`
	Category   string  `gorm:"column:category;size:30;not null"`
	Quantity   int     `gorm:"column:quantity;not null"`
	UnitPrice  float64 `gorm:"column:unit_price;not null"`
	TotalPrice float64 `gorm:"column:total_price;not null"`
}

func (i *Item) TableName() string {
	return "items"
}
