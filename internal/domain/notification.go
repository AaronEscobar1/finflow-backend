package domain

import (
	"context"
	"time"
)

type Device struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // android | ios | web
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Notification struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Type      string         `json:"type"` // debt_due | budget_exceeded | system
	Payload   map[string]any `json:"payload,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type RegisterDeviceRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type NotificationRepository interface {
	RegisterDevice(ctx context.Context, userID int64, req RegisterDeviceRequest) (*Device, error)
	RemoveDevice(ctx context.Context, userID int64, token string) error
	FindDevicesByUser(ctx context.Context, userID int64) ([]Device, error)
	CreateNotification(ctx context.Context, userID int64, title, body, nType string, payload map[string]any) (*Notification, error)
	FindByUser(ctx context.Context, userID int64, limit, offset int) ([]Notification, error)
	MarkRead(ctx context.Context, id int64) error
	MarkAllRead(ctx context.Context, userID int64) error
	DeleteNotification(ctx context.Context, id int64) error
}

type PushService interface {
	SendPush(ctx context.Context, token, title, body string, payload map[string]string) error
}
