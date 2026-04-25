package model

import "time"

type FinanceResponse struct {
	ID          int       `json:"id,omitempty"`
	Type        string    `json:"type,omitempty"`
	Category    string    `json:"category,omitempty"`
	Description string    `json:"description,omitempty"`
	Amount      int       `json:"amount,omitempty"`
	ExtraNote   string    `json:"extra_note,omitempty"`
	ProofImage  string    `json:"proof_image,omitempty"`
	ModifiedBy  string    `json:"modified_by,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
}

type AddFinanceRequest struct {
	Type        string `json:"type" validate:"required,alpha"`
	Category    string `json:"category" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"required,min=1,max=255"`
	Amount      int    `json:"amount" validate:"required,number"`
	ExtraNote   string `json:"extra_note,omitempty" validate:"omitempty,max=255"`
	ProofImage  string `json:"-" validate:"required"`
	ModifiedBy  string `json:"-" validate:"required,min=2,max=50"`
	DapurID     uint   `json:"-" validate:"required,min=1"`
}

type UpdateFinanceRequest struct {
	ID          int    `param:"id" validate:"required,number"`
	Type        string `json:"type" validate:"required,alpha"`
	Category    string `json:"category,omitempty" validate:"omitempty,min=1,max=100"`
	Description string `json:"description,omitempty" validate:"omitempty,min=1,max=255"`
	Amount      int    `json:"amount,omitempty" validate:"omitempty,number"`
	ExtraNote   string `json:"extra_note,omitempty" validate:"omitempty,max=255"`
	ProofImage  string `json:"-" validate:"omitempty"`
	ModifiedBy  string `json:"-" validate:"required,min=2,max=50"`
	DapurID     uint   `json:"-" validate:"required,min=1"`
}

type DeleteFinanceRequest struct {
	ID      int  `param:"id" validate:"required,number"`
	DapurID uint `json:"-" validate:"required,min=1"`
}

type FindByIdFinanceRequest struct {
	ID      int  `param:"id" validate:"required,number"`
	DapurID uint `json:"-" validate:"required,min=1"`
}

type FindAllFinanceRequest struct {
	SearchQuery string `query:"search_query,omitempty" validate:"omitempty,max=30"`
	StartDate   string `query:"start_date,omitempty" validate:"omitempty"`
	EndDate     string `query:"end_date,omitempty" validate:"omitempty"`
	Page        int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size        int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	DapurID     uint   `json:"-"`
}
