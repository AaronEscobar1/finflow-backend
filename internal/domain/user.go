package domain

import (
	"context"
	"time"
)

// User representa a un usuario registrado.
type User struct {
	ID                int64
	Email             string
	PasswordHash      *string // nil para usuarios solo-Google
	FullName          *string
	AvatarURL         *string
	AuthProvider      string // 'email' | 'google'
	GoogleSub         *string
	EmailVerified     bool
	PreferredCurrency string
	PreferredLanguage string
	ThemePreference   string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// RegisterRequest es el payload de registro. Password viene cifrado (RSA base64).
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	OTPCode  string `json:"otp_code"`
}

// LoginRequest es el payload de login. Password viene cifrado (RSA base64).
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UpdateProfileRequest actualiza datos y preferencias del usuario. Campos opcionales (puntero = no tocar).
type UpdateProfileRequest struct {
	FullName          *string `json:"full_name,omitempty"`
	AvatarURL         *string `json:"avatar_url,omitempty"`
	PreferredCurrency *string `json:"preferred_currency,omitempty"`
	PreferredLanguage *string `json:"preferred_language,omitempty"`
	ThemePreference   *string `json:"theme_preference,omitempty"`
}

// OTPRequest solicita el envío de un OTP a un email para una acción dada.
type OTPRequest struct {
	Email  string `json:"email"`
	Action string `json:"action"` // REGISTER | RECOVER
}

// ResetPasswordRequest restablece la contraseña consumiendo un OTP RECOVER.
type ResetPasswordRequest struct {
	Email       string `json:"email"`
	OTPCode     string `json:"otp_code"`
	NewPassword string `json:"new_password"` // cifrada (RSA base64)
}

// GoogleLoginRequest inicia sesión con un id_token de Google.
type GoogleLoginRequest struct {
	IDToken string `json:"id_token"`
}

// AuthResponse es la respuesta de registro/login.
type AuthResponse struct {
	Token string `json:"token"`
}

// UserRepository define la persistencia de usuarios y sesiones.
// ValidateSession y ExtendSession satisfacen el SessionValidator del common.
type UserRepository interface {
	CreateWithDefaults(ctx context.Context, user *User, currency string) error // crea el usuario y siembra categorías+cuenta en una sola transacción; asigna user.ID
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateProfile(ctx context.Context, userID int64, req UpdateProfileRequest) (*User, error)

	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	FindByGoogleSub(ctx context.Context, googleSub string) (*User, error)
	LinkGoogleSub(ctx context.Context, userID int64, googleSub string) error
	RevokeUserSessions(ctx context.Context, userID int64) error

	CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	ValidateSession(ctx context.Context, token string) (bool, error)
	ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error
	RevokeSession(ctx context.Context, token string) error
}

// AuthUseCase define la lógica de autenticación.
type AuthUseCase interface {
	Register(ctx context.Context, req RegisterRequest) (AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (AuthResponse, error)
	GetProfile(ctx context.Context, userID int64) (*User, error)
	UpdateProfile(ctx context.Context, userID int64, req UpdateProfileRequest) (*User, error)
	Logout(ctx context.Context, token string) error
	RequestOTP(ctx context.Context, req OTPRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	LoginWithGoogle(ctx context.Context, req GoogleLoginRequest) (AuthResponse, error)
}
