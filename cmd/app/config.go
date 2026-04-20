package main

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type config struct {
	ListenAddr         string
	DatabasePath       string
	BaseURL            string
	GitHubClientID     string
	GitHubClientSecret string
	CookieSecure       bool
	SessionTTL         time.Duration
}

func loadConfig() (config, error) {
	cfg := config{
		ListenAddr:   envOrDefault("LISTEN_ADDR", ":8080"),
		DatabasePath: envOrDefault("DATABASE_PATH", "./data/app.db"),
		BaseURL:      envOrDefault("BASE_URL", "http://localhost:8080"),
		CookieSecure: envBool("COOKIE_SECURE", false),
		SessionTTL:   14 * 24 * time.Hour,
	}
	cfg.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	cfg.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")

	if cfg.GitHubClientID == "" || cfg.GitHubClientSecret == "" {
		return cfg, errors.New("GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET are required")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
