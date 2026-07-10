package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/aaron/finflow-backend/internal/domain"
)

type NotificationUseCase struct {
	repo    domain.NotificationRepository
	pushSrv domain.PushService
}

func NewNotificationUseCase(repo domain.NotificationRepository, pushSrv domain.PushService) *NotificationUseCase {
	return &NotificationUseCase{
		repo:    repo,
		pushSrv: pushSrv,
	}
}

func (uc *NotificationUseCase) RegisterDevice(ctx context.Context, userID int64, req domain.RegisterDeviceRequest) (*domain.Device, error) {
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		return nil, fmt.Errorf("%w: el token del dispositivo es requerido", domain.ErrValidation)
	}

	req.Platform = strings.TrimSpace(strings.ToLower(req.Platform))
	if req.Platform != "android" && req.Platform != "ios" && req.Platform != "web" {
		return nil, fmt.Errorf("%w: plataforma de dispositivo inválida (debe ser 'android', 'ios' o 'web')", domain.ErrValidation)
	}

	return uc.repo.RegisterDevice(ctx, userID, req)
}

func (uc *NotificationUseCase) RemoveDevice(ctx context.Context, userID int64, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("%w: el token del dispositivo es requerido", domain.ErrValidation)
	}
	return uc.repo.RemoveDevice(ctx, userID, token)
}

func (uc *NotificationUseCase) GetNotifications(ctx context.Context, userID int64, limit, offset int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return uc.repo.FindByUser(ctx, userID, limit, offset)
}

func (uc *NotificationUseCase) MarkRead(ctx context.Context, userID int64, id int64) error {
	return uc.repo.MarkRead(ctx, id)
}

func (uc *NotificationUseCase) MarkAllRead(ctx context.Context, userID int64) error {
	return uc.repo.MarkAllRead(ctx, userID)
}

func (uc *NotificationUseCase) DeleteNotification(ctx context.Context, userID int64, id int64) error {
	return uc.repo.DeleteNotification(ctx, id)
}

func (uc *NotificationUseCase) CreateAndSend(ctx context.Context, userID int64, title, body, nType string, payload map[string]any) (*domain.Notification, error) {
	// 1) Guardar en Base de Datos
	notif, err := uc.repo.CreateNotification(ctx, userID, title, body, nType, payload)
	if err != nil {
		return nil, fmt.Errorf("error al guardar notificación en DB: %w", err)
	}

	// 2) Buscar todos los tokens push del usuario
	devices, err := uc.repo.FindDevicesByUser(ctx, userID)
	if err != nil {
		slog.Error("Fallo al buscar dispositivos para notificaciones push", "user_id", userID, "error", err)
		return notif, nil // Devolvemos la notif guardada en DB de todos modos
	}

	// 3) Enviar el mensaje push a cada dispositivo registrado
	if len(devices) > 0 {
		// Convert payload to map[string]string for FCM
		stringPayload := make(map[string]string)
		stringPayload["notification_id"] = fmt.Sprintf("%d", notif.ID)
		stringPayload["type"] = nType
		for k, v := range payload {
			stringPayload[k] = fmt.Sprintf("%v", v)
		}

		for _, d := range devices {
			go func(token, platform string) {
				pushCtx := context.Background()
				if err := uc.pushSrv.SendPush(pushCtx, token, title, body, stringPayload); err != nil {
					slog.Error("No se pudo enviar notificación push via FCM", "token", token, "platform", platform, "error", err)
				}
			}(d.Token, d.Platform)
		}
	}

	return notif, nil
}
