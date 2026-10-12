package model

import "time"

type NotificationResponse struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
	IsRead    bool      `json:"is_read"`
	ActorID   *uint     `json:"actor_id,omitempty"`
	MetaJSON  string    `json:"meta_json,omitempty"`
	CreatedAt time.Time `json:"created_at,omitzero"`
}

type FindAllNotificationsRequest struct {
	Page      int    `query:"page,omitempty" validate:"omitempty,min=1"`
	Size      int    `query:"size,omitempty" validate:"omitempty,min=1,max=100"`
	IsRead    *bool  `query:"is_read,omitempty"`
	Type      string `query:"type,omitempty" validate:"omitempty,max=30"`
	DapurID   uint   `json:"-"`
	UserID    uint   `json:"-"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}
