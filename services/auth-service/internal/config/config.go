package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	OAuthRedirectBase  string // e.g. http://localhost:8080
	FrontendURL        string // e.g. http://localhost:3000
}

func Load() Config {
	accessMins := getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15)
	refreshHours := getEnvInt("REFRESH_TOKEN_TTL_HOURS", 24*7)

	return Config{
		Port:            getEnv("AUTH_PORT", "8081"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://moktopus:moktopus@localhost:5432/moktopus?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:  time.Duration(accessMins) * time.Minute,
		RefreshTokenTTL: time.Duration(refreshHours) * time.Hour,

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		OAuthRedirectBase:  getEnv("OAUTH_REDIRECT_BASE", "http://localhost:8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
