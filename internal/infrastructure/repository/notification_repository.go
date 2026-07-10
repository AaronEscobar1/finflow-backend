package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) scanDevice(row pgx.Row) (*domain.Device, error) {
	var d domain.Device
	err := row.Scan(&d.ID, &d.UserID, &d.Token, &d.Platform, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer dispositivo: %w", err)
	}
	return &d, nil
}

func (r *NotificationRepository) scanNotification(row pgx.Row) (*domain.Notification, error) {
	var n domain.Notification
	var payloadBytes []byte
	err := row.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Type, &payloadBytes, &n.ReadAt, &n.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error al leer notificación: %w", err)
	}
	if len(payloadBytes) > 0 {
		if err := json.Unmarshal(payloadBytes, &n.Payload); err != nil {
			return nil, fmt.Errorf("error al decodificar payload de notificación: %w", err)
		}
	}
	return &n, nil
}

func (r *NotificationRepository) RegisterDevice(ctx context.Context, userID int64, req domain.RegisterDeviceRequest) (*domain.Device, error) {
	const q = `
		INSERT INTO notifications.devices (user_id, token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE
		SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, updated_at = now()
		RETURNING id, user_id, token, platform, created_at, updated_at`
	return r.scanDevice(r.pool.QueryRow(ctx, q, userID, req.Token, req.Platform))
}

func (r *NotificationRepository) RemoveDevice(ctx context.Context, userID int64, token string) error {
	const q = `
		DELETE FROM notifications.devices
		WHERE user_id = $1 AND token = $2`
	tag, err := r.pool.Exec(ctx, q, userID, token)
	if err != nil {
		return fmt.Errorf("error al eliminar dispositivo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) FindDevicesByUser(ctx context.Context, userID int64) ([]domain.Device, error) {
	const q = `
		SELECT id, user_id, token, platform, created_at, updated_at
		FROM notifications.devices
		WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("error al listar dispositivos: %w", err)
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		d, err := r.scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}
	return devices, nil
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, userID int64, title, body, nType string, payload map[string]any) (*domain.Notification, error) {
	var payloadBytes []byte
	if payload != nil {
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error al serializar payload de notificación: %w", err)
		}
	}

	const q = `
		INSERT INTO notifications.notifications (user_id, title, body, type, payload)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, body, type, payload, read_at, created_at`
	return r.scanNotification(r.pool.QueryRow(ctx, q, userID, title, body, nType, payloadBytes))
}

func (r *NotificationRepository) FindByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Notification, error) {
	const q = `
		SELECT id, user_id, title, body, type, payload, read_at, created_at
		FROM notifications.notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error al listar notificaciones: %w", err)
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		n, err := r.scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *n)
	}
	return notifications, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id int64) error {
	const q = `
		UPDATE notifications.notifications
		SET read_at = now()
		WHERE id = $1 AND read_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al marcar notificación como leída: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID int64) error {
	const q = `
		UPDATE notifications.notifications
		SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("error al marcar todas las notificaciones como leídas: %w", err)
	}
	return nil
}

func (r *NotificationRepository) DeleteNotification(ctx context.Context, id int64) error {
	const q = `
		DELETE FROM notifications.notifications
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("error al eliminar notificación: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
