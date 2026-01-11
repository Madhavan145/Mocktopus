package config

import "os"

type Config struct {
	Port string
}

func Load() Config {
	return Config{
		Port: getEnv("CATALOG_PORT", "8090"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
