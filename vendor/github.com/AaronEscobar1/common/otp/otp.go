// Package otp genera y valida códigos OTP numéricos de un solo uso.
package otp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// GenerateNumericOTP genera un código OTP numérico criptográficamente seguro
// de exactamente `digits` dígitos decimales.
func GenerateNumericOTP(digits int) (string, error) {
	if digits <= 0 {
		return "", errors.New("la cantidad de dígitos del OTP debe ser mayor que cero")
	}
	var sb strings.Builder
	sb.Grow(digits)
	for i := 0; i < digits; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("error al generar dígito OTP: %w", err)
		}
		sb.WriteByte(byte('0' + n.Int64()))
	}
	return sb.String(), nil
}
