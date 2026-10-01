// Command api runs the MicroJob HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/myahse/microjob/backend/internal/api"
	"github.com/myahse/microjob/backend/internal/auth"
	"github.com/myahse/microjob/backend/internal/config"
	"github.com/myahse/microjob/backend/internal/db"
	"github.com/myahse/microjob/backend/internal/payments"
	"github.com/myahse/microjob/backend/internal/seed"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if cfg.JWTSecret == "dev-secret-change-me" {
		slog.Warn("JWT_SECRET is not set; using an insecure development secret")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	if err := seed.Admin(ctx, pool, cfg.AdminEmail, cfg.AdminPassword); err != nil {
		return err
	}
	if cfg.SeedDemo {
		if err := seed.Demo(ctx, pool); err != nil {
			return err
		}
		slog.Info("demo data ready", "password", seed.DemoPassword)
	}

	provider, err := payments.New(cfg.PaymentProvider)
	if err != nil {
		return err
	}
	srv := api.NewServer(api.Options{
		DB:             pool,
		Tokens:         auth.NewTokens(cfg.JWTSecret),
		Payments:       provider,
		UploadDir:      cfg.UploadDir,
		AllowedOrigins: cfg.AllowedOrigins,
	})
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdown)
	}()
	slog.Info("MicroJob API listening", "addr", cfg.Addr, "payments", provider.Name())
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
