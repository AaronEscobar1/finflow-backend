package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var t domain.Transaction
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.AccountID,
		&t.Amount,
		&t.Type,
		&t.CategoryID,
		&t.Currency,
		&t.Description,
		&t.Date,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer transacción: %w", err)
	}
	return &t, nil
}

func (r *TransactionRepository) Create(ctx context.Context, userID int64, req domain.CreateTransactionRequest, date time.Time) (*domain.Transaction, error) {
	const q = `
		INSERT INTO finance.transactions (user_id, account_id, amount, type, category_id, currency, description, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, account_id, amount, type, category_id, currency, description, date, created_at, updated_at, deleted_at`
	return r.scanTransaction(r.pool.QueryRow(ctx, q,
		userID, req.AccountID, req.Amount, req.Type, req.CategoryID, req.Currency, req.Description, date,
	))
}

func (r *TransactionRepository) FindByID(ctx context.Context, id int64) (*domain.Transaction, error) {
	const q = `
		SELECT id, user_id, account_id, amount, type, category_id, currency, description, date, created_at, updated_at, deleted_at
		FROM finance.transactions
		WHERE id = $1`
	return r.scanTransaction(r.pool.QueryRow(ctx, q, id))
}

func (r *TransactionRepository) FindByUser(ctx context.Context, userID int64, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	var sets []string
	var args []any
	idx := 1

	sets = append(sets, fmt.Sprintf("user_id = $%d", idx))
	args = append(args, userID)
	idx++

	if filter.Trashed {
		sets = append(sets, "deleted_at IS NOT NULL")
	} else {
		sets = append(sets, "deleted_at IS NULL")
	}

	if filter.From != nil {
		sets = append(sets, fmt.Sprintf("date >= $%d", idx))
		args = append(args, *filter.From)
		idx++
	}
	if filter.To != nil {
		sets = append(sets, fmt.Sprintf("date <= $%d", idx))
		args = append(args, *filter.To)
		idx++
	}
	if filter.Type != nil {
		sets = append(sets, fmt.Sprintf("type = $%d", idx))
		args = append(args, *filter.Type)
		idx++
	}
	if filter.CategoryID != nil {
		sets = append(sets, fmt.Sprintf("category_id = $%d", idx))
		args = append(args, *filter.CategoryID)
		idx++
	}
	if filter.AccountID != nil {
		sets = append(sets, fmt.Sprintf("account_id = $%d", idx))
		args = append(args, *filter.AccountID)
		idx++
	}
	if filter.Query != nil && *filter.Query != "" {
		// Use gin trigram index with ILIKE or standard word matching
		sets = append(sets, fmt.Sprintf("description ILIKE $%d", idx))
		args = append(args, "%"+*filter.Query+"%")
		idx++
	}

	whereClause := strings.Join(sets, " AND ")
	limitOffset := ""
	if filter.Limit > 0 {
		limitOffset += fmt.Sprintf(" LIMIT $%d", idx)
		args = append(args, filter.Limit)
		idx++
	}
	if filter.Offset > 0 {
		limitOffset += fmt.Sprintf(" OFFSET $%d", idx)
		args = append(args, filter.Offset)
		idx++
	}

	q := fmt.Sprintf(`
		SELECT id, user_id, account_id, amount, type, category_id, currency, description, date, created_at, updated_at, deleted_at
		FROM finance.transactions
		WHERE %s
		ORDER BY date DESC, created_at DESC%s`, whereClause, limitOffset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar transacciones: %w", err)
	}
	defer rows.Close()

	var txs []domain.Transaction
	for rows.Next() {
		t, err := r.scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		txs = append(txs, *t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando transacciones: %w", err)
	}

	return txs, nil
}

func (r *TransactionRepository) Update(ctx context.Context, id int64, req domain.UpdateTransactionRequest, date *time.Time) (*domain.Transaction, error) {
	sets := []string{}
	args := []any{}
	idx := 1

	if req.AccountID != nil {
		sets = append(sets, fmt.Sprintf("account_id = $%d", idx))
		args = append(args, *req.AccountID)
		idx++
	}
	if req.Amount != nil {
		sets = append(sets, fmt.Sprintf("amount = $%d", idx))
		args = append(args, *req.Amount)
		idx++
	}
	if req.Type != nil {
		sets = append(sets, fmt.Sprintf("type = $%d", idx))
		args = append(args, *req.Type)
		idx++
	}
	if req.CategoryID != nil {
		sets = append(sets, fmt.Sprintf("category_id = $%d", idx))
		args = append(args, *req.CategoryID)
		idx++
	}
	if req.Currency != nil {
		sets = append(sets, fmt.Sprintf("currency = $%d", idx))
		args = append(args, *req.Currency)
		idx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *req.Description)
		idx++
	}
	if date != nil {
		sets = append(sets, fmt.Sprintf("date = $%d", idx))
		args = append(args, *date)
		idx++
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE finance.transactions
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, user_id, account_id, amount, type, category_id, currency, description, date, created_at, updated_at, deleted_at`,
		strings.Join(sets, ", "), idx)

	return r.scanTransaction(r.pool.QueryRow(ctx, q, args...))
}

func (r *TransactionRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.transactions
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar transacción (soft): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.transactions
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al restaurar transacción: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) PermanentDelete(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM finance.transactions
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar transacción permanentemente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TransactionRepository) GetSummary(ctx context.Context, userID int64, from, to *time.Time) (*domain.TransactionSummary, error) {
	var sets []string
	var args []any
	idx := 1

	sets = append(sets, fmt.Sprintf("user_id = $%d", idx))
	args = append(args, userID)
	idx++

	sets = append(sets, "deleted_at IS NULL")

	if from != nil {
		sets = append(sets, fmt.Sprintf("date >= $%d", idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		sets = append(sets, fmt.Sprintf("date <= $%d", idx))
		args = append(args, *to)
		idx++
	}

	whereClause := strings.Join(sets, " AND ")
	q := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expense,
			COUNT(*) as count
		FROM finance.transactions
		WHERE %s`, whereClause)

	var s domain.TransactionSummary
	err := r.pool.QueryRow(ctx, q, args...).Scan(&s.TotalIncome, &s.TotalExpense, &s.Count)
	if err != nil {
		return nil, fmt.Errorf("error al calcular resumen: %w", err)
	}
	s.Balance = s.TotalIncome - s.TotalExpense
	return &s, nil
}

