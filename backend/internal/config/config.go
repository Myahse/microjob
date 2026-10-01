// Package config reads runtime settings from the environment.
package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	JWTSecret       string
	UploadDir       string
	AllowedOrigins  []string
	PaymentProvider string
	AdminEmail      string
	AdminPassword   string
	SeedDemo        bool
}

func Load() Config {
	return Config{
		Addr:            env("ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://microjob:microjob@localhost:5432/microjob?sslmode=disable"),
		JWTSecret:       env("JWT_SECRET", "dev-secret-change-me"),
		UploadDir:       env("UPLOAD_DIR", "./uploads"),
		AllowedOrigins:  strings.Split(env("ALLOWED_ORIGINS", "*"), ","),
		PaymentProvider: env("PAYMENT_PROVIDER", "simulated"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
		SeedDemo:        os.Getenv("SEED_DEMO") == "true",
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
