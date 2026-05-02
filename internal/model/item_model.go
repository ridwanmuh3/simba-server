package model

import (
	"time"
)

type ItemResponse struct {
	ID           string    `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	Category     string    `json:"category,omitempty"`
	Stock        float64   `json:"stock,omitempty"`
	InitialStock float64   `json:"initial_stock,omitempty"`
	MeasureUnit  string    `json:"measure_unit,omitempty"`
	UnitPrice    float64   `json:"unit_price,omitempty"`
	TotalPrice   float64   `json:"total_price,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitzero"`
}

type StockResponse struct {
	ID            int          `json:"id,omitempty"`
	Type          string       `json:"type,omitempty"`
	Amount        float64      `json:"amount,omitempty"`
	PreviousStock float64      `json:"previous_stock"`
	NewStock      float64      `json:"new_stock"`
	UnitPrice     float64      `json:"unit_price,omitempty"`
	TotalPrice    float64      `json:"total_price,omitempty"`
	Supplier      string       `json:"supplier,omitempty"`
	ModifiedBy    string       `json:"modified_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at,omitzero"`
	Item          ItemResponse `json:"item,omitzero"`
}

type StocksFinanceSummaryResponse struct {
	MasterItemsTotalBudget float64 `json:"master_items_total_budget"`
	BudgetIn               float64 `json:"budget_in"`
	BudgetOut              float64 `json:"budget_out"`
	Profit                 float64 `json:"profit"`
	CurrentBudget          float64 `json:"current_budget"`
}

type ItemStocksSummaryResponse struct {
	ItemID       string  `json:"item_id,omitempty"`
	Name         string  `json:"name,omitempty"`
	Category     string  `json:"category,omitempty"`
	InitialStock float64 `json:"initial_stock"`
	MeasureUnit  string  `json:"measure_unit,omitempty"`
	TotalIn      float64 `json:"total_in"`
	TotalOut     float64 `json:"total_out"`
	CurrentStock float64 `json:"current_stock"`
	BuyPrice     float64 `json:"buy_price"`  // Ditambahkan
	SellPrice    float64 `json:"sell_price"` // Ditambahkan
	StockValue   float64 `json:"stock_value"`
}

type InvoiceSummary struct {
	Items      []StockResponse `json:"items"`
	GrandTotal float64         `json:"grand_total"`
}

type InvoiceResponse struct {
	ID              uint      `json:"id"`
	StockType       string    `json:"stock_type"`
	CompanyName     string    `json:"company_name"`
	CompanyContact  string    `json:"company_contact"`
	CompanyAddress  string    `json:"company_address"`
	InvoiceNumber   string    `json:"invoice_number"`
	PONumber        string    `json:"po_number"`
	QuoNumber       string    `json:"quo_number"`
	ReceiverName    string    `json:"receiver_name"`
	ReceiverAddress string    `json:"receiver_address"`
	InvoiceDate     string    `json:"invoice_date"`
	Keterangan      string    `json:"keterangan"`
	Penanggungjawab string    `json:"penanggungjawab"`
	Jabatan         string    `json:"jabatan"`
	BankAccount     string    `json:"bank_account"`
	HasItems        bool      `json:"has_items"`
	CreatedAt       time.Time `json:"created_at,omitzero"`
	UpdatedAt       time.Time `json:"updated_at,omitzero"`
}

type InvoiceItemResponse struct {
	ID          uint    `json:"id"`
	ItemName    string  `json:"item_name"`
	Category    string  `json:"category"`
	MeasureUnit string  `json:"measure_unit"`
	Amount      float64 `json:"amount"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
	Supplier    string  `json:"supplier"`
	StockType   string  `json:"stock_type"`
}

type InvoiceDetailResponse struct {
	InvoiceResponse
	Items      []InvoiceItemResponse `json:"items"`
	GrandTotal float64               `json:"grand_total"`
}

