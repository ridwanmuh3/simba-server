package entity

import (
	"time"

	"gorm.io/gorm"
)

type Item struct {
	ID           string  `gorm:"size:30;primaryKey;index;not null"`
	Name         string  `gorm:"size:90;not null;uniqueIndex:idx_items_name_dapur"`
	Category     string  `gorm:"size:30;not null"`
	InitialStock float64 `gorm:"not null"`
	Stock        float64 `gorm:"not null"`
	MeasureUnit  string  `gorm:"size:30;not null"`
	UnitPrice    float64 `gorm:"not null"`
	TotalPrice   float64 `gorm:"not null"`
	ModifiedBy   string  `gorm:"size:50;not null"`
	DapurID      uint    `gorm:"not null;index;uniqueIndex:idx_items_name_dapur"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (i *Item) TableName() string {
	return "items"
}

func (i *Item) AfterDelete(tx *gorm.DB) (err error) {
	err = tx.Where("item_id = ?", i.ID).Delete(&StockTracking{}).Error

	if err != nil {
		return err
	}

	return nil
}
