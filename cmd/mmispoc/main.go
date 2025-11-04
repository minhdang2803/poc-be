package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mmispoc/internal/database"
	"mmispoc/internal/repository"
	"mmispoc/internal/service"
	httptransport "mmispoc/internal/transport/http"
)

func main() {
	cfg := loadConfig()

	db, err := database.OpenPostgres(database.PostgresConfig{URL: cfg.DatabaseURL})
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	orderRepo := repository.NewOrder(db)
	restaurantRepo := repository.NewRestaurant(db)
	ingredientRepo := repository.NewIngredient(db)
	userRepo := repository.NewUser(db)

	orderService := service.NewOrder(orderRepo, restaurantRepo, ingredientRepo)
	userService := service.NewUser(userRepo, restaurantRepo, cfg.JWTSecret, cfg.JWTTokenTTL)
	handler := withCORS(httptransport.NewRouter(userService, orderService))

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on %s", cfg.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	waitForShutdown(server, cfg.ShutdownTimeout)
}

type config struct {
	Address         string
	DatabaseURL     string
	ShutdownTimeout time.Duration
	JWTSecret       string
	JWTTokenTTL     time.Duration
}

func loadConfig() config {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://appuser:supersecretpassword@localhost:5432/appdb"
	}

	timeout := 5 * time.Second
	if raw := os.Getenv("SHUTDOWN_TIMEOUT"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			timeout = parsed
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}

	jwtTTL := service.DefaultTokenTTL()
	if raw := os.Getenv("JWT_TOKEN_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			jwtTTL = parsed
		}
	}

	return config{
		Address:         addr,
		DatabaseURL:     dbURL,
		ShutdownTimeout: timeout,
		JWTSecret:       jwtSecret,
		JWTTokenTTL:     jwtTTL,
	}
}

func waitForShutdown(server *http.Server, timeout time.Duration) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
