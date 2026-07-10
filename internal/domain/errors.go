package domain

import "errors"

var (
	// ErrInvalidCredentials se devuelve cuando email/contraseña no coinciden.
	ErrInvalidCredentials = errors.New("credenciales incorrectas")
	// ErrEmailTaken se devuelve cuando el email ya está registrado.
	ErrEmailTaken = errors.New("el correo electrónico ya está registrado")
	// ErrUserNotFound se devuelve cuando no existe el usuario.
	ErrUserNotFound = errors.New("usuario no encontrado")
	// ErrValidation agrupa errores de validación de entrada.
	ErrValidation = errors.New("datos de entrada inválidos")
	// ErrNotFound se devuelve cuando no existe el recurso.
	ErrNotFound = errors.New("recurso no encontrado")
	// ErrConflict se devuelve cuando hay un conflicto de datos (e.g. duplicado).
	ErrConflict = errors.New("conflicto de datos")
	// ErrForbidden se devuelve cuando el usuario no tiene permiso.
	ErrForbidden = errors.New("acceso denegado al recurso")
)
