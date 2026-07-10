package push

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FirebasePushService struct {
	client *messaging.Client
}

func NewFirebasePushService(credsOrFile string) (*FirebasePushService, error) {
	var opt option.ClientOption
	if strings.HasPrefix(strings.TrimSpace(credsOrFile), "{") {
		opt = option.WithCredentialsJSON([]byte(credsOrFile))
	} else {
		opt = option.WithCredentialsFile(credsOrFile)
	}

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error al inicializar firebase app: %w", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error al obtener cliente de mensajería firebase: %w", err)
	}

	slog.Info("Servicio Firebase Push Notifications (FCM) inicializado correctamente")
	return &FirebasePushService{client: client}, nil
}

func (s *FirebasePushService) SendPush(ctx context.Context, token, title, body string, payload map[string]string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:  payload,
		Token: token,
	}

	_, err := s.client.Send(ctx, message)
	if err != nil {
		slog.Error("Fallo al enviar notificación push via FCM", "token", token, "error", err)
		return fmt.Errorf("error al enviar mensaje FCM: %w", err)
	}

	slog.Info("Notificación push enviada con éxito via FCM", "token", token)
	return nil
}
