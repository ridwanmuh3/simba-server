package entity

import "gorm.io/gorm"

type StockTracking struct {
	gorm.Model
	Type          string  `gorm:"size:10;not null"`
	Amount        float64     `gorm:"not null"`
	PreviousStock float64     `gorm:"not null"`
	NewStock      float64     `gorm:"not null"`
	UnitPrice     float64 `gorm:"not null"`
	TotalPrice    float64 `gorm:"not null"`
	Supplier      string  `gorm:"size:100;not null"`
	ModifiedBy    string  `gorm:"size:50;not null"`
	DapurID       uint    `gorm:"not null;index"`
	ItemID        string  `gorm:"size:30;not null;index"`
	Item          Item    `gorm:"foreignKey:ItemID;references:ID;constraint:OnUpdate:CASCADE;OnDelete:CASCADE"`
}

func (s *StockTracking) TableName() string {
	return "stock_tracks"
}
