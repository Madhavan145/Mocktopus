package exporter

import (
	"fmt"
	"strings"

	"moktopus/iac-service/internal/model"
)

// ExportDockerCompose generates Docker Compose YAML from a CTM graph
func ExportDockerCompose(graph model.Graph) *model.ExportResponse {
	var sb strings.Builder

	sb.WriteString("version: \"3.8\"\n\nservices:\n")

	for _, node := range graph.Nodes {
		sb.WriteString(generateComposeService(node, graph))
	}

	// Add volumes section if databases/storage present
	hasVolumes := false
	for _, node := range graph.Nodes {
		if node.Type == "database" || node.Type == "cache" || node.Type == "queue" || node.Type == "storage" || node.Type == "stream" || node.Type == "monitoring" || node.Type == "kubernetes" {
			hasVolumes = true
			break
		}
	}
	if hasVolumes {
		sb.WriteString("\nvolumes:\n")
		for _, node := range graph.Nodes {
			switch node.Type {
			case "database":
				sb.WriteString(fmt.Sprintf("  %s_data:\n", sanitizeName(node.ID)))
			case "stream":
				sb.WriteString(fmt.Sprintf("  %s_data:\n", sanitizeName(node.ID)))
			case "monitoring":
				sb.WriteString(fmt.Sprintf("  %s_data:\n", sanitizeName(node.ID)))
			case "kubernetes":
				sb.WriteString("  k3s_data:\n")
			}
		}
	}

	// Add networks section
	sb.WriteString("\nnetworks:\n  default:\n    driver: bridge\n")

	content := sb.String()
	return &model.ExportResponse{
		Format:  "docker-compose",
		Content: content,
		Files: []model.File{
			{Name: "docker-compose.yml", Content: content},
		},
	}
}

func generateComposeService(node model.Node, graph model.Graph) string {
	name := sanitizeName(node.ID)

	// Find dependencies (edges where this node is the source)
	var deps []string
	for _, edge := range graph.Edges {
		if edge.Source == node.ID {
			deps = append(deps, sanitizeName(edge.Target))
		}
	}

	switch node.Type {
	case "service", "worker":
		replicas := getConfigInt(node.Config, "replicas", 1)
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("  %s:\n", name))
		sb.WriteString(fmt.Sprintf("    build: ./services/%s\n", name))
		sb.WriteString("    environment:\n")
		sb.WriteString("      - NODE_ENV=production\n")
		if replicas > 1 {
			sb.WriteString("    deploy:\n")
			sb.WriteString(fmt.Sprintf("      replicas: %d\n", replicas))
		}
		if len(deps) > 0 {
			sb.WriteString("    depends_on:\n")
			for _, dep := range deps {
				sb.WriteString(fmt.Sprintf("      - %s\n", dep))
			}
		}
		sb.WriteString("\n")
		return sb.String()

	case "database":
		engine := getConfigString(node.Config, "engine", "postgres")
		var image string
		switch engine {
		case "mysql":
			image = "mysql:8"
		case "mongodb":
			image = "mongo:7"
		default:
			image = "postgres:16-alpine"
		}
		return fmt.Sprintf(`  %s:
    image: %s
    environment:
      - POSTGRES_DB=%s
      - POSTGRES_USER=app
      - POSTGRES_PASSWORD=${DB_PASSWORD:-secret}
    volumes:
      - %s_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

`, name, image, name, name)

	case "cache":
		return fmt.Sprintf(`  %s:
    image: redis:7-alpine
    command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
    ports:
      - "6379:6379"

`, name)

	case "queue":
		return fmt.Sprintf(`  %s:
    image: rabbitmq:3-management-alpine
    environment:
      - RABBITMQ_DEFAULT_USER=app
      - RABBITMQ_DEFAULT_PASS=${RABBIT_PASSWORD:-secret}
    ports:
      - "5672:5672"
      - "15672:15672"

`, name)

	case "load-balancer", "api-gateway":
		return fmt.Sprintf(`  %s:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
%s
`, name, composeDeps(deps))

	case "object-store", "storage":
		return fmt.Sprintf(`  %s:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      - MINIO_ROOT_USER=admin
      - MINIO_ROOT_PASSWORD=${MINIO_PASSWORD:-secret}
    ports:
      - "9000:9000"
      - "9001:9001"

`, name)

	case "stream":
		return fmt.Sprintf(`  %s:
    image: confluentinc/cp-kafka:7.5.0
    environment:
      - KAFKA_BROKER_ID=1
      - KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181
      - KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://%s:9092
      - KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
    ports:
      - "9092:9092"

`, name, name)

	case "notification":
		return fmt.Sprintf(`  %s:
    image: mailhog/mailhog:latest
    ports:
      - "1025:1025"
      - "8025:8025"

`, name)

	case "monitoring":
		return fmt.Sprintf(`  %s:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro

`, name)

	case "serverless", "lambda", "function":
		return fmt.Sprintf(`  %s:
    build: ./functions/%s
    environment:
      - NODE_ENV=production
    ports:
      - "9000:8080"

`, name, name)

	case "kubernetes", "container":
		return fmt.Sprintf(`  %s:
    image: rancher/k3s:latest
    privileged: true
    environment:
      - K3S_KUBECONFIG_OUTPUT=/output/kubeconfig.yaml
    ports:
      - "6443:6443"
      - "8080:80"
    volumes:
      - k3s_data:/var/lib/rancher/k3s

`, name)

	default:
		return fmt.Sprintf("  %s:\n    build: ./services/%s\n\n", name, name)
	}
}

func composeDeps(deps []string) string {
	if len(deps) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, dep := range deps {
		sb.WriteString(fmt.Sprintf("      - %s\n", dep))
	}
	return sb.String()
}
