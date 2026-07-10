package push

import (
	"context"
	"log/slog"
)

type MockPushService struct{}

func NewMockPushService() *MockPushService {
	return &MockPushService{}
}

func (s *MockPushService) SendPush(ctx context.Context, token, title, body string, payload map[string]string) error {
	slog.Info("📢 [Mock Push Notification] Enviando push...",
		"token", token,
		"title", title,
		"body", body,
		"payload", payload,
	)
	return nil
}
