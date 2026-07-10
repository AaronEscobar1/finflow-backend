package domain

import "context"

// EmailService abstrae el envío de correos transaccionales de FinFlow.
type EmailService interface {
	SendOTP(ctx context.Context, toEmail, code, action string) error
}
