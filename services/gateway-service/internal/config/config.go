package config

import "os"

type Config struct {
	Port              string
	AuthServiceURL    string
	ProjectURL        string
	TopologyURL       string
	SimulationURL     string
	MetricsURL        string
	CostURL           string
	ChaosURL          string
	ReplayURL         string
	RecommendationURL string
	CatalogURL        string
	IaCURL            string
	TemplateURL       string
	AdminURL          string
	AllowedOrigin     string
	JWTSecret         string
}

func Load() Config {
	return Config{
		Port:              getEnv("GATEWAY_PORT", "8080"),
		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		ProjectURL:        getEnv("PROJECT_SERVICE_URL", "http://localhost:8082"),
		TopologyURL:       getEnv("TOPOLOGY_SERVICE_URL", "http://localhost:8083"),
		SimulationURL:     getEnv("SIMULATION_SERVICE_URL", "http://localhost:8084"),
		MetricsURL:        getEnv("METRICS_SERVICE_URL", "http://localhost:8085"),
		CostURL:           getEnv("COST_SERVICE_URL", "http://localhost:8086"),
		ChaosURL:          getEnv("CHAOS_SERVICE_URL", "http://localhost:8087"),
		ReplayURL:         getEnv("REPLAY_SERVICE_URL", "http://localhost:8088"),
		RecommendationURL: getEnv("RECOMMENDATION_SERVICE_URL", "http://localhost:8089"),
		CatalogURL:        getEnv("CATALOG_SERVICE_URL", "http://localhost:8090"),
		IaCURL:            getEnv("IAC_SERVICE_URL", "http://localhost:8091"),
		TemplateURL:       getEnv("TEMPLATE_SERVICE_URL", "http://localhost:8092"),
		AdminURL:          getEnv("ADMIN_SERVICE_URL", "http://localhost:8093"),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:3000"),
		JWTSecret:         getEnv("JWT_SECRET", "dev-secret-change-me"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
