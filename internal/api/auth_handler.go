package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/AaronEscobar1/common-go/middleware"
	"github.com/AaronEscobar1/common-go/response"
	"github.com/AaronEscobar1/common-go/security"
	"github.com/aaron/finflow-backend/internal/domain"
)

type AuthHandler struct {
	uc domain.AuthUseCase
}

func NewAuthHandler(uc domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

// mapError traduce errores de dominio a respuestas HTTP estandarizadas.
func mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		response.Error(w, r.Context(), http.StatusOK, "USER_ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", err.Error())
	case errors.Is(err, domain.ErrValidation):
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrUserNotFound):
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrConflict):
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", err.Error())
	default:
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "ocurrió un error procesando la solicitud")
	}
}

// HandlePublicKey devuelve la clave pública RSA en PEM para cifrar contraseñas en tránsito.
// @Summary Obtener clave pública RSA
// @Description Obtiene la clave pública RSA en formato PEM para cifrar contraseñas antes de enviarlas.
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]string "Clave pública obtenida"
// @Router /api/auth/public-key [get]
func (h *AuthHandler) HandlePublicKey(w http.ResponseWriter, r *http.Request) {
	pem, err := security.GetPublicKeyPEM()
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "no se pudo obtener la clave pública")
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "clave pública obtenida", map[string]string{"public_key": pem})
}

// HandleRegister registra un nuevo usuario.
// @Summary Registrar un usuario
// @Description Crea una cuenta de usuario con correo, contraseña cifrada por RSA y validación de código OTP.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body domain.RegisterRequest true "Datos de registro"
// @Success 200 {object} domain.AuthResponse "Registro exitoso"
// @Router /api/auth/register [post]
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	res, err := h.uc.Register(r.Context(), req)
	if err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "CREATED", "usuario registrado correctamente", res)
}

// HandleLogin inicia sesión con correo y contraseña.
// @Summary Iniciar sesión
// @Description Autentica al usuario usando correo y contraseña cifrada por RSA, emitiendo un token JWT.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body domain.LoginRequest true "Credenciales de acceso"
// @Success 200 {object} domain.AuthResponse "Login exitoso"
// @Router /api/auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	res, err := h.uc.Login(r.Context(), req)
	if err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "inicio de sesión exitoso", res)
}

// HandleGetProfile obtiene el perfil del usuario autenticado.
// @Summary Obtener perfil del usuario
// @Description Obtiene los datos del usuario autenticado a partir del token JWT.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} userResponse "Perfil obtenido"
// @Router /api/v1/me [get]
func (h *AuthHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}
	user, err := h.uc.GetProfile(r.Context(), userID)
	if err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "perfil obtenido", toUserResponse(user))
}

// HandleUpdateProfile actualiza el perfil del usuario.
// @Summary Actualizar perfil del usuario
// @Description Actualiza campos opcionales del perfil del usuario (nombre, avatar, moneda, idioma, tema).
// @Tags Auth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.UpdateProfileRequest true "Datos de actualización"
// @Success 200 {object} userResponse "Perfil actualizado"
// @Router /api/v1/me [put]
func (h *AuthHandler) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}
	var req domain.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	user, err := h.uc.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "perfil actualizado", toUserResponse(user))
}

// HandleLogout cierra la sesión del usuario.
// @Summary Cerrar sesión
// @Description Revoca el token de sesión actual en el backend.
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {string} string "Sesión cerrada"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	token, _ := middleware.GetTokenFromContext(r.Context())
	if err := h.uc.Logout(r.Context(), token); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", "no se pudo cerrar la sesión")
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "sesión cerrada", nil)
}

// HandleRequestOTP solicita un código OTP.
// @Summary Solicitar código OTP por email
// @Description Envía un código OTP de 6 dígitos al correo electrónico del usuario para REGISTER o RECOVER.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body domain.OTPRequest true "Datos de la solicitud de OTP"
// @Success 200 {string} string "Código enviado"
// @Router /api/v1/auth/otp/request [post]
func (h *AuthHandler) HandleRequestOTP(w http.ResponseWriter, r *http.Request) {
	var req domain.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	if err := h.uc.RequestOTP(r.Context(), req); err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "código de verificación enviado", nil)
}

// HandleResetPassword restablece la contraseña usando un OTP.
// @Summary Restablecer contraseña con OTP
// @Description Restablece la contraseña de acceso consumiendo un código OTP de tipo RECOVER.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body domain.ResetPasswordRequest true "Datos de restablecimiento"
// @Success 200 {string} string "Contraseña restablecida"
// @Router /api/v1/auth/password/reset [post]
func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req domain.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	if err := h.uc.ResetPassword(r.Context(), req); err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "contraseña restablecida", nil)
}

// HandleGoogleLogin inicia sesión con Google.
// @Summary Iniciar sesión con Google
// @Description Autentica al usuario usando un id_token de Google OAuth.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body domain.GoogleLoginRequest true "Token de Google"
// @Success 200 {object} domain.AuthResponse "Login exitoso"
// @Router /api/auth/google [post]
func (h *AuthHandler) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req domain.GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}
	res, err := h.uc.LoginWithGoogle(r.Context(), req)
	if err != nil {
		mapError(w, r, err)
		return
	}
	response.Success(w, r.Context(), "SUCCESS", "inicio de sesión con Google exitoso", res)
}
