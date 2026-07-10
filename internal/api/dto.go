package api

import "github.com/aaron/finflow-backend/internal/domain"

// userResponse es el perfil expuesto al cliente (sin password_hash).
type userResponse struct {
	ID                int64   `json:"id"`
	Email             string  `json:"email"`
	FullName          *string `json:"full_name"`
	AvatarURL         *string `json:"avatar_url"`
	AuthProvider      string  `json:"auth_provider"`
	EmailVerified     bool    `json:"email_verified"`
	PreferredCurrency string  `json:"preferred_currency"`
	PreferredLanguage string  `json:"preferred_language"`
	ThemePreference   string  `json:"theme_preference"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID: u.ID, Email: u.Email, FullName: u.FullName, AvatarURL: u.AvatarURL,
		AuthProvider: u.AuthProvider, EmailVerified: u.EmailVerified,
		PreferredCurrency: u.PreferredCurrency, PreferredLanguage: u.PreferredLanguage,
		ThemePreference: u.ThemePreference,
	}
}
