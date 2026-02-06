package model

import "time"

type ItemResponse struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Category    string  `json:"category,omitempty"`
	Stock       int     `json:"stock,omitempty"`
	MeasureUnit string  `json:"measure_unit,omitempty"`
	UnitPrice   float64 `json:"unit_price,omitempty"`
	TotalPrice  float64 `json:"total_price,omitempty"`
}

type StockResponse struct {
	ID            int          `json:"id,omitempty"`
	Type          string       `json:"type,omitempty"`
	Amount        int          `json:"amount,omitempty"`
	PreviousStock int          `json:"previous_stock,omitempty"`
	NewStock      int          `json:"new_stock,omitempty"`
	Supplier      string       `json:"supplier,omitempty"`
	ModifiedBy    string       `json:"modified_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at,omitzero"`
	Item          ItemResponse `json:"item,omitzero"`
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
	UnitPrice   float64 `json:"unit_price,omitempty" validate:"omitempty,numeric,gt=0"`
	ModifiedBy  string  `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type UpdateItemStockRequest struct {
	ID         string `param:"id,omitempty" validate:"omitempty,printascii,max=30"`
	Type       string `json:"type" validate:"required,alpha,min=2,max=10"`
	Amount     int    `json:"amount" validate:"required,numeric,min=0"`
	Supplier   string `json:"supplier" validate:"omitempty,alphaspace,max=100"`
	ModifiedBy string `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type DeleteItemRequest struct {
	ID string `param:"id" validate:"required,printascii"`
}

type DeleteStockRequest struct {
	ID      string `param:"id" validate:"required,printascii"`
	StockID int    `param:"stock_id" validate:"required,number"`
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
