package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

type InvoiceRepository struct{}

func NewInvoiceRepository() *InvoiceRepository {
	return &InvoiceRepository{}
}

type InvoiceHistoryRow struct {
	ID              uint           `gorm:"column:id"`
	StockType       string         `gorm:"column:stock_type"`
	CompanyName     string         `gorm:"column:company_name"`
	CompanyContact  string         `gorm:"column:company_contact"`
	CompanyAddress  string         `gorm:"column:company_address"`
	InvoiceNumber   string         `gorm:"column:invoice_number"`
	PONumber        string         `gorm:"column:po_number"`
	Kebutuhan       string         `gorm:"column:kebutuhan"`
	ReceiverName    string         `gorm:"column:receiver_name"`
	ReceiverAddress string         `gorm:"column:receiver_address"`
	InvoiceDate     string         `gorm:"column:invoice_date"`
	Keterangan      string         `gorm:"column:keterangan"`
	Penanggungjawab string         `gorm:"column:penanggungjawab"`
	Jabatan         string         `gorm:"column:jabatan"`
	BankAccount     string         `gorm:"column:bank_account"`
	HasItems        bool           `gorm:"column:has_items"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
	GrandTotal      float64        `gorm:"column:grand_total"`
}

func (r *InvoiceRepository) FindById(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error) {
	var invoice entity.Invoice
	err := db.Where("id = ? AND dapur_id = ?", id, dapurID).First(&invoice).Error
	return &invoice, err
}

func (r *InvoiceRepository) FindByIdWithItems(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error) {
	var invoice entity.Invoice
	err := db.Preload("Items").Where("id = ? AND dapur_id = ?", id, dapurID).First(&invoice).Error
	return &invoice, err
}

func (r *InvoiceRepository) FindAll(db *gorm.DB, dapurID uint, searchQuery, startDate, endDate string, page, size int) ([]InvoiceHistoryRow, int64, error) {
	query := db.Table("invoices").Where("dapur_id = ?", dapurID)

	if searchQuery != "" {
		search := "%" + searchQuery + "%"
		query = query.Where(
			"invoice_number LIKE ? OR company_name LIKE ? OR company_contact LIKE ? OR po_number LIKE ? OR kebutuhan LIKE ?",
			search, search, search, search, search,
		)
	}

	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	selectColumns := []string{
		"invoices.id",
		"invoices.stock_type",
		"invoices.company_name",
		"invoices.company_contact",
		"invoices.company_address",
		"invoices.invoice_number",
		"invoices.po_number",
		"invoices.kebutuhan",
		"invoices.receiver_name",
		"invoices.receiver_address",
		"invoices.invoice_date",
		"invoices.keterangan",
		"invoices.penanggungjawab",
		"invoices.jabatan",
		"invoices.bank_account",
		"invoices.grand_total",
		"invoices.created_at",
		"invoices.updated_at",
		"EXISTS (SELECT 1 FROM invoice_items WHERE invoice_items.invoice_id = invoices.id) AS has_items",
	}

	var rows []InvoiceHistoryRow
	if err := query.
		Select(strings.Join(selectColumns, ", ")).
		Order("invoices.created_at DESC, invoices.id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *InvoiceRepository) Save(db *gorm.DB, invoice *entity.Invoice) error {
	return db.Create(invoice).Error
}

func (r *InvoiceRepository) SaveItems(db *gorm.DB, items []entity.InvoiceItem) error {
	if len(items) == 0 {
		return nil
	}
	return db.Create(&items).Error
}

func (r *InvoiceRepository) DeleteItemsByInvoiceID(db *gorm.DB, invoiceID uint) error {
	return db.Where("invoice_id = ?", invoiceID).Delete(&entity.InvoiceItem{}).Error
}

func (r *InvoiceRepository) Delete(db *gorm.DB, id uint, dapurID uint) error {
	return db.Where("id = ? AND dapur_id = ?", id, dapurID).Delete(&entity.Invoice{}).Error
}
