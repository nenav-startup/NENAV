// Package config loads application configuration from environment variables
// with sensible defaults for local development. Configuration is validated
// at startup — the server fails fast on invalid required values.
//
// Reference: Master Prompt §41 — Configuration
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	App   App
	DB    Database
	Redis Redis
	Auth  Auth
	CORS  CORS
}

type App struct {
	Env             string
	Port            string
	LogLevel        string
	ShutdownTimeout time.Duration
}

type Database struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Redis struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type Auth struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	OTPExpiry       time.Duration
	OTPLength       int
}

type CORS struct {
	AllowedOrigins []string
}

// DSN returns the PostgreSQL connection string.
func (d Database) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// Addr returns the Redis address in host:port format.
func (r Redis) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// IsProd returns true if running in production.
func (c *Config) IsProd() bool {
	return c.App.Env == "production"
}

// Load reads configuration from environment variables and validates required values.
func Load() (*Config, error) {
	cfg := &Config{
		App: App{
			Env:             env("APP_ENV", "local"),
			Port:            env("APP_PORT", "8080"),
			LogLevel:        env("APP_LOG_LEVEL", "debug"),
			ShutdownTimeout: duration("APP_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		DB: Database{
			Host:            env("DB_HOST", "localhost"),
			Port:            env("DB_PORT", "5432"),
			User:            env("DB_USER", "nenav"),
			Password:        env("DB_PASSWORD", "nenav_local_dev"),
			Name:            env("DB_NAME", "nenav"),
			SSLMode:         env("DB_SSL_MODE", "disable"),
			MaxOpenConns:    integer("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    integer("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: duration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: Redis{
			Host:     env("REDIS_HOST", "localhost"),
			Port:     env("REDIS_PORT", "6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       integer("REDIS_DB", 0),
		},
		Auth: Auth{
			JWTSecret:       env("JWT_SECRET", "change-me-in-production"),
			AccessTokenTTL:  duration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: duration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
			OTPExpiry:       duration("OTP_EXPIRY", 5*time.Minute),
			OTPLength:       integer("OTP_LENGTH", 6),
		},
		CORS: CORS{
			AllowedOrigins: envSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate checks that critical configuration values are safe.
func (c *Config) validate() error {
	if c.IsProd() && c.Auth.JWTSecret == "change-me-in-production" {
		return fmt.Errorf("JWT_SECRET must be set in production")
	}
	if c.IsProd() && c.DB.Password == "nenav_local_dev" {
		return fmt.Errorf("DB_PASSWORD must be changed in production")
	}
	if c.DB.MaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be >= 1")
	}
	port, err := strconv.Atoi(c.App.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("APP_PORT must be a valid port number, got %q", c.App.Port)
	}
	return nil
}

// --- helpers ---

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func integer(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