type UpdateInvoiceRequest struct {
	ID              uint   `param:"id"`
	DapurID         uint   `json:"-" validate:"required,min=1"`
	CompanyName     string `json:"company_name" validate:"required,min=2,max=100"`
	CompanyAddress  string `json:"company_address" validate:"required,min=2,max=200"`
	CompanyContact  string `json:"company_contact" validate:"required,min=2,max=100"`
	ReceiverName    string `json:"receiver_name" validate:"omitempty,max=100"`
	ReceiverAddress string `json:"receiver_address" validate:"omitempty,max=200"`
	InvoiceDate     string `json:"invoice_date" validate:"omitempty,max=100"`
	Keterangan      string `json:"keterangan" validate:"omitempty,max=500"`
	Penanggungjawab string `json:"penanggungjawab" validate:"omitempty,max=100"`
	Jabatan         string `json:"jabatan" validate:"omitempty,max=100"`
	BankAccount     string `json:"bank_account" validate:"omitempty,max=100"`
	StockIDs        []int  `json:"stock_ids"`
}

type InvoiceData struct {
	StockType string

	// Data Perusahaan (Kiri Atas)
	CompanyName    string
	CompanyAddress string
	CompanyContact string

	// Data Invoice (Kanan Atas)
	InvoiceNo string
	Date      string
	PONo      string
	QuoNo     string

	// Data Penerima (Tengah)
	ReceiverName    string
	ReceiverAddress string

	// Tabel & Total
	Items      []StockResponse
	GrandTotal float64

	// Footer
	Keterangan      string
	Penanggungjawab string
	Jabatan         string
	BankAccount     string
}
type GenerateInvoiceRequest struct {
	CompanyName    string `json:"company_name" validate:"required,min=2,max=100"`
	CompanyAddress string `json:"company_address" validate:"required,min=2,max=200"`
	CompanyContact string `json:"company_contact" validate:"required,min=2,max=100"`

	InvoiceNo string `json:"invoice_no" validate:"required,min=1,max=50"`
	Date      string `json:"date"`

	PONo  string `json:"po_no" validate:"omitempty,max=50"`
	QuoNo string `json:"quo_no" validate:"omitempty,max=50"`

	ReceiverName    string `json:"receiver_name" validate:"required,min=2,max=100"`
	ReceiverAddress string `json:"receiver_address" validate:"required,min=2,max=200"`

	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	StockIDs []uint `json:"stock_ids"`

	// StockType: "OUT" (default) or "IN" — determines which stock tab context to pull.
	StockType string `json:"stock_type" validate:"omitempty,oneof=IN OUT"`

	Keterangan      string `json:"keterangan" validate:"omitempty,max=500"`
	Penanggungjawab string `json:"penanggungjawab" validate:"required,min=2,max=100"`
	Jabatan         string `json:"jabatan" validate:"required,min=2,max=100"`
	BankAccount     string `json:"bank_account" validate:"omitempty,max=100"`
	DapurID         uint   `json:"-" validate:"required,min=1"`
}
type AddItemRequest struct {
	Name        string     `json:"name" validate:"required,min=1,max=90"`
	Category    string     `json:"category" validate:"required,min=1,max=50"`
	Stock       float64    `json:"stock" validate:"required,numeric,gt=0"`
	MeasureUnit string     `json:"measure_unit" validate:"required,min=1,max=30"`
	UnitPrice   float64    `json:"unit_price" validate:"required,numeric,gt=0"`
	DateAdded   *time.Time `json:"date_added,omitempty" validate:"omitempty"`
	ModifiedBy  string     `json:"-" validate:"required,min=2,max=50"`
	DapurID     uint       `json:"-" validate:"required,min=1"`
}

type AddItemBatchRequest struct {
	Items   []AddItemRequest `validate:"required"`
	DapurID uint             `json:"-" validate:"required,min=1"`
}

type UpdateItemRequest struct {
	ID           string     `param:"id,omitempty" validate:"omitempty,printascii,max=30"`
	Name         string     `json:"name,omitempty" validate:"omitempty,min=1,max=90"`
	Category     string     `json:"category,omitempty" validate:"omitempty,min=1,max=50"`
	MeasureUnit  string     `json:"measure_unit,omitempty" validate:"omitempty,min=1,max=30"`
	UnitPrice    float64    `json:"unit_price,omitempty" validate:"omitempty,numeric,gt=0"`
	Stock        float64    `json:"stock" validate:"omitempty,numeric,gt=0"`
	InitialStock *float64   `json:"initial_stock,omitempty" validate:"omitempty,numeric,gte=0"`
	DateAdded    *time.Time `json:"date_added,omitempty" validate:"omitempty"`
	ModifiedBy   string     `json:"-" validate:"required,min=2,max=50"`
	DapurID      uint       `json:"-" validate:"required,min=1"`
}

