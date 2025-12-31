package model

import "time"

var UserRoles = []string{"Admin", "Super Admin"}

type UserResponse struct {
	ID         int       `json:"id,omitempty"`
	Username   string    `json:"username,omitempty"`
	Fullname   string    `json:"fullname,omitempty"`
	Role       string    `json:"role,omitempty"`
	Token      string    `json:"token,omitempty"`
	IsActive   bool      `json:"is_active,omitempty"`
	LastActive time.Time `json:"last_active,omitzero"`
	CreatedAt  time.Time `json:"created_at,omitzero"`
	UpdatedAt  time.Time `json:"updated_at,omitzero"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=5,max=30"`
	Fullname string `json:"fullname" validate:"required,alphaspace,min=2,max=50"`
	Role     string `json:"role" validate:"required,userrole,min=4,max=20"`
	Password string `json:"password" validate:"required,printascii,min=4,max=30"`
}

type UpdateUserRequest struct {
	ID          int    `json:"-" validate:"required,numeric,min=0"`
	Fullname    string `json:"fullname,omitempty" validate:"omitempty,alphaspace,min=2,max=50"`
	OldPassword string `json:"old_password,omitempty" validate:"omitempty,printascii,required_with=NewPassword,min=4,max=30"`
	NewPassword string `json:"new_password,omitempty" validate:"omitempty,printascii,nefield=OldPassword,min=4,max=30"`
}

type DeleteUserRequest struct {
	ID int `json:"id" validate:"required,numeric,min=0"`
}

type FindByIdUserRequest struct {
	ID int `json:"id" validate:"required,numeric,min=0"`
}

type FindAllUserRequest struct {
	Fullname string `query:"fullname,omitempty" validate:"omitempty,max=30"`
	Page     int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size     int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
}
