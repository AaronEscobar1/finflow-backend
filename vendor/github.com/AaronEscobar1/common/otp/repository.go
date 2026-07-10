package otp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository define el contrato de persistencia de OTP. Los consumidores
// (backends) lo inyectan en sus casos de uso de autenticación.
type Repository interface {
	CreateOTP(ctx context.Context, email, code, action string, ttl time.Duration) error
	ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error
	HasRecentOTP(ctx context.Context, email, action string, withinSeconds int) (bool, error)
}

// PostgresRepository implementa Repository sobre una tabla Postgres parametrizable.
// qualifiedTable debe incluir el esquema (ej. "security.user_otp") y tener las columnas
// email, otp_code, action, expires_at, consumed_at, created_at.
type PostgresRepository struct {
	pool  *pgxpool.Pool
	table string
}

func NewPostgresRepository(pool *pgxpool.Pool, qualifiedTable string) *PostgresRepository {
	return &PostgresRepository{pool: pool, table: qualifiedTable}
}

func (r *PostgresRepository) CreateOTP(ctx context.Context, email, code, action string, ttl time.Duration) error {
	q := fmt.Sprintf(
		"INSERT INTO %s (email, otp_code, action, expires_at) VALUES ($1, $2, $3, now() + make_interval(secs => $4))",
		r.table,
	)
	_, err := r.pool.Exec(ctx, q, email, code, action, ttl.Seconds())
	if err != nil {
		return fmt.Errorf("error al crear OTP: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error {
	// Consume atómicamente el OTP vigente más reciente que coincida.
	q := fmt.Sprintf(`
		UPDATE %s SET consumed_at = now()
		WHERE id = (
			SELECT id FROM %s
			WHERE email = $1 AND otp_code = $2 AND action = $3
			  AND consumed_at IS NULL AND expires_at > now()
			ORDER BY created_at DESC
			LIMIT 1
		)
		RETURNING id`, r.table, r.table)
	var id int64
	err := r.pool.QueryRow(ctx, q, email, code, action).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("código OTP inválido, expirado o ya consumido")
	}
	if err != nil {
		return fmt.Errorf("error al validar OTP: %w", err)
	}
	return nil
}

func (r *PostgresRepository) HasRecentOTP(ctx context.Context, email, action string, withinSeconds int) (bool, error) {
	q := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s
			WHERE email = $1 AND action = $2
			  AND created_at > now() - make_interval(secs => $3)
		)`, r.table)
	var exists bool
	if err := r.pool.QueryRow(ctx, q, email, action, withinSeconds).Scan(&exists); err != nil {
		return false, fmt.Errorf("error al verificar OTP reciente: %w", err)
	}
	return exists, nil
}