func (r *TransactionRepository) GetByCategory(ctx context.Context, userID int64, from, to *time.Time, txType *string) ([]domain.CategoryAnalytics, error) {
	var sets []string
	var args []any
	idx := 1

	sets = append(sets, fmt.Sprintf("t.user_id = $%d", idx))
	args = append(args, userID)
	idx++

	sets = append(sets, "t.deleted_at IS NULL")

	if from != nil {
		sets = append(sets, fmt.Sprintf("t.date >= $%d", idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		sets = append(sets, fmt.Sprintf("t.date <= $%d", idx))
		args = append(args, *to)
		idx++
	}
	if txType != nil {
		sets = append(sets, fmt.Sprintf("t.type = $%d", idx))
		args = append(args, *txType)
		idx++
	}

	whereClause := strings.Join(sets, " AND ")

	// Calculate overall total for percentage calculation
	totalQ := fmt.Sprintf(`
		SELECT COALESCE(SUM(amount), 0)
		FROM finance.transactions t
		WHERE %s`, whereClause)

	var overallTotal float64
	err := r.pool.QueryRow(ctx, totalQ, args...).Scan(&overallTotal)
	if err != nil {
		return nil, fmt.Errorf("error al calcular total general de analíticas: %w", err)
	}

	q := fmt.Sprintf(`
		SELECT
			t.category_id,
			c.name,
			c.color,
			c.icon_key,
			SUM(t.amount) as total,
			COUNT(*) as count
		FROM finance.transactions t
		JOIN finance.categories c ON t.category_id = c.id
		WHERE %s
		GROUP BY t.category_id, c.name, c.color, c.icon_key
		ORDER BY total DESC`, whereClause)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("error al obtener analíticas por categoría: %w", err)
	}
	defer rows.Close()

	var analytics []domain.CategoryAnalytics
	for rows.Next() {
		var a domain.CategoryAnalytics
		err := rows.Scan(
			&a.CategoryID,
			&a.CategoryName,
			&a.CategoryColor,
			&a.CategoryIcon,
			&a.Total,
			&a.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear analítica de categoría: %w", err)
		}
		if overallTotal > 0 {
			a.Percentage = (a.Total / overallTotal) * 100.0
		} else {
			a.Percentage = 0
		}
		analytics = append(analytics, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando analíticas: %w", err)
	}

	return analytics, nil
}
