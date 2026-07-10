package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) scanAccount(row pgx.Row) (*domain.Account, error) {
	var a domain.Account
	err := row.Scan(
		&a.ID,
		&a.UserID,
		&a.Name,
		&a.Type,
		&a.Currency,
		&a.Color,
		&a.IconKey,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer cuenta: %w", err)
	}
	return &a, nil
}

func (r *AccountRepository) Create(ctx context.Context, userID int64, req domain.CreateAccountRequest) (*domain.Account, error) {
	const q = `
		INSERT INTO finance.accounts (user_id, name, type, currency, color, icon_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, type, currency, color, icon_key, created_at, updated_at, deleted_at`
	return r.scanAccount(r.pool.QueryRow(ctx, q,
		userID, req.Name, req.Type, req.Currency, req.Color, req.IconKey,
	))
}

func (r *AccountRepository) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	const q = `
		SELECT id, user_id, name, type, currency, color, icon_key, created_at, updated_at, deleted_at
		FROM finance.accounts
		WHERE id = $1`
	return r.scanAccount(r.pool.QueryRow(ctx, q, id))
}

func (r *AccountRepository) FindAllByUser(ctx context.Context, userID int64, includeTrashed bool) ([]domain.Account, error) {
	var q string
	if includeTrashed {
		q = `
			SELECT id, user_id, name, type, currency, color, icon_key, created_at, updated_at, deleted_at
			FROM finance.accounts
			WHERE user_id = $1
			ORDER BY created_at DESC`
	} else {
		q = `
			SELECT id, user_id, name, type, currency, color, icon_key, created_at, updated_at, deleted_at
			FROM finance.accounts
			WHERE user_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC`
	}

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("error al listar cuentas: %w", err)
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		a, err := r.scanAccount(rows)
		if err != nil {
			return nil, err
		}
		// Calculate balance for each account
		bal, err := r.GetBalance(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		a.Balance = bal
		accounts = append(accounts, *a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando cuentas: %w", err)
	}

	return accounts, nil
}

func (r *AccountRepository) Update(ctx context.Context, id int64, req domain.UpdateAccountRequest) (*domain.Account, error) {
	const q = `
		UPDATE finance.accounts
		SET
			name = COALESCE($2, name),
			type = COALESCE($3, type),
			currency = COALESCE($4, currency),
			color = COALESCE($5, color),
			icon_key = COALESCE($6, icon_key)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, name, type, currency, color, icon_key, created_at, updated_at, deleted_at`
	return r.scanAccount(r.pool.QueryRow(ctx, q,
		id, req.Name, req.Type, req.Currency, req.Color, req.IconKey,
	))
}

func (r *AccountRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.accounts
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar cuenta (soft): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.accounts
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al restaurar cuenta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) PermanentDelete(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM finance.accounts
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar cuenta permanentemente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AccountRepository) GetBalance(ctx context.Context, accountID int64) (float64, error) {
	const q = `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0)
		FROM finance.transactions
		WHERE account_id = $1 AND deleted_at IS NULL`
	var balance float64
	err := r.pool.QueryRow(ctx, q, accountID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("error al calcular balance de cuenta: %w", err)
	}
	return balance, nil
}
