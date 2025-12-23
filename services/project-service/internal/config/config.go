package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{Port: getEnv("PROJECT_PORT", "8082")}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
