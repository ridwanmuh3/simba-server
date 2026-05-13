package entity

import "time"

// Invoice represents the data model for the invoice generation form.
type Invoice struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	DapurID         uint   `gorm:"not null;index;uniqueIndex:uidx_inv_no_type;uniqueIndex:uidx_po_no_type"`
	StockType       string `gorm:"type:varchar(10);not null;uniqueIndex:uidx_inv_no_type;uniqueIndex:uidx_po_no_type"`
	CompanyName     string `gorm:"type:varchar(255);not null"`
	CompanyContact  string `gorm:"type:varchar(50);not null"`
	CompanyAddress  string `gorm:"type:text;not null"`
	InvoiceNumber   string `gorm:"type:varchar(100);not null;uniqueIndex:uidx_inv_no_type"`
	PONumber        string `gorm:"type:varchar(100);uniqueIndex:uidx_po_no_type,where:po_number <> ''"`
	Kebutuhan       string `gorm:"type:varchar(100)"`
	ReceiverName    string `gorm:"type:varchar(255)"`
	ReceiverAddress string `gorm:"type:text"`
	InvoiceDate     string `gorm:"type:varchar(100)"`
	Keterangan      string `gorm:"type:text"`
	Penanggungjawab string `gorm:"type:varchar(255)"`
	Jabatan         string `gorm:"type:varchar(255)"`
	BankAccount     string `gorm:"type:varchar(100)"`
	GrandTotal      float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Items           []InvoiceItem `gorm:"foreignKey:InvoiceID;references:ID;constraint:OnUpdate:CASCADE;OnDelete:CASCADE"`
}

// InvoiceItem stores the line items attached to a saved invoice so the PDF can
// be regenerated on demand from the stored snapshot.
type InvoiceItem struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	InvoiceID     uint   `gorm:"not null;index"`
	ItemID        string `gorm:"type:varchar(30)"`
	ItemName      string `gorm:"type:varchar(255);not null"`
	Category      string `gorm:"type:varchar(100)"`
	MeasureUnit   string `gorm:"type:varchar(30)"`
	Amount        float64
	UnitPrice     float64
	TotalPrice    float64
	PreviousStock float64
	NewStock      float64
	Supplier      string `gorm:"type:varchar(100)"`
	StockType     string `gorm:"type:varchar(10)"`
	CreatedAt     time.Time
}
