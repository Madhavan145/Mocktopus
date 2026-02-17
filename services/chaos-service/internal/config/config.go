package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{Port: getEnv("CHAOS_PORT", "8087")}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
