package config

import "os"

type Config struct {
	Port     string
	GRPCPort string
}

func Load() Config {
	return Config{
		Port:     getEnv("SIMULATION_PORT", "8084"),
		GRPCPort: getEnv("SIMULATION_GRPC_PORT", "9084"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
