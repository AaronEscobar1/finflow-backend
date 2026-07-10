package domain

import (
	"context"
	"time"
)

type Category struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	IconKey   *string    `json:"icon_key"`
	Color     *string    `json:"color"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateCategoryRequest struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	IconKey *string `json:"icon_key"`
	Color   *string `json:"color"`
}

type UpdateCategoryRequest struct {
	Name    *string `json:"name,omitempty"`
	Type    *string `json:"type,omitempty"`
	IconKey *string `json:"icon_key,omitempty"`
	Color   *string `json:"color,omitempty"`
}

type CategoryRepository interface {
	Create(ctx context.Context, userID int64, req CreateCategoryRequest) (*Category, error)
	FindByID(ctx context.Context, id int64) (*Category, error)
	FindAllByUser(ctx context.Context, userID int64, typeFilter string, includeTrashed bool) ([]Category, error)
	Update(ctx context.Context, id int64, req UpdateCategoryRequest) (*Category, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	PermanentDelete(ctx context.Context, id int64) error
}
