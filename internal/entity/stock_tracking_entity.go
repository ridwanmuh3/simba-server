package entity

import "gorm.io/gorm"

type StockTracking struct {
	gorm.Model
	Type          string `gorm:"size:10;not null"`
	Amount        int    `gorm:"not null"`
	PreviousStock int    `gorm:"not null"`
	NewStock      int    `gorm:"not null"`
	Supplier      string `gorm:"size:100;not null"`
	ModifiedBy    string `gorm:"size:50;not null"`
	ItemID        string `gorm:"size:30;not null;index"`
	Item          Item   `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (s *StockTracking) TableName() string {
	return "stock_tracking"
}
