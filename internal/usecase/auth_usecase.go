package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/AaronEscobar1/common/middleware"
	"github.com/AaronEscobar1/common/oauth"
	"github.com/AaronEscobar1/common/otp"
	"github.com/AaronEscobar1/common/security"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

var dummyPasswordHash, _ = security.HashPassword("finflow-timing-guard-constante")

type authUseCase struct {
	userRepo       domain.UserRepository
	otpRepo        otp.Repository
	emailSrv       domain.EmailService
	jwtSecret      string
	googleClientID string
}

func NewAuthUseCase(userRepo domain.UserRepository, otpRepo otp.Repository, emailSrv domain.EmailService, jwtSecret, googleClientID string) domain.AuthUseCase {
	return &authUseCase{
		userRepo:       userRepo,
		otpRepo:        otpRepo,
		emailSrv:       emailSrv,
		jwtSecret:      jwtSecret,
		googleClientID: googleClientID,
	}
}

// generateToken firma un JWT HS256 con el claim user_id (numérico) que exige el AuthMiddleware del common.
func (s *authUseCase) generateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"iat":     time.Now().Unix(),
		"jti":     fmt.Sprintf("%d", time.Now().UnixNano()),
		"exp":     time.Now().AddDate(10, 0, 0).Unix(), // larga vida; la sesión en BD acota la validez real
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// issueSession genera token + registra sesión con ventana deslizante.
func (s *authUseCase) issueSession(ctx context.Context, userID int64) (string, error) {
	token, err := s.generateToken(userID)
	if err != nil {
		return "", fmt.Errorf("error al emitir token de sesión: %w", err)
	}
	expires := time.Now().Add(middleware.SessionSlidingWindow)
	if err := s.userRepo.CreateSession(ctx, userID, token, expires); err != nil {
		return "", err
	}
	return token, nil
}

func (s *authUseCase) Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthResponse, error) {
	var res domain.AuthResponse
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return res, fmt.Errorf("%w: email y contraseña son requeridos", domain.ErrValidation)
	}

	if strings.TrimSpace(req.OTPCode) == "" {
		return res, fmt.Errorf("%w: el código de verificación (otp_code) es requerido", domain.ErrValidation)
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return res, err
	}
	if exists {
		return res, domain.ErrEmailTaken
	}

	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, email, req.OTPCode, "REGISTER"); err != nil {
		return res, fmt.Errorf("%w: %s", domain.ErrValidation, err.Error())
	}

	plain, err := security.DecryptPassword(req.Password)
	if err != nil {
		return res, fmt.Errorf("%w: contraseña ilegible", domain.ErrValidation)
	}
	hash, err := security.HashPassword(plain)
	if err != nil {
		return res, fmt.Errorf("error al procesar contraseña: %w", err)
	}

	var fullName *string
	if n := strings.TrimSpace(req.FullName); n != "" {
		fullName = &n
	}
	user := &domain.User{
		Email: email, PasswordHash: &hash, FullName: fullName,
		AuthProvider: "email", PreferredCurrency: "USD", PreferredLanguage: "es", ThemePreference: "system",
	}
	if err := s.userRepo.CreateWithDefaults(ctx, user, user.PreferredCurrency); err != nil {
		slog.Error("Fallo al crear usuario con defaults", "error", err)
		return res, fmt.Errorf("error al inicializar la cuenta del usuario")
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return res, err
	}
	slog.Info("Registro exitoso", "user_id", user.ID)
	res.Token = token
	return res, nil
}

func (s *authUseCase) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthResponse, error) {
	var res domain.AuthResponse
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		return res, fmt.Errorf("%w: email y contraseña son requeridos", domain.ErrValidation)
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = security.CheckPasswordHash("x", dummyPasswordHash)
			return res, domain.ErrInvalidCredentials
		}
		return res, err
	}
	if user.PasswordHash == nil {
		// Usuario solo-Google intentando login con contraseña.
		return res, domain.ErrInvalidCredentials
	}

	plain, err := security.DecryptPassword(req.Password)
	if err != nil {
		return res, domain.ErrInvalidCredentials
	}
	if !security.CheckPasswordHash(plain, *user.PasswordHash) {
		return res, domain.ErrInvalidCredentials
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return res, err
	}
	slog.Info("Login exitoso", "user_id", user.ID)
	res.Token = token
	return res, nil
}

func (s *authUseCase) GetProfile(ctx context.Context, userID int64) (*domain.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *authUseCase) UpdateProfile(ctx context.Context, userID int64, req domain.UpdateProfileRequest) (*domain.User, error) {
	return s.userRepo.UpdateProfile(ctx, userID, req)
}

func (s *authUseCase) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.userRepo.RevokeSession(ctx, token)
}

