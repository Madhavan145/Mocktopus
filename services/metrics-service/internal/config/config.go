package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{Port: getEnv("METRICS_PORT", "8085")}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
