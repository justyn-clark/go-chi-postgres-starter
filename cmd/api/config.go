package main

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	// Server
	Port        string
	Environment string
	LogLevel    string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret string

	// Optional: API Access Token (for service-to-service auth)
	APIAccessToken string
}

// LoadConfig loads configuration from environment variables with validation
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Server
	cfg.Port = getEnv("PORT", "8080")
	cfg.Environment = getEnv("ENVIRONMENT", "development")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	// Database - required
	cfg.DatabaseURL = getEnvRequired("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("required environment variable DATABASE_URL is not set")
	}

	// JWT - required in production, but has default for development
	cfg.JWTSecret = getEnv("JWT_SECRET", "dev-secret-change-in-production")
	if cfg.JWTSecret == "dev-secret-change-in-production" && cfg.Environment == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production environment")
	}
	if cfg.JWTSecret == "dev-secret-change-in-production" {
		fmt.Fprintf(os.Stderr, "WARNING: Using default JWT_SECRET. Change this in production!\n")
	}

	// API Access Token (optional - for service-to-service auth without login)
	cfg.APIAccessToken = getEnv("API_ACCESS_TOKEN", "")

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

// getEnvRequired gets a required environment variable or returns empty string
func getEnvRequired(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
