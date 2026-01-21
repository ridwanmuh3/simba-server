package entity

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Item struct {
	ID          string  `gorm:"size:30;primaryKey;index;not null"`
	Name        string  `gorm:"size:90;not null"`
	Category    string  `gorm:"size:30;not null"`
	Quantity    int     `gorm:"not null"`
	MeasureUnit string  `gorm:"size:30;not null"`
	UnitPrice   float64 `gorm:"not null"`
	TotalPrice  float64 `gorm:"not null"`
	ModifiedBy  string  `gorm:"size:50;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (i *Item) TableName() string {
	return "items"
}

func (i *Item) BeforeCreate(tx *gorm.DB) error {
	var lastItem Item

	err := tx.Unscoped().Order("id DESC").First(&lastItem).Error
	newID := 1
	if err == nil {
		parts := strings.Split(lastItem.ID, "-")
		if len(parts) == 3 {
			lastNumber, _ := strconv.Atoi(parts[2])
			newID = lastNumber + 1
		}
	}

	i.ID = fmt.Sprintf("MBG-BHN-%04d", newID)

	return nil
}
