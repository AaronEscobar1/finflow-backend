package repository

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida: se omite el test de integración")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("no se pudo conectar: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func uniqueEmail() string {
	return "u" + strconv.FormatInt(time.Now().UnixNano(), 36) + "@test.com"
}

func TestUserRepository_CreateFindSeed(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	hash := "hash"
	name := "Juan"
	u := &domain.User{Email: uniqueEmail(), PasswordHash: &hash, FullName: &name,
		AuthProvider: "email", PreferredCurrency: "USD", PreferredLanguage: "es", ThemePreference: "system"}

	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create falló: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("Create no asignó ID")
	}

	got, err := repo.FindByEmail(ctx, u.Email)
	if err != nil || got.ID != u.ID {
		t.Fatalf("FindByEmail falló: %v (got %+v)", err, got)
	}

	exists, _ := repo.ExistsByEmail(ctx, u.Email)
	if !exists {
		t.Fatal("ExistsByEmail debería ser true")
	}

	if err := repo.SeedDefaults(ctx, u.ID, "USD"); err != nil {
		t.Fatalf("SeedDefaults falló: %v", err)
	}
	var catCount, accCount int
	pool.QueryRow(ctx, "SELECT count(*) FROM finance.categories WHERE user_id=$1", u.ID).Scan(&catCount)
	pool.QueryRow(ctx, "SELECT count(*) FROM finance.accounts WHERE user_id=$1", u.ID).Scan(&accCount)
	if catCount != 8 || accCount != 1 {
		t.Fatalf("siembra incorrecta: categorías=%d (esperado 8), cuentas=%d (esperado 1)", catCount, accCount)
	}
}

func TestUserRepository_PasswordAndGoogle(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	hash := "hash1"
	u := &domain.User{Email: uniqueEmail(), PasswordHash: &hash, AuthProvider: "email",
		PreferredCurrency: "USD", PreferredLanguage: "es", ThemePreference: "system"}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create falló: %v", err)
	}

	if err := repo.UpdatePassword(ctx, u.ID, "hash2"); err != nil {
		t.Fatalf("UpdatePassword falló: %v", err)
	}
	got, _ := repo.FindByID(ctx, u.ID)
	if got.PasswordHash == nil || *got.PasswordHash != "hash2" {
		t.Fatal("la contraseña no se actualizó")
	}

	sub := "google-sub-" + uniqueEmail()
	if err := repo.LinkGoogleSub(ctx, u.ID, sub); err != nil {
		t.Fatalf("LinkGoogleSub falló: %v", err)
	}
	byGoogle, err := repo.FindByGoogleSub(ctx, sub)
	if err != nil || byGoogle.ID != u.ID {
		t.Fatalf("FindByGoogleSub falló: %v", err)
	}

	// RevokeUserSessions: crear sesión y verificar que se revoca.
	if err := repo.CreateSession(ctx, u.ID, "tok-"+sub, timeNowPlusHour()); err != nil {
		t.Fatalf("CreateSession falló: %v", err)
	}
	if err := repo.RevokeUserSessions(ctx, u.ID); err != nil {
		t.Fatalf("RevokeUserSessions falló: %v", err)
	}
	ok, _ := repo.ValidateSession(ctx, "tok-"+sub)
	if ok {
		t.Fatal("la sesión debería estar revocada")
	}
}

func timeNowPlusHour() time.Time { return time.Now().Add(time.Hour) }

func TestUserRepository_SessionLifecycle(t *testing.T) {
	pool := testPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	hash := "hash"
	u := &domain.User{Email: uniqueEmail(), PasswordHash: &hash, AuthProvider: "email",
		PreferredCurrency: "USD", PreferredLanguage: "es", ThemePreference: "system"}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create falló: %v", err)
	}

	token := "jwt.token.abc"
	if err := repo.CreateSession(ctx, u.ID, token, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("CreateSession falló: %v", err)
	}
	ok, err := repo.ValidateSession(ctx, token)
	if err != nil || !ok {
		t.Fatalf("ValidateSession debería ser true: %v", err)
	}
	if err := repo.ExtendSession(ctx, token, time.Now().Add(48*time.Hour)); err != nil {
		t.Fatalf("ExtendSession falló: %v", err)
	}
	if err := repo.RevokeSession(ctx, token); err != nil {
		t.Fatalf("RevokeSession falló: %v", err)
	}
	ok, _ = repo.ValidateSession(ctx, token)
	if ok {
		t.Fatal("ValidateSession debería ser false tras revocar")
	}
}
