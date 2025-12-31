package model

import "time"

type Auth struct {
	ID       int
	Fullname string
	Role     string
}

type LoginUserRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=5,max=30"`
	Password string `json:"password" validate:"required,printascii,min=4,max=30"`
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
