package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
)

type smtpMailer struct {
	host, port, user, password, from string
}

func newSMTPMailer(host, user, pass string) *smtpMailer {
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}
	slog.Info("🚀 Mailer SMTP inicializado", "host", host, "from", from)
	return &smtpMailer{host: host, port: port, user: user, password: pass, from: from}
}

func (m *smtpMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	headers := "Subject: " + subject + "\r\n" +
		"MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg := []byte(headers + "\r\n" + htmlBody)
	auth := smtp.PlainAuth("", m.user, m.password, m.host)
	addr := m.host + ":" + m.port
	// net/smtp no soporta cancelación por context.Context: una vez iniciado el envío
	// no se interrumpe aunque ctx se cancele. Es aceptable para correos transaccionales cortos.
	if err := smtp.SendMail(addr, auth, m.from, []string{to}, msg); err != nil {
		return fmt.Errorf("error al enviar correo SMTP: %w", err)
	}
	slog.Info("Correo enviado vía SMTP", "to", to)
	return nil
}
