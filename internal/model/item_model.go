package model

import "time"

type ItemResponse struct {
	ID           string  `json:"id,omitempty"`
	Name         string  `json:"name,omitempty"`
	Category     string  `json:"category,omitempty"`
	Stock        int     `json:"stock,omitempty"`
	InitialStock int     `json:"initial_stock,omitempty"`
	MeasureUnit  string  `json:"measure_unit,omitempty"`
	UnitPrice    float64 `json:"unit_price,omitempty"`
	TotalPrice   float64 `json:"total_price,omitempty"`
}

type StockResponse struct {
	ID            int          `json:"id,omitempty"`
	Type          string       `json:"type,omitempty"`
	Amount        int          `json:"amount,omitempty"`
	PreviousStock int          `json:"previous_stock,omitempty"`
	NewStock      int          `json:"new_stock,omitempty"`
	UnitPrice     float64      `json:"unit_price,omitempty"`
	TotalPrice    float64      `json:"total_price,omitempty"`
	Supplier      string       `json:"supplier,omitempty"`
	ModifiedBy    string       `json:"modified_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at,omitzero"`
	Item          ItemResponse `json:"item,omitzero"`
}

type StocksFinanceSummaryResponse struct {
	MasterItemsTotalBudget float64 `json:"master_items_total_budget,omitempty"`
	BudgetIn               float64 `json:"budget_in,omitempty"`
	BudgetOut              float64 `json:"budget_out,omitempty"`
	Profit                 float64 `json:"profit,omitempty"`
	CurrentBudget          float64 `json:"current_budget,omitempty"`
}

type ItemStocksSummaryResponse struct {
	ItemID       string  `json:"item_id,omitempty"`
	Name         string  `json:"name,omitempty"`
	Category     string  `json:"category,omitempty"`
	InitialStock int     `json:"initial_stock,omitempty"`
	MeasureUnit  string  `json:"measure_unit,omitempty"`
	TotalIn      int     `json:"total_in,omitempty"`
	TotalOut     int     `json:"total_out,omitempty"`
	CurrentStock int     `json:"current_stock,omitempty"`
	StockValue   float64 `json:"stock_value,omitempty"`
}

type InvoiceData struct {
	// Data Perusahaan (Kiri Atas)
	CompanyName    string
	CompanyAddress string
	CompanyContact string
	LogoPath       string // Contoh: "./uploads/logo.png"

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
}

type AddItemRequest struct {
	Name        string  `json:"name" validate:"required,printascii,min=3,max=90"`
	Category    string  `json:"category" validate:"required,alphaspace,min=3,max=30"`
	Stock       int     `json:"stock" validate:"required,numeric,gt=0"`
	MeasureUnit string  `json:"measure_unit" validate:"required,printascii,min=1,max=30"`
	UnitPrice   float64 `json:"unit_price" validate:"required,numeric,gt=0"`
	ModifiedBy  string  `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type AddItemBatchRequest struct {
	Items []AddItemRequest `validate:"required"`
}

type UpdateItemRequest struct {
	ID          string  `param:"id,omitempty" validate:"omitempty,printascii,max=30"`
	Name        string  `json:"name,omitempty" validate:"omitempty,printascii,min=3,max=90"`
	Category    string  `json:"category,omitempty" validate:"omitempty,alphaspace,min=3,max=30"`
	MeasureUnit string  `json:"measure_unit,omitempty" validate:"omitempty,printascii,min=1,max=30"`
	UnitPrice   float64 `json:"unit_price,omitempty" validate:"omitempty,number,gt=0"`
	ModifiedBy  string  `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type UpdateItemStockRequest struct {
	ID         string  `param:"id,omitempty" validate:"omitempty,printascii,max=30"`
	Type       string  `json:"type" validate:"required,alpha,min=2,max=10"`
	Amount     int     `json:"amount" validate:"required,numeric,min=0"`
	UnitPrice  float64 `json:"unit_price" validate:"required,numeric,gt=1"`
	Supplier   string  `json:"supplier" validate:"omitempty,alphaspace,max=100"`
	ModifiedBy string  `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type DeleteItemRequest struct {
	ID string `param:"id" validate:"required,printascii"`
}

type DeleteStockRequest struct {
	ID      string `param:"id" validate:"required,printascii"`
	StockID int    `param:"stock_id" validate:"required,numeric"`
}

type FindByIdItemRequest struct {
	ID string `param:"id" validate:"required,printascii"`
}

type FindAllItemsRequest struct {
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
}

type FindAllStocksRequest struct {
	Type        string `query:"type,omitempty" validate:"omitempty,max=10"`
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
}

type GetInvoiceItemsRequest struct {
	Filename string `query:"filename,omitempty" validate:"omitempty,alphanum,max=100"`
	DateFrom string `query:"date_from,omitempty" validate:"omitempty,datetime"`
	DateTo   string `query:"date_to,omitempty" validate:"omitempty,datetime"`
}

type GetItemStockSummaryRequest struct {
	StartDate string `query:"start_date,omitempty" validate:"omitempty,datetime"`
	EndDate   string `query:"end_date,omitempty" validate:"omitempty,datetime"`
	Page      int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size      int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
}
