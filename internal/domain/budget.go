package domain

import (
	"context"
	"time"
)

type Budget struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	CategoryID *int64     `json:"category_id"`
	Amount     float64    `json:"amount"`
	Period     string     `json:"period"` // weekly | monthly | yearly
	Currency   string     `json:"currency"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

type BudgetWithSpent struct {
	Budget
	CategoryName  *string `json:"category_name"`
	CategoryColor *string `json:"category_color"`
	CategoryIcon  *string `json:"category_icon"`
	Spent         float64 `json:"spent"`
	Remaining     float64 `json:"remaining"`
}

type CreateBudgetRequest struct {
	CategoryID *int64  `json:"category_id"`
	Amount     float64 `json:"amount"`
	Period     string  `json:"period"`
	Currency   string  `json:"currency"`
}

type UpdateBudgetRequest struct {
	CategoryID *int64   `json:"category_id,omitempty"`
	Amount     *float64 `json:"amount,omitempty"`
	Period     *string  `json:"period,omitempty"`
	Currency   *string  `json:"currency,omitempty"`
}

type BudgetRepository interface {
	Create(ctx context.Context, userID int64, req CreateBudgetRequest) (*Budget, error)
	FindByID(ctx context.Context, id int64) (*Budget, error)
	FindWithSpentByID(ctx context.Context, id int64) (*BudgetWithSpent, error)
	FindAllByUser(ctx context.Context, userID int64, includeTrashed bool) ([]BudgetWithSpent, error)
	Update(ctx context.Context, id int64, req UpdateBudgetRequest) (*Budget, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	PermanentDelete(ctx context.Context, id int64) error
}