// RequestOTP genera y envía un OTP por email para REGISTER o RECOVER (con throttle de 60s).
func (s *authUseCase) RequestOTP(ctx context.Context, req domain.OTPRequest) error {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if email == "" || (action != "REGISTER" && action != "RECOVER") {
		return fmt.Errorf("%w: email y acción válida (REGISTER|RECOVER) son requeridos", domain.ErrValidation)
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if action == "REGISTER" && exists {
		return domain.ErrEmailTaken
	}
	if action == "RECOVER" && !exists {
		// Anti-enumeración: responder éxito sin enviar nada.
		slog.Debug("Solicitud de OTP RECOVER para email inexistente; se ignora silenciosamente")
		return nil
	}

	recent, err := s.otpRepo.HasRecentOTP(ctx, email, action, 60)
	if err != nil {
		return err
	}
	if recent {
		return fmt.Errorf("%w: espera unos segundos antes de solicitar otro código", domain.ErrValidation)
	}

	code, err := otp.GenerateNumericOTP(6)
	if err != nil {
		return fmt.Errorf("error al generar código: %w", err)
	}
	if err := s.otpRepo.CreateOTP(ctx, email, code, action, 5*time.Minute); err != nil {
		return err
	}
	if err := s.emailSrv.SendOTP(ctx, email, code, action); err != nil {
		slog.Error("Fallo al enviar OTP por email", "error", err, "email", email)
		return fmt.Errorf("no se pudo enviar el código de verificación")
	}
	slog.Info("OTP enviado", "email", email, "action", action)
	return nil
}

// ResetPassword consume un OTP RECOVER y actualiza la contraseña; revoca todas las sesiones del usuario.
func (s *authUseCase) ResetPassword(ctx context.Context, req domain.ResetPasswordRequest) error {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.OTPCode == "" || req.NewPassword == "" {
		return fmt.Errorf("%w: email, otp_code y new_password son requeridos", domain.ErrValidation)
	}
	if err := s.otpRepo.ValidateAndConsumeOTP(ctx, email, req.OTPCode, "RECOVER"); err != nil {
		return fmt.Errorf("%w: %s", domain.ErrValidation, err.Error())
	}
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("error al buscar usuario para reset: %w", err)
	}
	plain, err := security.DecryptPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("%w: contraseña ilegible", domain.ErrValidation)
	}
	hash, err := security.HashPassword(plain)
	if err != nil {
		return fmt.Errorf("error al procesar contraseña: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}
	if err := s.userRepo.RevokeUserSessions(ctx, user.ID); err != nil {
		slog.Error("No se pudieron revocar sesiones tras reset", "error", err, "user_id", user.ID)
		return fmt.Errorf("error al revocar sesiones tras restablecer contraseña: %w", err)
	}
	slog.Info("Contraseña restablecida", "user_id", user.ID)
	return nil
}

// LoginWithGoogle verifica el id_token de Google y crea/vincula al usuario, devolviendo un token de sesión.
func (s *authUseCase) LoginWithGoogle(ctx context.Context, req domain.GoogleLoginRequest) (domain.AuthResponse, error) {
	var res domain.AuthResponse
	if s.googleClientID == "" {
		return res, fmt.Errorf("inicio de sesión con Google no está configurado")
	}
	if strings.TrimSpace(req.IDToken) == "" {
		return res, fmt.Errorf("%w: id_token requerido", domain.ErrValidation)
	}
	gu, err := oauth.VerifyGoogleIDToken(ctx, req.IDToken, s.googleClientID)
	if err != nil {
		return res, domain.ErrInvalidCredentials
	}
	if !gu.EmailVerified {
		return res, fmt.Errorf("%w: el email de Google no está verificado", domain.ErrValidation)
	}

	// 1) ¿Ya existe por google_sub?
	user, err := s.userRepo.FindByGoogleSub(ctx, gu.Sub)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return res, err
	}
	if user == nil {
		email := strings.ToLower(strings.TrimSpace(gu.Email))
		// 2) ¿Existe por email? -> vincular identidad Google.
		existing, ferr := s.userRepo.FindByEmail(ctx, email)
		if ferr == nil {
			if err := s.userRepo.LinkGoogleSub(ctx, existing.ID, gu.Sub); err != nil {
				return res, err
			}
			user = existing
		} else if errors.Is(ferr, domain.ErrUserNotFound) {
			// 3) Crear usuario nuevo solo-Google con siembra de defaults.
			var name *string
			if gu.Name != "" {
				name = &gu.Name
			}
			var avatar *string
			if gu.Picture != "" {
				avatar = &gu.Picture
			}
			newUser := &domain.User{
				Email: email, PasswordHash: nil, FullName: name, AvatarURL: avatar,
				AuthProvider: "google", GoogleSub: &gu.Sub, EmailVerified: true,
				PreferredCurrency: "USD", PreferredLanguage: "es", ThemePreference: "system",
			}
			if err := s.userRepo.CreateWithDefaults(ctx, newUser, newUser.PreferredCurrency); err != nil {
				return res, err
			}
			user = newUser
		} else {
			return res, ferr
		}
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return res, err
	}
	slog.Info("Login con Google exitoso", "user_id", user.ID)
	res.Token = token
	return res, nil
}
