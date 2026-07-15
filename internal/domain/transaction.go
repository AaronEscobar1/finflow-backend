package domain

import (
	"context"
	"time"
)

type Transaction struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	AccountID   int64      `json:"account_id"`
	Amount      float64    `json:"amount"`
	Type        string     `json:"type"` // income | expense
	CategoryID  int64      `json:"category_id"`
	Currency    string     `json:"currency"`
	Description string     `json:"description"`
	Date        time.Time  `json:"date"`
	InputInBs   bool       `json:"input_in_bs"`
	RateType    *string    `json:"rate_type"`
	RateValue   float64    `json:"rate_value"`
	DebtID      *int64     `json:"debt_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateTransactionRequest struct {
	AccountID   int64   `json:"account_id"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	CategoryID  int64   `json:"category_id"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	Date        string  `json:"date"` // RFC3339 format
	InputInBs   bool    `json:"input_in_bs"`
	RateType    *string `json:"rate_type"`
	RateValue   float64 `json:"rate_value"`
	DebtID      *int64  `json:"debt_id,omitempty"`
}

type UpdateTransactionRequest struct {
	AccountID   *int64   `json:"account_id,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	Type        *string  `json:"type,omitempty"`
	CategoryID  *int64   `json:"category_id,omitempty"`
	Currency    *string  `json:"currency,omitempty"`
	Description *string  `json:"description,omitempty"`
	Date        *string  `json:"date,omitempty"`
	InputInBs   *bool    `json:"input_in_bs,omitempty"`
	RateType    *string  `json:"rate_type,omitempty"`
	RateValue   *float64 `json:"rate_value,omitempty"`
	DebtID      *int64   `json:"debt_id,omitempty"`
}

type TransactionFilter struct {
	From       *time.Time
	To         *time.Time
	Type       *string
	CategoryID *int64
	AccountID  *int64
	Query      *string
	Trashed    bool
	Limit      int
	Offset     int
}

type TransactionSummary struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	Balance      float64 `json:"balance"`
	Count        int     `json:"count"`
}

type CategoryAnalytics struct {
	CategoryID    int64   `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	CategoryColor *string `json:"category_color"`
	CategoryIcon  *string `json:"category_icon"`
	Total         float64 `json:"total"`
	Count         int     `json:"count"`
	Percentage    float64 `json:"percentage"`
	Type          string  `json:"type"`
}

type TransactionRepository interface {
	Create(ctx context.Context, userID int64, req CreateTransactionRequest, date time.Time) (*Transaction, error)
	FindByID(ctx context.Context, id int64) (*Transaction, error)
	FindByUser(ctx context.Context, userID int64, filter TransactionFilter) ([]Transaction, error)
	Update(ctx context.Context, id int64, req UpdateTransactionRequest, date *time.Time) (*Transaction, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	PermanentDelete(ctx context.Context, id int64) error
	DeleteByDebtID(ctx context.Context, debtID int64) error
	GetSummary(ctx context.Context, userID int64, from, to *time.Time) (*TransactionSummary, error)
	GetByCategory(ctx context.Context, userID int64, from, to *time.Time, txType *string) ([]CategoryAnalytics, error)
}
