package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BudgetRepository struct {
	pool *pgxpool.Pool
}

func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{pool: pool}
}

func (r *BudgetRepository) scanBudget(row pgx.Row) (*domain.Budget, error) {
	var b domain.Budget
	err := row.Scan(
		&b.ID,
		&b.UserID,
		&b.CategoryID,
		&b.Amount,
		&b.Period,
		&b.Currency,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer presupuesto: %w", err)
	}
	return &b, nil
}

func (r *BudgetRepository) scanBudgetWithSpent(row pgx.Row) (*domain.BudgetWithSpent, error) {
	var b domain.BudgetWithSpent
	err := row.Scan(
		&b.ID,
		&b.UserID,
		&b.CategoryID,
		&b.Amount,
		&b.Period,
		&b.Currency,
		&b.CreatedAt,
		&b.UpdatedAt,
		&b.DeletedAt,
		&b.CategoryName,
		&b.CategoryColor,
		&b.CategoryIcon,
		&b.Spent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer presupuesto con gasto: %w", err)
	}
	b.Remaining = b.Amount - b.Spent
	return &b, nil
}

func (r *BudgetRepository) Create(ctx context.Context, userID int64, req domain.CreateBudgetRequest) (*domain.Budget, error) {
	const q = `
		INSERT INTO finance.budgets (user_id, category_id, amount, period, currency)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, category_id, amount, period, currency, created_at, updated_at, deleted_at`
	budget, err := r.scanBudget(r.pool.QueryRow(ctx, q,
		userID, req.CategoryID, req.Amount, req.Period, req.Currency,
	))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return budget, nil
}

func (r *BudgetRepository) FindByID(ctx context.Context, id int64) (*domain.Budget, error) {
	const q = `
		SELECT id, user_id, category_id, amount, period, currency, created_at, updated_at, deleted_at
		FROM finance.budgets
		WHERE id = $1`
	return r.scanBudget(r.pool.QueryRow(ctx, q, id))
}

func (r *BudgetRepository) FindAllByUser(ctx context.Context, userID int64, includeTrashed bool) ([]domain.BudgetWithSpent, error) {
	var q string
	if includeTrashed {
		q = `
			SELECT
				b.id, b.user_id, b.category_id, b.amount, b.period, b.currency, b.created_at, b.updated_at, b.deleted_at,
				c.name as category_name, c.color as category_color, c.icon_key as category_icon,
				COALESCE((
					SELECT SUM(t.amount)
					FROM finance.transactions t
					WHERE t.user_id = b.user_id
					  AND t.type = 'expense'
					  AND t.deleted_at IS NULL
					  AND (b.category_id IS NULL OR t.category_id = b.category_id)
					  AND t.date >= CASE b.period
						  WHEN 'weekly' THEN now() - interval '7 days'
						  WHEN 'monthly' THEN date_trunc('month', now())
						  WHEN 'yearly' THEN date_trunc('year', now())
						  ELSE now() - interval '30 days'
					  END
				), 0) as spent
			FROM finance.budgets b
			LEFT JOIN finance.categories c ON b.category_id = c.id
			WHERE b.user_id = $1
			ORDER BY b.created_at DESC`
	} else {
		q = `
			SELECT
				b.id, b.user_id, b.category_id, b.amount, b.period, b.currency, b.created_at, b.updated_at, b.deleted_at,
				c.name as category_name, c.color as category_color, c.icon_key as category_icon,
				COALESCE((
					SELECT SUM(t.amount)
					FROM finance.transactions t
					WHERE t.user_id = b.user_id
					  AND t.type = 'expense'
					  AND t.deleted_at IS NULL
					  AND (b.category_id IS NULL OR t.category_id = b.category_id)
					  AND t.date >= CASE b.period
						  WHEN 'weekly' THEN now() - interval '7 days'
						  WHEN 'monthly' THEN date_trunc('month', now())
						  WHEN 'yearly' THEN date_trunc('year', now())
						  ELSE now() - interval '30 days'
					  END
				), 0) as spent
			FROM finance.budgets b
			LEFT JOIN finance.categories c ON b.category_id = c.id
			WHERE b.user_id = $1 AND b.deleted_at IS NULL
			ORDER BY b.created_at DESC`
	}

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("error al listar presupuestos: %w", err)
	}
	defer rows.Close()

	var budgets []domain.BudgetWithSpent
	for rows.Next() {
		b, err := r.scanBudgetWithSpent(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, *b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando presupuestos: %w", err)
	}

	return budgets, nil
}

func (r *BudgetRepository) Update(ctx context.Context, id int64, req domain.UpdateBudgetRequest) (*domain.Budget, error) {
	// Dynamically build set query
	sets := []string{}
	args := []any{}
	idx := 1

	// category_id is nullable (can be updated to nil, i.e., global budget)
	if req.CategoryID != nil {
		sets = append(sets, fmt.Sprintf("category_id = $%d", idx))
		if *req.CategoryID == 0 {
			args = append(args, nil)
		} else {
			args = append(args, *req.CategoryID)
		}
		idx++
	}

	if req.Amount != nil {
		sets = append(sets, fmt.Sprintf("amount = $%d", idx))
		args = append(args, *req.Amount)
		idx++
	}
	if req.Period != nil {
		sets = append(sets, fmt.Sprintf("period = $%d", idx))
		args = append(args, *req.Period)
		idx++
	}
	if req.Currency != nil {
		sets = append(sets, fmt.Sprintf("currency = $%d", idx))
		args = append(args, *req.Currency)
		idx++
	}

	if len(sets) == 0 {
		return r.FindByID(ctx, id)
	}

	args = append(args, id)
	q := fmt.Sprintf(`
		UPDATE finance.budgets
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, user_id, category_id, amount, period, currency, created_at, updated_at, deleted_at`,
		strings.Join(sets, ", "), idx)

	budget, err := r.scanBudget(r.pool.QueryRow(ctx, q, args...))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return budget, nil
}

func (r *BudgetRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.budgets
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar presupuesto (soft): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BudgetRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.budgets
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al restaurar presupuesto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *BudgetRepository) PermanentDelete(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM finance.budgets
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar presupuesto permanentemente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
