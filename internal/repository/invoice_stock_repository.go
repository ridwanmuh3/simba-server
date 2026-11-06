package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// FindByIdForUpdate locks the invoice row, scoped to dapur.
func (r *InvoiceRepository) FindByIdForUpdate(db *gorm.DB, id uint, dapurID uint) (*entity.Invoice, error) {
	var invoice entity.Invoice
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Scopes(Tenant(dapurID)).
		Where("id = ?", id).
		First(&invoice).Error
	return &invoice, err
}

// Update persists a mutated invoice (create-or-update via GORM Save).
func (r *InvoiceRepository) Update(db *gorm.DB, invoice *entity.Invoice) error {
	return db.Save(invoice).Error
}

// FindStocksForInvoice loads stock tracks (with their item) for invoice
// generation, filtered by type, optional IDs and date range, tenant-scoped.
func (r *InvoiceRepository) FindStocksForInvoice(db *gorm.DB, stockType string, stockIDs []uint, dateFrom, dateTo string, dapurID uint) ([]entity.StockTracking, error) {
	var stocks []entity.StockTracking
	query := db.Model(new(entity.StockTracking)).Preload("Item").
		Scopes(Tenant(dapurID)).
		Where("type = ?", stockType)

	if len(stockIDs) > 0 {
		query = query.Where("id IN ?", stockIDs)
	} else {
		if dateFrom != "" {
			parsedFrom, err := time.Parse(time.RFC3339Nano, dateFrom)
			if err != nil {
				return nil, err
			}
			query = query.Where("created_at >= ?", parsedFrom.UTC())
		}
		if dateTo != "" {
			parsedTo, err := time.Parse(time.RFC3339Nano, dateTo)
			if err != nil {
				return nil, err
			}
			query = query.Where("created_at <= ?", parsedTo.UTC())
		}
	}

	if err := query.Order("item_id ASC").Limit(500).Find(&stocks).Error; err != nil {
		return nil, err
	}
	return stocks, nil
}

// FindStocksByIDs loads tracks (with item) by ID, tenant-scoped.
func (r *InvoiceRepository) FindStocksByIDs(db *gorm.DB, stockIDs []int, dapurID uint) ([]entity.StockTracking, error) {
	var stocks []entity.StockTracking
	err := db.Preload("Item").
		Scopes(Tenant(dapurID)).
		Where("id IN ?", stockIDs).
		Find(&stocks).Error
	return stocks, err
}

// InvoiceItemFlatRow is one row of the flattened invoice-items report.
type InvoiceItemFlatRow struct {
	Date           time.Time `gorm:"column:date"`
	InvoiceNumber  string    `gorm:"column:invoice_number"`
	ItemName       string    `gorm:"column:item_name"`
	MeasureUnit    string    `gorm:"column:measure_unit"`
	Amount         float64   `gorm:"column:amount"`
	StockType      string    `gorm:"column:stock_type"`
	BuyPrice       float64   `gorm:"column:buy_price"`
	SellPrice      float64   `gorm:"column:sell_price"`
	TotalBuyPrice  float64   `gorm:"column:total_buy_price"`
	TotalSellPrice float64   `gorm:"column:total_sell_price"`
}

// FindItemsFlat returns one page of the flattened invoice-items report.
func (r *InvoiceRepository) FindItemsFlat(db *gorm.DB, dapurID uint, stockType, searchQuery, startDate, endDate string, page, size int) ([]InvoiceItemFlatRow, int64, error) {
	base := db.Table("invoice_items").
		Joins("JOIN invoices ON invoice_items.invoice_id = invoices.id").
		Scopes(TenantColumn("invoices", dapurID))

	if stockType != "" {
		base = base.Where("invoice_items.stock_type = ?", stockType)
	}
	if searchQuery != "" {
		base = base.Where("invoice_items.item_name LIKE ?", "%"+searchQuery+"%")
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if startDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, startDate); err == nil {
			p := parsed.In(loc)
			base = base.Where("invoice_items.created_at >= ?", time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, loc).UTC())
		}
	}
	if endDate != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, endDate); err == nil {
			p := parsed.In(loc)
			base = base.Where("invoice_items.created_at <= ?", time.Date(p.Year(), p.Month(), p.Day(), 23, 59, 59, 999999999, loc).UTC())
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []InvoiceItemFlatRow
	err := base.
		Select(`
			invoice_items.created_at AS date,
			invoices.invoice_number,
			invoice_items.item_name,
			invoice_items.measure_unit,
			invoice_items.amount,
			invoice_items.stock_type,
			CASE WHEN invoice_items.stock_type = 'IN'
				THEN invoice_items.unit_price
				ELSE COALESCE((
					SELECT AVG(st.unit_price) FROM stock_tracks st
					WHERE st.item_id = invoice_items.item_id AND st.type = 'IN'
					AND st.deleted_at IS NULL AND st.dapur_id = invoices.dapur_id
				), COALESCE(items.unit_price, 0))
			END AS buy_price,
			CASE WHEN invoice_items.stock_type = 'OUT'
				THEN invoice_items.unit_price
				ELSE COALESCE(items.unit_price, 0)
			END AS sell_price,
			CASE WHEN invoice_items.stock_type = 'IN'
				THEN invoice_items.total_price
				ELSE invoice_items.amount * COALESCE((
					SELECT AVG(st.unit_price) FROM stock_tracks st
					WHERE st.item_id = invoice_items.item_id AND st.type = 'IN'
					AND st.deleted_at IS NULL AND st.dapur_id = invoices.dapur_id
				), COALESCE(items.unit_price, 0))
			END AS total_buy_price,
			CASE WHEN invoice_items.stock_type = 'OUT'
				THEN invoice_items.total_price
				ELSE invoice_items.amount * COALESCE(items.unit_price, 0)
			END AS total_sell_price
		`).
		Joins("LEFT JOIN items ON invoice_items.item_id = items.id AND items.deleted_at IS NULL").
		Order("invoices.created_at DESC, invoice_items.id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&rows).Error
	return rows, total, err
}
