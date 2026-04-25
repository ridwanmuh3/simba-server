package model

import "time"

type DapurResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
}

type CreateDapurRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"omitempty,max=255"`
}

type UpdateDapurRequest struct {
	ID          uint   `json:"-" validate:"required"`
	Name        string `json:"name" validate:"omitempty,min=2,max=100"`
	Description string `json:"description" validate:"omitempty,max=255"`
	IsActive    *bool  `json:"is_active" validate:"omitempty"`
}

type SelectDapurRequest struct {
	DapurID uint `json:"dapur_id" validate:"required,min=1"`
	UserID  int  `json:"-" validate:"required,min=1"`
}

type DeleteDapurRequest struct {
	ID uint `json:"-" validate:"required"`
}
