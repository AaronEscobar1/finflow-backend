package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implementa domain.UserRepository sobre PostgreSQL (pgx).
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// hashToken deriva el sha256 hex del token JWT para almacenarlo/buscarlo sin guardar el token plano.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO security.users (email, password_hash, full_name, auth_provider, google_sub, email_verified, preferred_currency, preferred_language, theme_preference)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q,
		u.Email, u.PasswordHash, u.FullName, u.AuthProvider, u.GoogleSub, u.EmailVerified,
		u.PreferredCurrency, u.PreferredLanguage, u.ThemePreference,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error al crear usuario: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, avatar_url, auth_provider, google_sub, email_verified,
		       preferred_currency, preferred_language, theme_preference, created_at, updated_at
		FROM security.users WHERE email = $1`
	return r.scanUser(r.pool.QueryRow(ctx, q, email))
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, avatar_url, auth_provider, google_sub, email_verified,
		       preferred_currency, preferred_language, theme_preference, created_at, updated_at
		FROM security.users WHERE id = $1`
	return r.scanUser(r.pool.QueryRow(ctx, q, id))
}

func (r *UserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.AvatarURL, &u.AuthProvider,
		&u.GoogleSub, &u.EmailVerified, &u.PreferredCurrency, &u.PreferredLanguage, &u.ThemePreference,
		&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer usuario: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM security.users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error al verificar email: %w", err)
	}
	return exists, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID int64, req domain.UpdateProfileRequest) (*domain.User, error) {
	// COALESCE: solo actualiza los campos no nulos del request.
	const q = `
		UPDATE security.users SET
			full_name = COALESCE($2, full_name),
			avatar_url = COALESCE($3, avatar_url),
			preferred_currency = COALESCE($4, preferred_currency),
			preferred_language = COALESCE($5, preferred_language),
			theme_preference = COALESCE($6, theme_preference)
		WHERE id = $1
		RETURNING id, email, password_hash, full_name, avatar_url, auth_provider, google_sub, email_verified,
		          preferred_currency, preferred_language, theme_preference, created_at, updated_at`
	return r.scanUser(r.pool.QueryRow(ctx, q, userID,
		req.FullName, req.AvatarURL, req.PreferredCurrency, req.PreferredLanguage, req.ThemePreference))
}

type categoryConfig struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	IconKey string `json:"icon_key"`
	Color   string `json:"color"`
}

// seedDefaultsTx siembra las categorías por defecto y la cuenta "Efectivo" dentro de una transacción existente.
func seedDefaultsTx(ctx context.Context, tx pgx.Tx, userID int64, currency string) error {
	var categories []categoryConfig

	// Intentar leer de finance.system_configs
	var configJSON []byte
	err := tx.QueryRow(ctx, "SELECT value FROM finance.system_configs WHERE key = 'default_categories'").Scan(&configJSON)
	if err == nil {
		if errUnmarshal := json.Unmarshal(configJSON, &categories); errUnmarshal != nil {
			slog.Warn("No se pudo deserializar JSON de categorías por defecto de la base de datos, usando fallback", "error", errUnmarshal)
			categories = nil
		}
	} else {
		slog.Warn("No se pudo leer la tabla finance.system_configs para categorías, usando fallback", "error", err)
	}

	// Fallback si falló la base de datos o el unmarshal
	if len(categories) == 0 {
		categories = []categoryConfig{
			{Name: "Salario", Type: "income", IconKey: "salary", Color: "#16A34A"},
			{Name: "Ventas", Type: "income", IconKey: "sales", Color: "#0EA5E9"},
			{Name: "Alimentación", Type: "expense", IconKey: "food", Color: "#F97316"},
			{Name: "Servicios", Type: "expense", IconKey: "services", Color: "#3B82F6"},
			{Name: "Salud", Type: "expense", IconKey: "health", Color: "#EF4444"},
			{Name: "Transporte", Type: "expense", IconKey: "transport", Color: "#EAB308"},
			{Name: "Hogar", Type: "expense", IconKey: "home", Color: "#A855F7"},
		}
	}

	for _, c := range categories {
		if _, err := tx.Exec(ctx,
			"INSERT INTO finance.categories (user_id, name, type, icon_key, color) VALUES ($1, $2, $3, $4, $5)",
			userID, c.Name, c.Type, c.IconKey, c.Color); err != nil {
			return fmt.Errorf("error al sembrar categoría %s: %w", c.Name, err)
		}
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO finance.accounts (user_id, name, type, currency) VALUES ($1, 'Efectivo', 'cash', $2)",
		userID, currency); err != nil {
		return fmt.Errorf("error al sembrar cuenta por defecto: %w", err)
	}
	return nil
}

// SeedDefaults siembra las categorías por defecto y la cuenta "Efectivo" del usuario nuevo.
func (r *UserRepository) SeedDefaults(ctx context.Context, userID int64, currency string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de siembra: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := seedDefaultsTx(ctx, tx, userID, currency); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// CreateWithDefaults crea el usuario y siembra categorías+cuenta en una sola transacción; asigna u.ID.
func (r *UserRepository) CreateWithDefaults(ctx context.Context, u *domain.User, currency string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error al iniciar transacción de registro: %w", err)
	}
	defer tx.Rollback(ctx)
	const q = `INSERT INTO security.users (email, password_hash, full_name, auth_provider, google_sub, email_verified, preferred_currency, preferred_language, theme_preference)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at, updated_at`
	if err := tx.QueryRow(ctx, q, u.Email, u.PasswordHash, u.FullName, u.AuthProvider, u.GoogleSub, u.EmailVerified, u.PreferredCurrency, u.PreferredLanguage, u.ThemePreference).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return fmt.Errorf("error al crear usuario: %w", err)
	}
	if err := seedDefaultsTx(ctx, tx, u.ID, currency); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *UserRepository) CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO security.sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, hashToken(token), expiresAt)
	if err != nil {
		return fmt.Errorf("error al crear sesión: %w", err)
	}
	return nil
}

func (r *UserRepository) ValidateSession(ctx context.Context, token string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM security.sessions
			WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		)`, hashToken(token)).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("error al validar sesión: %w", err)
	}
	return ok, nil
}

func (r *UserRepository) ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE security.sessions SET expires_at = $2 WHERE token_hash = $1 AND revoked_at IS NULL",
		hashToken(token), newExpiresAt)
	if err != nil {
		return fmt.Errorf("error al extender sesión: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) RevokeSession(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE security.sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL",
		hashToken(token))
	if err != nil {
		return fmt.Errorf("error al revocar sesión: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, "UPDATE security.users SET password_hash = $2 WHERE id = $1", userID, passwordHash)
	if err != nil {
		return fmt.Errorf("error al actualizar contraseña: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) FindByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, avatar_url, auth_provider, google_sub, email_verified,
		       preferred_currency, preferred_language, theme_preference, created_at, updated_at
		FROM security.users WHERE google_sub = $1`
	return r.scanUser(r.pool.QueryRow(ctx, q, googleSub))
}

func (r *UserRepository) LinkGoogleSub(ctx context.Context, userID int64, googleSub string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE security.users SET google_sub = $2 WHERE id = $1", userID, googleSub)
	if err != nil {
		return fmt.Errorf("error al vincular cuenta de Google: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) RevokeUserSessions(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE security.sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL", userID)
	if err != nil {
		return fmt.Errorf("error al revocar sesiones del usuario: %w", err)
	}
	return nil
}
