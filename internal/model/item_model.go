package model

type ItemResponse struct {
	ID          string  `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Category    string  `json:"category,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
	MeasureUnit string  `json:"measure_unit,omitempty"`
	UnitPrice   float64 `json:"unit_price,omitempty"`
	TotalPrice  float64 `json:"total_price,omitempty"`
}

type AddItemRequest struct {
	Name        string  `json:"name" validate:"required,printascii,min=3,max=90"`
	Category    string  `json:"category" validate:"required,alphaspace,min=3,max=30"`
	Quantity    int     `json:"quantity" validate:"required,numeric,gt=0"`
	MeasureUnit string  `json:"measure_unit" validate:"required,printascii,min=3,max=30"`
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
	Quantity    int     `json:"quantity,omitempty" validate:"omitempty,numeric,gt=0"`
	MeasureUnit string  `json:"measure_unit,omitempty" validate:"omitempty,printascii,min=3,max=30"`
	UnitPrice   float64 `json:"unit_price,omitempty" validate:"omitempty,numeric,gt=0"`
	ModifiedBy  string  `json:"-" validate:"required,alphaspace,min=2,max=50"`
}

type DeleteItemRequest struct {
	ID string `param:"id" validate:"required,printascii"`
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
