// Package oauth verifica tokens de proveedores OAuth (Google) del lado del servidor.
package oauth

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/idtoken"
)

// GoogleUser contiene los datos de identidad extraídos de un Google ID token válido.
type GoogleUser struct {
	Sub           string
	Email         string
	Name          string
	Picture       string
	EmailVerified bool
}

// VerifyGoogleIDToken valida el id_token contra las claves públicas de Google
// y comprueba que la audiencia coincida con clientID. Devuelve la identidad.
func VerifyGoogleIDToken(ctx context.Context, rawIDToken, clientID string) (*GoogleUser, error) {
	if rawIDToken == "" {
		return nil, errors.New("id_token vacío")
	}
	payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
	if err != nil {
		return nil, fmt.Errorf("id_token de Google inválido: %w", err)
	}
	return parseGoogleClaims(payload.Claims)
}

// parseGoogleClaims extrae y normaliza los claims de Google. Aislada para tests.
func parseGoogleClaims(claims map[string]interface{}) (*GoogleUser, error) {
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("el token no contiene 'sub'")
	}
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	verified := false
	switch v := claims["email_verified"].(type) {
	case bool:
		verified = v
	case string:
		verified = v == "true"
	}

	return &GoogleUser{
		Sub:           sub,
		Email:         email,
		Name:          name,
		Picture:       picture,
		EmailVerified: verified,
	}, nil
}
