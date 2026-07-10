// Package email implementa domain.EmailService usando el mailer del common
// y plantillas HTML propias de FinFlow.
package email

import (
	"context"
	"fmt"

	"github.com/AaronEscobar1/common-go/mailer"
	"github.com/aaron/finflow-backend/internal/domain"
)

type service struct {
	m mailer.Mailer
}

// NewService construye el EmailService seleccionando el proveedor por entorno (Resend/SMTP/Mock).
func NewService() domain.EmailService {
	return &service{m: mailer.New()}
}

const otpSubject = "Código de seguridad FinFlow"

func (s *service) SendOTP(ctx context.Context, toEmail, code, action string) error {
	return s.m.Send(ctx, toEmail, otpSubject, buildOTPHTML(code, action))
}

// buildOTPHTML arma el cuerpo HTML del correo OTP con la identidad de FinFlow.
func buildOTPHTML(code, action string) string {
	motivo := "verificar tu identidad"
	switch action {
	case "REGISTER":
		motivo = "completar tu registro en FinFlow"
	case "RECOVER":
		motivo = "restablecer tu contraseña"
	}
	return fmt.Sprintf(`
<div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;max-width:600px;margin:auto;padding:30px;border:1px solid #e1e8ed;border-radius:12px;background:#fff;">
  <div style="text-align:center;margin-bottom:25px;">
    <h1 style="color:#1A60FF;margin:0;font-size:28px;font-weight:700;">FinFlow</h1>
    <span style="color:#64748b;font-size:14px;">Tus finanzas, claras</span>
  </div>
  <div style="border-top:1px solid #f1f5f9;padding-top:25px;">
    <h2 style="color:#0f172a;margin-top:0;font-size:20px;">Código de verificación</h2>
    <p style="color:#334155;line-height:1.6;font-size:15px;">Usa este código para %s:</p>
    <div style="background:#f8fafc;border:1px dashed #cbd5e1;padding:20px;text-align:center;font-size:32px;font-weight:700;letter-spacing:8px;color:#1e293b;border-radius:8px;margin:25px 0;">%s</div>
    <p style="color:#64748b;font-size:13px;line-height:1.5;">Este código es de un solo uso y expira en <strong>5 minutos</strong>. Si no lo solicitaste, ignora este correo.</p>
  </div>
  <div style="border-top:1px solid #f1f5f9;margin-top:35px;padding-top:20px;text-align:center;color:#94a3b8;font-size:12px;">
    <p style="margin:0;">&copy; 2026 FinFlow. Todos los derechos reservados.</p>
  </div>
</div>`, motivo, code)
}
