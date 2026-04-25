package entity

import "gorm.io/gorm"

type Dapur struct {
	gorm.Model
	Name        string `gorm:"size:100;uniqueIndex;not null"`
	Description string `gorm:"size:255"`
	IsActive    bool   `gorm:"default:true"`
}

func (d *Dapur) TableName() string {
	return "dapurs"
}
