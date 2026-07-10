package domain

import (
	"context"
	"time"
)

type Debt struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	Amount      float64    `json:"amount"`
	Direction   string     `json:"direction"` // receivable | payable
	PersonName  string     `json:"person_name"`
	Description string     `json:"description"`
	Currency    string     `json:"currency"`
	Date        time.Time  `json:"date"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	IsPaid      bool       `json:"is_paid"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateDebtRequest struct {
	Amount      float64 `json:"amount"`
	Direction   string  `json:"direction"`
	PersonName  string  `json:"person_name"`
	Description string  `json:"description"`
	Currency    string  `json:"currency"`
	Date        string  `json:"date"` // RFC3339 format
	DueDate     *string `json:"due_date,omitempty"`
}

type UpdateDebtRequest struct {
	Amount      *float64 `json:"amount,omitempty"`
	Direction   *string  `json:"direction,omitempty"`
	PersonName  *string  `json:"person_name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
	Date        *string  `json:"date,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
}

type DebtRepository interface {
	Create(ctx context.Context, userID int64, req CreateDebtRequest, date time.Time, dueDate *time.Time) (*Debt, error)
	FindByID(ctx context.Context, id int64) (*Debt, error)
	FindAllByUser(ctx context.Context, userID int64, direction *string, paid *bool, includeTrashed bool) ([]Debt, error)
	Update(ctx context.Context, id int64, req UpdateDebtRequest, date *time.Time, dueDate *time.Time) (*Debt, error)
	MarkPaid(ctx context.Context, id int64) (*Debt, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	PermanentDelete(ctx context.Context, id int64) error
}
