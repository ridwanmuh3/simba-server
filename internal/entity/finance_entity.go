package entity

import (
	"gorm.io/gorm"
)

type Finance struct {
	gorm.Model
	Type        string `gorm:"size:20;not null"`
	Category    string `gorm:"size:30;not null"`
	Description string `gorm:"size:100;not null"`
	Amount      int    `gorm:"not null"`
	ExtraNote   string `gorm:"type:text"`
	ProofImage  string `gorm:"size:100;not null"`
	ModifiedBy  string `gorm:"size:100;not null"`
}

func (f *Finance) TableName() string {
	return "finances"
}
