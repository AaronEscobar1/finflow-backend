package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultResendURL = "https://api.resend.com/emails"
const defaultResendFrom = "FinFlow <onboarding@resend.dev>"

type resendMailer struct {
	apiKey  string
	from    string
	baseURL string
	client  *http.Client
}

func newResendMailer(apiKey string) *resendMailer {
	from := strings.TrimSpace(os.Getenv("RESEND_FROM"))
	if from == "" {
		from = defaultResendFrom
	}
	slog.Info("🚀 Mailer Resend inicializado", "from", from)
	return &resendMailer{
		apiKey:  apiKey,
		from:    from,
		baseURL: defaultResendURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (m *resendMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	body, err := json.Marshal(resendRequest{From: m.from, To: []string{to}, Subject: subject, HTML: htmlBody})
	if err != nil {
		return fmt.Errorf("error al serializar correo: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("error al construir petición Resend: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("error de red al enviar correo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		slog.Error("Resend devolvió estado no exitoso", "status", resp.StatusCode, "respuesta", string(detail))
		return fmt.Errorf("Resend devolvió estado %d", resp.StatusCode)
	}
	slog.Info("Correo enviado vía Resend", "to", to)
	return nil
}
