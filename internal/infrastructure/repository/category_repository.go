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

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) scanCategory(row pgx.Row) (*domain.Category, error) {
	var c domain.Category
	err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Type,
		&c.IconKey,
		&c.Color,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer categoría: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) Create(ctx context.Context, userID int64, req domain.CreateCategoryRequest) (*domain.Category, error) {
	const q = `
		INSERT INTO finance.categories (user_id, name, type, icon_key, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at`
	cat, err := r.scanCategory(r.pool.QueryRow(ctx, q,
		userID, req.Name, req.Type, req.IconKey, req.Color,
	))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return cat, nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id int64) (*domain.Category, error) {
	const q = `
		SELECT id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at
		FROM finance.categories
		WHERE id = $1`
	return r.scanCategory(r.pool.QueryRow(ctx, q, id))
}

func (r *CategoryRepository) FindAllByUser(ctx context.Context, userID int64, typeFilter string, includeTrashed bool) ([]domain.Category, error) {
	var q string
	var args []any
	args = append(args, userID)

	if includeTrashed {
		if typeFilter != "" {
			q = `
				SELECT id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at
				FROM finance.categories
				WHERE user_id = $1 AND type = $2
				ORDER BY name ASC`
			args = append(args, typeFilter)
		} else {
			q = `
				SELECT id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at
				FROM finance.categories
				WHERE user_id = $1
				ORDER BY name ASC`
		}
	} else {
		if typeFilter != "" {
			q = `
				SELECT id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at
				FROM finance.categories
				WHERE user_id = $1 AND type = $2 AND deleted_at IS NULL
				ORDER BY name ASC`
			args = append(args, typeFilter)
		} else {
			q = `
				SELECT id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at
				FROM finance.categories
				WHERE user_id = $1 AND deleted_at IS NULL
				ORDER BY name ASC`
		}
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("error al listar categorías: %w", err)
	}
	defer rows.Close()

	var categories []domain.Category
	for rows.Next() {
		c, err := r.scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, *c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando categorías: %w", err)
	}

	return categories, nil
}

func (r *CategoryRepository) Update(ctx context.Context, id int64, req domain.UpdateCategoryRequest) (*domain.Category, error) {
	const q = `
		UPDATE finance.categories
		SET
			name = COALESCE($2, name),
			type = COALESCE($3, type),
			icon_key = COALESCE($4, icon_key),
			color = COALESCE($5, color)
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, user_id, name, type, icon_key, color, created_at, updated_at, deleted_at`
	cat, err := r.scanCategory(r.pool.QueryRow(ctx, q,
		id, req.Name, req.Type, req.IconKey, req.Color,
	))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, domain.ErrConflict
		}
		return nil, err
	}
	return cat, nil
}

func (r *CategoryRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.categories
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar categoría (soft): %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE finance.categories
		SET deleted_at = NULL
		WHERE id = $1 AND deleted_at IS NOT NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al restaurar categoría: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) PermanentDelete(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM finance.categories
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar categoría permanentemente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
