// Package mailer envía correos electrónicos mediante el proveedor configurado
// por variables de entorno (Resend, SMTP) con un fallback Mock para desarrollo.
package mailer

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Mailer es el contrato de envío de correo. Las plantillas/asuntos son
// responsabilidad de cada proyecto consumidor.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// New selecciona la implementación según el entorno: Resend → SMTP → Mock.
func New() Mailer {
	if key := strings.TrimSpace(os.Getenv("RESEND_API_KEY")); key != "" {
		return newResendMailer(key)
	}
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	user := strings.TrimSpace(os.Getenv("SMTP_USER"))
	pass := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	if host != "" && user != "" && pass != "" {
		return newSMTPMailer(host, user, pass)
	}
	slog.Warn("⚠️ Ningún proveedor de correo configurado (RESEND_API_KEY o SMTP_*). Usando Mock (consola).")
	return NewMock()
}

type mockMailer struct{}

// NewMock crea un mailer que solo registra el correo en consola.
func NewMock() Mailer { return &mockMailer{} }

func (m *mockMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	slog.Info("MOCK MAILER: correo 'enviado'", "to", to, "subject", subject)
	return nil
}
