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

type DebtRepository struct {
	pool *pgxpool.Pool
}

func NewDebtRepository(pool *pgxpool.Pool) *DebtRepository {
	return &DebtRepository{pool: pool}
}

func (r *DebtRepository) scanDebt(row pgx.Row) (*domain.Debt, error) {
	var d domain.Debt
	err := row.Scan(
		&d.ID,
		&d.UserID,
		&d.Amount,
		&d.Direction,
		&d.PersonName,
		&d.Description,
		&d.Currency,
		&d.Date,
		&d.DueDate,
		&d.IsPaid,
		&d.PaidAt,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer deuda: %w", err)
	}
	return &d, nil
}

func (r *DebtRepository) Create(ctx context.Context, userID int64, req domain.CreateDebtRequest, date time.Time, dueDate *time.Time) (*domain.Debt, error) {
	const q = `
		INSERT INTO finance.debts (user_id, amount, direction, person_name, description, currency, date, due_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at`
	return r.scanDebt(r.pool.QueryRow(ctx, q,
		userID, req.Amount, req.Direction, req.PersonName, req.Description, req.Currency, date, dueDate,
	))
}

func (r *DebtRepository) FindByID(ctx context.Context, id int64) (*domain.Debt, error) {
	const q = `
		SELECT id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at
		FROM finance.debts
		WHERE id = $1`
	return r.scanDebt(r.pool.QueryRow(ctx, q, id))
}

func (r *DebtRepository) FindAllByUser(ctx context.Context, userID int64, direction *string, paid *bool, includeTrashed bool) ([]domain.Debt, error) {
	var sets []string
	var args []any
	idx := 1

	sets = append(sets, fmt.Sprintf("user_id = $%d", idx))
	args = append(args, userID)
	idx++

	if includeTrashed {
		// No filter on deleted_at
	} else {
		sets = append(sets, "deleted_at IS NULL")
	}

	if direction != nil && *direction != "" {
		sets = append(sets, fmt.Sprintf("direction = $%d", idx))
		args = append(args, *direction)
		idx++
	}

	if paid != nil {
		sets = append(sets, fmt.Sprintf("is_paid = $%d", idx))
		args = append(args, *paid)
		idx++
	}

	whereClause := strings.Join(sets, " AND ")
	q := fmt.Sprintf(`
		SELECT id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at
		FROM finance.debts
		WHERE %s
		ORDER BY date DESC, created_at DESC`, whereClause)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar deudas: %w", err)
	}
	defer rows.Close()

	var debts []domain.Debt
	for rows.Next() {
		d, err := r.scanDebt(rows)
		if err != nil {
			return nil, err
		}
		debts = append(debts, *d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando deudas: %w", err)
	}

	return debts, nil
}

func (r *DebtRepository) Update(ctx context.Context, id int64, req domain.UpdateDebtRequest, date *time.Time, dueDate *time.Time) (*domain.Debt, error) {
	sets := []string{}
	args := []any{}
	idx := 1

	if req.Amount != nil {
		sets = append(sets, fmt.Sprintf("amount = $%d", idx))
		args = append(args, *req.Amount)
		idx++
	}
	if req.Direction != nil {
		sets = append(sets, fmt.Sprintf("direction = $%d", idx))
		args = append(args, *req.Direction)
		idx++
	}
	if req.PersonName != nil {
		sets = append(sets, fmt.Sprintf("person_name = $%d", idx))
		args = append(args, *req.PersonName)
		idx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, *req.Description)
		idx++
	}
	if req.Currency != nil {
		sets = append(sets, fmt.Sprintf("currency = $%d", idx))
		args = append(args, *req.Currency)
		idx++
	}
	if date != nil {
		sets = append(sets, fmt.Sprintf("date = $%d", idx))
		args = append(args, *date)
		idx++
	}
	// DuetDate can be updated to null, so check if provided as parameter.
	// Since we are parsing it, we pass it down. We'll always build dynamic sets.
	// Let's check how we handle due_date update. If req.DueDate is not nil (meaning present in JSON),
	// we update it. (Wait, req.DueDate is *string).
	if req.DueDate != nil {
		sets = append(sets, fmt.Sprintf("due_date = $%d", idx))
		if dueDate == nil {
			args = append(args, nil)
		} else {
			args = append(args, *dueDate)
		}
		idx++
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE finance.debts
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at`,
		strings.Join(sets, ", "), idx)

	return r.scanDebt(r.pool.QueryRow(ctx, q, args...))
}

func (r *DebtRepository) MarkPaid(ctx context.Context, id int64) (*domain.Debt, error) {
	const q = `
		UPDATE finance.debts
		SET is_paid = true, paid_at = now()
		WHERE id = $1 AND deleted_at IS NULL AND is_paid = false
		RETURNING id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at`
	return r.scanDebt(r.pool.QueryRow(ctx, q, id))
}

func (r *DebtRepository) MarkUnpaid(ctx context.Context, id int64) (*domain.Debt, error) {
	const q = `
		UPDATE finance.debts
		SET is_paid = false, paid_at = NULL
		WHERE id = $1 AND deleted_at IS NULL AND is_paid = true
		RETURNING id, user_id, amount, direction, person_name, description, currency, date, due_date, is_paid, paid_at, created_at, updated_at, deleted_at`
	return r.scanDebt(r.pool.QueryRow(ctx, q, id))
}

func (r *DebtRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.debts
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar deuda (soft): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DebtRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.debts
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al restaurar deuda: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DebtRepository) PermanentDelete(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM finance.debts
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar deuda permanentemente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
