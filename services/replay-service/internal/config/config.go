package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{Port: getEnv("REPLAY_PORT", "8088")}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