type UpdateItemStockRequest struct {
	ID         string  `param:"id,omitempty" validate:"omitempty,printascii,max=30"`
	Type       string  `json:"type" validate:"required,alpha,min=2,max=10"`
	Amount     float64 `json:"amount" validate:"required,numeric,gt=0"`
	UnitPrice  float64 `json:"unit_price" validate:"required,numeric,gte=1"`
	Supplier   string  `json:"supplier" validate:"omitempty,max=100"`
	ModifiedBy string  `json:"-" validate:"required,min=2,max=50"`
	DapurID    uint    `json:"-" validate:"required,min=1"`
}

type DeleteItemRequest struct {
	ID         string `param:"id" validate:"required,printascii"`
	ModifiedBy string `json:"-" validate:"required"`
	DapurID    uint   `json:"-" validate:"required,min=1"`
}

type DeleteStockRequest struct {
	ID         string `param:"id" validate:"required,printascii"`
	StockID    int    `param:"stock_id" validate:"required,numeric"`
	ModifiedBy string `json:"-" validate:"required"`
	DapurID    uint   `json:"-" validate:"required,min=1"`
}

type EditStockRequest struct {
	ID         string  `param:"id" validate:"required,printascii,max=30"`
	StockID    int     `param:"stock_id" validate:"required,numeric"`
	Amount     float64 `json:"amount" validate:"required,numeric,gt=0"`
	UnitPrice  float64 `json:"unit_price" validate:"required,numeric,gt=0"`
	Supplier   string  `json:"supplier" validate:"omitempty,max=100"`
	ModifiedBy string  `json:"-" validate:"required,min=2,max=50"`
	DapurID    uint    `json:"-" validate:"required,min=1"`
}

type FindByIdItemRequest struct {
	ID      string `param:"id" validate:"required,printascii"`
	DapurID uint   `json:"-" validate:"required,min=1"`
}

type FindAllItemsRequest struct {
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	DapurID     uint   `json:"-"`
}

type FindAllStocksRequest struct {
	Type        string `query:"type,omitempty" validate:"omitempty,max=10"`
	Category    string `query:"category,omitempty" validate:"omitempty,max=50"`
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	DapurID     uint   `json:"-"`
}

type GetInvoiceItemsRequest struct {
	DateFrom  string `query:"date_from,omitempty" validate:"omitempty,max=50"`
	DateTo    string `query:"date_to,omitempty" validate:"omitempty,max=50"`
	StockType string `query:"stock_type,omitempty" validate:"omitempty,oneof=IN OUT"`
	StockIDs  []uint
	DapurID   uint
}

type FindAllInvoicesRequest struct {
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=100"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	DapurID     uint   `json:"-"`
}

type GetItemStockSummaryRequest struct {
	StartDate   string `query:"start_date,omitempty" validate:"omitempty,max=50"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty,max=50"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	DapurID     uint   `json:"-"`
}

type InvoiceItemFlatResponse struct {
	Date           string  `json:"date"`
	InvoiceNumber  string  `json:"invoice_number"`
	ItemName       string  `json:"item_name"`
	MeasureUnit    string  `json:"measure_unit"`
	Amount         float64 `json:"amount"`
	StockType      string  `json:"stock_type"`
	BuyPrice       float64 `json:"buy_price"`
	SellPrice      float64 `json:"sell_price"`
	TotalBuyPrice  float64 `json:"total_buy_price"`
	TotalSellPrice float64 `json:"total_sell_price"`
}

type GetInvoiceItemsFlatRequest struct {
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=100"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	StockType   string `query:"stock_type,omitempty" validate:"omitempty,oneof=IN OUT"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=9999"`
	DapurID     uint   `json:"-"`
}
