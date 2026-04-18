package entity

import "time"

// Invoice represents the data model for the invoice generation form.
type Invoice struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	StockType      string `gorm:"type:varchar(10);not null"  `
	CompanyName    string `gorm:"type:varchar(255);not null"  `
	CompanyContact string `gorm:"type:varchar(50);not null"  `
	CompanyAddress string `gorm:"type:text;not null"  `
	InvoiceNumber  string `gorm:"type:varchar(100);not null;uniqueIndex"  `
	PONumber       string `gorm:"type:varchar(100)" `
	QuoNumber      string `gorm:"type:varchar(100)" `
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
