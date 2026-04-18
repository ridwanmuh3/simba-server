package model

import "time"

type TokenResponse struct {
	Token string `json:"token"`
}

type Auth struct {
	ID           int    `json:"id"`
	Fullname     string `json:"fullname"`
	Role         string `json:"role"`
	Token        string `json:"-"`
	RefreshToken string `json:"-"`
}

type RefreshSessionRequest struct {
	RefreshToken string `validate:"required,max=255"`
}

type LoginUserRequest struct {
	Username   string `json:"username" validate:"required,alphanum,min=5,max=30"`
	Password   string `json:"password" validate:"required,printascii,min=4,max=30"`
	RememberMe bool   `json:"remember_me,omitempty" validate:"omitempty,boolean"`
}

type UpdateLoginStatusUserRequest struct {
	ID         int       `json:"-" validate:"required,numeric,min=0"`
	IsActive   bool      `json:"is_active" validate:"required,boolean"`
	LastActive time.Time `json:"last_active" validate:"required,datetime"`
}

type LogoutUserRequest struct {
	ID int `json:"id" validate:"required,numeric,min=0"`
}

type VerifyUserRequest struct {
	Token string `validate:"required,max=255"`
}
