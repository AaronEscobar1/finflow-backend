// @title FinFlow API
// @version 1.0
// @description API de FinFlow (Clean Architecture) para control financiero personal y familiar.
// @termsOfService http://swagger.io/terms/

// @contact.name Soporte FinFlow
// @contact.email soporte@finflow.dev

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Escribe: "Bearer <token>" para autenticarte
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/AaronEscobar1/common-go/database"
	"github.com/AaronEscobar1/common-go/middleware"
	"github.com/AaronEscobar1/common-go/otp"
	"github.com/AaronEscobar1/common-go/security"
	_ "github.com/aaron/finflow-backend/docs"
	"github.com/aaron/finflow-backend/internal/api"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/infrastructure/email"
	"github.com/aaron/finflow-backend/internal/infrastructure/push"
	"github.com/aaron/finflow-backend/internal/infrastructure/repository"
	"github.com/aaron/finflow-backend/internal/usecase"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// 1. Logger estructurado JSON.
	logLevel := slog.LevelDebug
	if env := strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV"))); env == "production" || env == "prod" {
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))
	slog.Info("Iniciando el backend de FinFlow (Clean Architecture)...")

	// 2. Cargar .env si existe.
	if err := godotenv.Load(); err != nil {
		slog.Warn("No se pudo cargar .env (puede no existir). Se usan variables del sistema.")
	}

	// 2b. Validar variables obligatorias. El servidor NO arranca sin ellas
	// (evita firmar JWT con cadena vacía o dejar la API Key admin sin valor).
	for _, key := range []string{"JWT_SECRET", "ADMIN_API_KEY"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			slog.Error("Variable de entorno obligatoria no definida", "variable", key)
			os.Exit(1)
		}
	}

	// 3. Inicializar llaves RSA en memoria (cifrado de contraseñas en tránsito).
	if err := security.InitRSAKeys(); err != nil {
		slog.Error("Fallo crítico al inicializar RSA", "error", err)
		os.Exit(1)
	}

	// 4. DATABASE_URL (con fallback de desarrollo).
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/finflow?sslmode=disable"
		slog.Warn("DATABASE_URL no definida; usando fallback de desarrollo local")
	}

	// 5. Conectar y migrar.
	searchPaths := []string{"security", "finance", "notifications", "telemetry", "public"}
	pool, err := database.ConnectAndMigrate(dbURL, searchPaths, "file://migrations")
	if err != nil {
		slog.Error("Fallo crítico en la base de datos/migraciones", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		slog.Info("Cerrando pool de base de datos...")
		pool.Close()
	}()

	// Leer secretos (ya validados como no vacíos al inicio).
	jwtSecret := os.Getenv("JWT_SECRET")
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")

	// Inyección de dependencias: repos -> usecases -> handlers.
	userRepo := repository.NewUserRepository(pool)
	middleware.SetSessionValidator(userRepo) // habilita validación/renovación de sesión en AuthMiddleware
	otpRepo := otp.NewPostgresRepository(pool, "security.user_otp")
	emailSrv := email.NewService()

	authUseCase := usecase.NewAuthUseCase(userRepo, otpRepo, emailSrv, jwtSecret, googleClientID)
	authHandler := api.NewAuthHandler(authUseCase)

	// Nuevos repositorios
	accountRepo := repository.NewAccountRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	debtRepo := repository.NewDebtRepository(pool)
	budgetRepo := repository.NewBudgetRepository(pool)
	notifRepo := repository.NewNotificationRepository(pool)

	// Inicializar FCM Push Service
	firebaseCreds := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if firebaseCreds == "" {
		firebaseCreds = "firebase-service-account.json"
	}

	var pushSrv domain.PushService
	var pushErr error
	isJSON := strings.HasPrefix(strings.TrimSpace(firebaseCreds), "{")

	if isJSON {
		pushSrv, pushErr = push.NewFirebasePushService(firebaseCreds)
		if pushErr != nil {
			slog.Error("Fallo al inicializar Firebase Push Notifications con JSON, usando mock como fallback", "error", pushErr)
			pushSrv = push.NewMockPushService()
		}
	} else if _, err := os.Stat(firebaseCreds); err == nil {
		pushSrv, pushErr = push.NewFirebasePushService(firebaseCreds)
		if pushErr != nil {
			slog.Error("Fallo al inicializar Firebase Push Notifications con archivo, usando mock como fallback", "error", pushErr)
			pushSrv = push.NewMockPushService()
		}
	} else {
		slog.Warn("Archivo de credenciales de Firebase no encontrado ni es JSON válido, usando mock para notificaciones push", "ruta/valor", firebaseCreds)
		pushSrv = push.NewMockPushService()
	}

	// Nuevos casos de uso
	accountUseCase := usecase.NewAccountUseCase(accountRepo)
	categoryUseCase := usecase.NewCategoryUseCase(categoryRepo)
	txUseCase := usecase.NewTransactionUseCase(txRepo, accountRepo, categoryRepo)
	debtUseCase := usecase.NewDebtUseCase(debtRepo)
	budgetUseCase := usecase.NewBudgetUseCase(budgetRepo, categoryRepo)
	notifUseCase := usecase.NewNotificationUseCase(notifRepo, pushSrv)

	// Nuevos handlers
	accountHandler := api.NewAccountHandler(accountUseCase)
	categoryHandler := api.NewCategoryHandler(categoryUseCase)
	txHandler := api.NewTransactionHandler(txUseCase)
	debtHandler := api.NewDebtHandler(debtUseCase)
	budgetHandler := api.NewBudgetHandler(budgetUseCase)
	notifHandler := api.NewNotificationHandler(notifUseCase)

	// 6. Router.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(ctx); err != nil {
			slog.Error("Healthcheck: base de datos sin respuesta", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"DOWN","database":"DISCONNECTED"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP","database":"CONNECTED"}`))
	})

	// Rate limit anti fuerza bruta en endpoints sensibles.
	const rlMax = 5
	const rlWindow = 10 * time.Minute
	loginRL := middleware.RateLimit(rlMax, rlWindow)
	registerRL := middleware.RateLimit(rlMax, rlWindow)
	otpRL := middleware.RateLimit(rlMax, rlWindow)
	resetRL := middleware.RateLimit(rlMax, rlWindow)
	googleRL := middleware.RateLimit(rlMax, rlWindow)

	// Rutas públicas de autenticación.
	mux.HandleFunc("GET /api/auth/public-key", authHandler.HandlePublicKey)
	mux.Handle("POST /api/auth/register", registerRL(http.HandlerFunc(authHandler.HandleRegister)))
	mux.Handle("POST /api/auth/login", loginRL(http.HandlerFunc(authHandler.HandleLogin)))
	mux.Handle("POST /api/v1/auth/otp/request", otpRL(http.HandlerFunc(authHandler.HandleRequestOTP)))
	mux.Handle("POST /api/v1/auth/password/reset", resetRL(http.HandlerFunc(authHandler.HandleResetPassword)))
	mux.Handle("POST /api/auth/google", googleRL(http.HandlerFunc(authHandler.HandleGoogleLogin)))

	// Rutas protegidas (JWT + sesión).
	auth := middleware.AuthMiddleware(jwtSecret)
	mux.Handle("GET /api/v1/me", auth(http.HandlerFunc(authHandler.HandleGetProfile)))
	mux.Handle("PUT /api/v1/me", auth(http.HandlerFunc(authHandler.HandleUpdateProfile)))
	mux.Handle("POST /api/v1/auth/logout", auth(http.HandlerFunc(authHandler.HandleLogout)))

	// --- Cuentas (Accounts) ---
	mux.Handle("GET /api/v1/accounts", auth(http.HandlerFunc(accountHandler.HandleList)))
	mux.Handle("POST /api/v1/accounts", auth(http.HandlerFunc(accountHandler.HandleCreate)))
	mux.Handle("GET /api/v1/accounts/{id}", auth(http.HandlerFunc(accountHandler.HandleGet)))
	mux.Handle("PUT /api/v1/accounts/{id}", auth(http.HandlerFunc(accountHandler.HandleUpdate)))
	mux.Handle("DELETE /api/v1/accounts/{id}", auth(http.HandlerFunc(accountHandler.HandleDelete)))
	mux.Handle("POST /api/v1/accounts/{id}/restore", auth(http.HandlerFunc(accountHandler.HandleRestore)))
	mux.Handle("DELETE /api/v1/accounts/{id}/permanent", auth(http.HandlerFunc(accountHandler.HandlePermanentDelete)))

	// --- Categorías (Categories) ---
	mux.Handle("GET /api/v1/categories", auth(http.HandlerFunc(categoryHandler.HandleList)))
	mux.Handle("POST /api/v1/categories", auth(http.HandlerFunc(categoryHandler.HandleCreate)))
	mux.Handle("PUT /api/v1/categories/{id}", auth(http.HandlerFunc(categoryHandler.HandleUpdate)))
	mux.Handle("DELETE /api/v1/categories/{id}", auth(http.HandlerFunc(categoryHandler.HandleDelete)))
	mux.Handle("POST /api/v1/categories/{id}/restore", auth(http.HandlerFunc(categoryHandler.HandleRestore)))
	mux.Handle("DELETE /api/v1/categories/{id}/permanent", auth(http.HandlerFunc(categoryHandler.HandlePermanentDelete)))

	// --- Transacciones (Transactions) ---
	mux.Handle("GET /api/v1/transactions", auth(http.HandlerFunc(txHandler.HandleList)))
	mux.Handle("POST /api/v1/transactions", auth(http.HandlerFunc(txHandler.HandleCreate)))
	mux.Handle("GET /api/v1/transactions/summary", auth(http.HandlerFunc(txHandler.HandleSummary)))
	mux.Handle("GET /api/v1/analytics/by-category", auth(http.HandlerFunc(txHandler.HandleByCategory)))
	mux.Handle("GET /api/v1/transactions/{id}", auth(http.HandlerFunc(txHandler.HandleGet)))
	mux.Handle("PUT /api/v1/transactions/{id}", auth(http.HandlerFunc(txHandler.HandleUpdate)))
	mux.Handle("DELETE /api/v1/transactions/{id}", auth(http.HandlerFunc(txHandler.HandleDelete)))
	mux.Handle("POST /api/v1/transactions/{id}/restore", auth(http.HandlerFunc(txHandler.HandleRestore)))
	mux.Handle("DELETE /api/v1/transactions/{id}/permanent", auth(http.HandlerFunc(txHandler.HandlePermanentDelete)))

	// --- Deudas (Debts) ---
	mux.Handle("GET /api/v1/debts", auth(http.HandlerFunc(debtHandler.HandleList)))
	mux.Handle("POST /api/v1/debts", auth(http.HandlerFunc(debtHandler.HandleCreate)))
	mux.Handle("GET /api/v1/debts/{id}", auth(http.HandlerFunc(debtHandler.HandleGet)))
	mux.Handle("PUT /api/v1/debts/{id}", auth(http.HandlerFunc(debtHandler.HandleUpdate)))
	mux.Handle("POST /api/v1/debts/{id}/pay", auth(http.HandlerFunc(debtHandler.HandleMarkPaid)))
	mux.Handle("DELETE /api/v1/debts/{id}", auth(http.HandlerFunc(debtHandler.HandleDelete)))
	mux.Handle("POST /api/v1/debts/{id}/restore", auth(http.HandlerFunc(debtHandler.HandleRestore)))
	mux.Handle("DELETE /api/v1/debts/{id}/permanent", auth(http.HandlerFunc(debtHandler.HandlePermanentDelete)))

	// --- Presupuestos (Budgets) ---
	mux.Handle("GET /api/v1/budgets", auth(http.HandlerFunc(budgetHandler.HandleList)))
	mux.Handle("POST /api/v1/budgets", auth(http.HandlerFunc(budgetHandler.HandleCreate)))
	mux.Handle("PUT /api/v1/budgets/{id}", auth(http.HandlerFunc(budgetHandler.HandleUpdate)))
	mux.Handle("DELETE /api/v1/budgets/{id}", auth(http.HandlerFunc(budgetHandler.HandleDelete)))
	mux.Handle("POST /api/v1/budgets/{id}/restore", auth(http.HandlerFunc(budgetHandler.HandleRestore)))
	mux.Handle("DELETE /api/v1/budgets/{id}/permanent", auth(http.HandlerFunc(budgetHandler.HandlePermanentDelete)))

	// --- Dispositivos y Notificaciones ---
	mux.Handle("POST /api/v1/devices", auth(http.HandlerFunc(notifHandler.HandleRegisterDevice)))
	mux.Handle("DELETE /api/v1/devices/{token}", auth(http.HandlerFunc(notifHandler.HandleRemoveDevice)))
	mux.Handle("GET /api/v1/notifications", auth(http.HandlerFunc(notifHandler.HandleListNotifications)))
	mux.Handle("PUT /api/v1/notifications/{id}/read", auth(http.HandlerFunc(notifHandler.HandleMarkRead)))
	mux.Handle("PUT /api/v1/notifications/read-all", auth(http.HandlerFunc(notifHandler.HandleMarkAllRead)))
	mux.Handle("DELETE /api/v1/notifications/{id}", auth(http.HandlerFunc(notifHandler.HandleDeleteNotification)))
	mux.Handle("POST /api/v1/notifications/test-push", auth(http.HandlerFunc(notifHandler.HandleTestPush)))

	// --- Documentación Swagger (Swagger UI) ---
	mux.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	// 7. Middlewares globales: trazabilidad + CORS.
	handler := middleware.CORS()(middleware.TraceAndLogMiddleware(pool)(mux))

	// 8. Servidor HTTP.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("Servidor HTTP escuchando", "puerto", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Fallo crítico del servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Apagado ordenado.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	sig := <-stop
	slog.Info("Señal de apagado detectada", "señal", sig.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Error en apagado ordenado", "error", err)
	} else {
		slog.Info("Servidor apagado limpiamente.")
	}
}
