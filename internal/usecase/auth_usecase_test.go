package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// mockUserRepo implementa domain.UserRepository en memoria.
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	users         map[string]*domain.User // por email
	byID          map[int64]*domain.User
	byGoogle      map[string]*domain.User // por google_sub
	sessionActive map[string]bool         // token -> activa
	sessionUser   map[string]int64        // token -> userID
	seeded        map[int64]bool
	nextID        int64
}

func newMockRepo() *mockUserRepo {
	return &mockUserRepo{
		users:         map[string]*domain.User{},
		byID:          map[int64]*domain.User{},
		byGoogle:      map[string]*domain.User{},
		sessionActive: map[string]bool{},
		sessionUser:   map[string]int64{},
		seeded:        map[int64]bool{},
		nextID:        1,
	}
}

func (m *mockUserRepo) CreateWithDefaults(ctx context.Context, u *domain.User, currency string) error {
	u.ID = m.nextID
	m.nextID++
	m.users[u.Email] = u
	m.byID[u.ID] = u
	m.seeded[u.ID] = true
	if u.GoogleSub != nil {
		m.byGoogle[*u.GoogleSub] = u
	}
	return nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := m.users[email]
	return ok, nil
}
func (m *mockUserRepo) UpdateProfile(ctx context.Context, userID int64, req domain.UpdateProfileRequest) (*domain.User, error) {
	u, ok := m.byID[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	if req.FullName != nil {
		u.FullName = req.FullName
	}
	return u, nil
}
func (m *mockUserRepo) CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	m.sessionActive[token] = true
	m.sessionUser[token] = userID
	return nil
}
func (m *mockUserRepo) ValidateSession(ctx context.Context, token string) (bool, error) {
	return m.sessionActive[token], nil
}
func (m *mockUserRepo) ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error {
	return nil
}
func (m *mockUserRepo) RevokeSession(ctx context.Context, token string) error {
	m.sessionActive[token] = false
	return nil
}
func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	u, ok := m.byID[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.PasswordHash = &passwordHash
	return nil
}
func (m *mockUserRepo) FindByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	if u, ok := m.byGoogle[googleSub]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}
func (m *mockUserRepo) LinkGoogleSub(ctx context.Context, userID int64, googleSub string) error {
	u, ok := m.byID[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.GoogleSub = &googleSub
	m.byGoogle[googleSub] = u
	return nil
}
func (m *mockUserRepo) RevokeUserSessions(ctx context.Context, userID int64) error {
	for token, uid := range m.sessionUser {
		if uid == userID {
			m.sessionActive[token] = false
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockOTPRepo implementa otp.Repository en memoria.
// ---------------------------------------------------------------------------

type mockOTPRepo struct {
	codes map[string]string // email|action -> code
}

func newMockOTP() *mockOTPRepo { return &mockOTPRepo{codes: map[string]string{}} }
func (m *mockOTPRepo) CreateOTP(ctx context.Context, email, code, action string, ttl time.Duration) error {
	m.codes[email+"|"+action] = code
	return nil
}
func (m *mockOTPRepo) ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error {
	key := email + "|" + action
	if m.codes[key] != "" && m.codes[key] == code {
		delete(m.codes, key)
		return nil
	}
	return fmt.Errorf("código OTP inválido")
}
func (m *mockOTPRepo) HasRecentOTP(ctx context.Context, email, action string, withinSeconds int) (bool, error) {
	return false, nil
}

// ---------------------------------------------------------------------------
// mockEmail implementa domain.EmailService capturando el último OTP enviado.
// ---------------------------------------------------------------------------

type mockEmail struct{ last string }

func (m *mockEmail) SendOTP(ctx context.Context, toEmail, code, action string) error {
	m.last = code
	return nil
}

// ---------------------------------------------------------------------------
// Helper para construir el usecase en los tests.
// ---------------------------------------------------------------------------

func newTestUC(repo domain.UserRepository) (domain.AuthUseCase, *mockOTPRepo, *mockEmail) {
	otpRepo := newMockOTP()
	em := &mockEmail{}
	return NewAuthUseCase(repo, otpRepo, em, "test-secret", "test-google-client"), otpRepo, em
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRequestOTP_AndRegister(t *testing.T) {
	repo := newMockRepo()
	uc, otpRepo, em := newTestUC(repo)
	ctx := context.Background()

	if err := uc.RequestOTP(ctx, domain.OTPRequest{Email: "a@b.com", Action: "REGISTER"}); err != nil {
		t.Fatalf("RequestOTP falló: %v", err)
	}
	if em.last == "" {
		t.Fatal("no se envió OTP")
	}
	_ = otpRepo

	res, err := uc.Register(ctx, domain.RegisterRequest{Email: "a@b.com", Password: "secret123", FullName: "Ana", OTPCode: em.last})
	if err != nil {
		t.Fatalf("Register falló: %v", err)
	}
	if res.Token == "" {
		t.Fatal("se esperaba token")
	}
	if !repo.seeded[1] {
		t.Fatal("se esperaba siembra")
	}
}

func TestRegister_RequiresValidOTP(t *testing.T) {
	repo := newMockRepo()
	uc, _, _ := newTestUC(repo)
	_, err := uc.Register(context.Background(), domain.RegisterRequest{Email: "a@b.com", Password: "x", OTPCode: "000000"})
	if err == nil {
		t.Fatal("se esperaba error con OTP inválido")
	}
}

func TestLogin_OK(t *testing.T) {
	repo := newMockRepo()
	uc, _, em := newTestUC(repo)
	ctx := context.Background()
	_ = uc.RequestOTP(ctx, domain.OTPRequest{Email: "a@b.com", Action: "REGISTER"})
	if _, err := uc.Register(ctx, domain.RegisterRequest{Email: "a@b.com", Password: "correcta", OTPCode: em.last}); err != nil {
		t.Fatalf("registro previo falló: %v", err)
	}
	res, err := uc.Login(ctx, domain.LoginRequest{Email: "a@b.com", Password: "correcta"})
	if err != nil || res.Token == "" {
		t.Fatalf("Login debería funcionar: %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockRepo()
	uc, _, em := newTestUC(repo)
	ctx := context.Background()
	_ = uc.RequestOTP(ctx, domain.OTPRequest{Email: "a@b.com", Action: "REGISTER"})
	if _, err := uc.Register(ctx, domain.RegisterRequest{Email: "a@b.com", Password: "correcta", OTPCode: em.last}); err != nil {
		t.Fatalf("registro previo falló: %v", err)
	}
	_, err := uc.Login(ctx, domain.LoginRequest{Email: "a@b.com", Password: "incorrecta"})
	if err != domain.ErrInvalidCredentials {
		t.Fatalf("se esperaba ErrInvalidCredentials, se obtuvo %v", err)
	}
}

func TestResetPassword_UpdatesAndRevokes(t *testing.T) {
	repo := newMockRepo()
	uc, _, em := newTestUC(repo)
	ctx := context.Background()

	// Registrar usuario (con OTP REGISTER) y loguear para tener una sesión activa.
	_ = uc.RequestOTP(ctx, domain.OTPRequest{Email: "r@b.com", Action: "REGISTER"})
	if _, err := uc.Register(ctx, domain.RegisterRequest{Email: "r@b.com", Password: "vieja123", FullName: "R", OTPCode: em.last}); err != nil {
		t.Fatalf("registro falló: %v", err)
	}
	loginRes, err := uc.Login(ctx, domain.LoginRequest{Email: "r@b.com", Password: "vieja123"})
	if err != nil {
		t.Fatalf("login falló: %v", err)
	}

	// Pedir OTP RECOVER y restablecer.
	if err := uc.RequestOTP(ctx, domain.OTPRequest{Email: "r@b.com", Action: "RECOVER"}); err != nil {
		t.Fatalf("RequestOTP RECOVER falló: %v", err)
	}
	if err := uc.ResetPassword(ctx, domain.ResetPasswordRequest{Email: "r@b.com", OTPCode: em.last, NewPassword: "nueva123"}); err != nil {
		t.Fatalf("ResetPassword falló: %v", err)
	}

	// La contraseña vieja ya no funciona; la nueva sí.
	if _, err := uc.Login(ctx, domain.LoginRequest{Email: "r@b.com", Password: "vieja123"}); err != domain.ErrInvalidCredentials {
		t.Fatalf("la contraseña vieja debería ser inválida, got %v", err)
	}
	if _, err := uc.Login(ctx, domain.LoginRequest{Email: "r@b.com", Password: "nueva123"}); err != nil {
		t.Fatalf("la contraseña nueva debería funcionar: %v", err)
	}

	// La sesión previa (loginRes.Token) debe haber sido revocada.
	ok, _ := repo.ValidateSession(ctx, loginRes.Token)
	if ok {
		t.Fatal("la sesión previa al reset debería estar revocada")
	}
}
