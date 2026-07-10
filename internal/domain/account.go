package domain

import (
	"context"
	"time"
)

type Account struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Currency  string     `json:"currency"`
	Color     *string    `json:"color"`
	IconKey   *string    `json:"icon_key"`
	Balance   float64    `json:"balance"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateAccountRequest struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Currency string  `json:"currency"`
	Color    *string `json:"color"`
	IconKey  *string `json:"icon_key"`
}

type UpdateAccountRequest struct {
	Name     *string `json:"name,omitempty"`
	Type     *string `json:"type,omitempty"`
	Currency *string `json:"currency,omitempty"`
	Color    *string `json:"color,omitempty"`
	IconKey  *string `json:"icon_key,omitempty"`
}

type AccountRepository interface {
	Create(ctx context.Context, userID int64, req CreateAccountRequest) (*Account, error)
	FindByID(ctx context.Context, id int64) (*Account, error)
	FindAllByUser(ctx context.Context, userID int64, includeTrashed bool) ([]Account, error)
	Update(ctx context.Context, id int64, req UpdateAccountRequest) (*Account, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	PermanentDelete(ctx context.Context, id int64) error
	GetBalance(ctx context.Context, accountID int64) (float64, error)
}
