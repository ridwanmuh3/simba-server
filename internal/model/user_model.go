package model

import "time"

type UserResponse struct {
	ID         int       `json:"id,omitempty"`
	Username   string    `json:"username,omitempty"`
	Fullname   string    `json:"fullname,omitempty"`
	Role       string    `json:"role,omitempty"`
	IsActive   bool      `json:"is_active,omitempty"`
	LastActive time.Time `json:"last_active,omitzero"`
	CreatedAt  time.Time `json:"created_at,omitzero"`
}

type UsersStatsResponse struct {
	UsersTotal           int64 `json:"users_total,omitempty"`
	UsersSuperAdminTotal int64 `json:"users_super_admin_total,omitempty"`
	UsersAdminTotal      int64 `json:"users_admin_total,omitempty"`
	UsersActiveTotal     int64 `json:"users_active_total,omitempty"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=5,max=30"`
	Fullname string `json:"fullname" validate:"required,min=2,max=50"`
	Role     string `json:"role" validate:"required,userrole,min=4,max=20"`
	Password string `json:"password" validate:"required,min=4,max=30"`
}

type UpdateUserRequest struct {
	ID       int    `json:"-" validate:"required,numeric,min=0"`
	Fullname string `json:"fullname,omitempty" validate:"omitempty,min=2,max=50"`
	Password string `json:"old_password,omitempty" validate:"omitempty,min=4,max=30"`
}

type DeleteUserRequest struct {
	AuthID int `json:"-" validate:"required,numeric"`
	ID     int `json:"id" validate:"required,numeric"`
}

type FindByIdUserRequest struct {
	ID int `json:"id" validate:"required,numeric"`
}

type ResetPasswordRequest struct {
	Username    string `json:"username" validate:"required,alphanum,min=5,max=30"`
	NewPassword string `json:"new_password" validate:"required,printascii,min=4,max=30"`
}

type FindAllUserRequest struct {
	Fullname string `query:"fullname,omitempty" validate:"omitempty,max=30"`
	Page     int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size     int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
}
