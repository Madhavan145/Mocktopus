package parser

import (
	"fmt"
	"regexp"
	"strings"

	"moktopus/iac-service/internal/model"
)

// ParseDockerCompose parses a docker-compose.yml into a CTM graph.
// Extracts services, dependencies, ports, environment variables, networks,
// volumes, resource limits, and infers connections from env references.
func ParseDockerCompose(content string) (*model.ImportResponse, error) {
	resp := &model.ImportResponse{
		Format:   "docker-compose",
		Warnings: []string{},
	}

	services := parseComposeServices(content)
	nodes := []model.Node{}
	edges := []model.Edge{}
	nodeIDs := map[string]bool{}
	edgeSet := map[string]bool{}

	// Create nodes
	for _, svc := range services {
		nodeType := inferServiceType(svc)
		config := map[string]any{
			"label":       svc.name,
			"iacResource": "docker-compose-service",
		}
		if svc.image != "" {
			config["image"] = svc.image
		}
		if len(svc.ports) > 0 {
			config["ports"] = svc.ports
		}
		if svc.replicas > 0 {
			config["replicas"] = svc.replicas
		}
		if svc.cpuLimit != "" {
			config["cpuLimit"] = svc.cpuLimit
		}
		if svc.memoryLimit != "" {
			config["memoryLimit"] = svc.memoryLimit
		}
		if len(svc.volumes) > 0 {
			config["volumes"] = svc.volumes
		}
		if svc.healthcheck != "" {
			config["healthcheck"] = svc.healthcheck
		}
		if len(svc.networks) > 0 {
			config["networks"] = svc.networks
		}

		nodes = append(nodes, model.Node{
			ID:     svc.name,
			Type:   nodeType,
			Config: config,
		})
		nodeIDs[svc.name] = true
	}

	// Create edges from depends_on
	for _, svc := range services {
		for _, dep := range svc.dependsOn {
			if nodeIDs[dep] && dep != svc.name {
				edgeKey := svc.name + "->" + dep
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edges = append(edges, model.Edge{
						ID:     fmt.Sprintf("dep-%s-%s", svc.name, dep),
						Source: svc.name,
						Target: dep,
					})
				}
			}
		}
	}

	// Infer edges from environment variables
	for _, svc := range services {
		for _, envVal := range svc.envValues {
			for _, otherSvc := range services {
				if otherSvc.name == svc.name {
					continue
				}
				// Check if env value references another service by name
				if strings.Contains(envVal, otherSvc.name) {
					edgeKey := svc.name + "->" + otherSvc.name
					if !edgeSet[edgeKey] {
						edgeSet[edgeKey] = true
						edges = append(edges, model.Edge{
							ID:     fmt.Sprintf("env-%s-%s", svc.name, otherSvc.name),
							Source: svc.name,
							Target: otherSvc.name,
						})
					}
				}
			}
		}
	}

	// Infer edges from links
	for _, svc := range services {
		for _, link := range svc.links {
			target := strings.Split(link, ":")[0]
			if nodeIDs[target] && target != svc.name {
				edgeKey := svc.name + "->" + target
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edges = append(edges, model.Edge{
						ID:     fmt.Sprintf("link-%s-%s", svc.name, target),
						Source: svc.name,
						Target: target,
					})
				}
			}
		}
	}

	resp.Graph = model.Graph{Nodes: nodes, Edges: edges}
	resp.ResourceCount = len(nodes)
	return resp, nil
}

type composeSvc struct {
	name        string
	image       string
	dependsOn   []string
	ports       []string
	envValues   []string
	replicas    int
	cpuLimit    string
	memoryLimit string
	volumes     []string
	healthcheck string
	networks    []string
	links       []string
}

func parseComposeServices(content string) []composeSvc {
	services := []composeSvc{}
	lines := strings.Split(content, "\n")

	inServices := false
	inService := false
	currentSvc := composeSvc{}
	inDependsOn := false
	inEnvironment := false
	inPorts := false
	inVolumes := false
	inNetworks := false
	inDeploy := false
	inDeployResources := false
	inDeployLimits := false
	inLinks := false
	inHealthcheck := false

	resetSubSections := func() {
		inDependsOn = false
		inEnvironment = false
		inPorts = false
		inVolumes = false
		inNetworks = false
		inDeploy = false
		inDeployResources = false
		inDeployLimits = false
		inLinks = false
		inHealthcheck = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Top-level "services:" key
		if indent == 0 && (trimmed == "services:" || strings.HasPrefix(trimmed, "services:")) {
			inServices = true
			continue
		}

		// Another top-level key means we left services
		if indent == 0 && !strings.HasPrefix(trimmed, " ") && trimmed != "services:" && strings.HasSuffix(trimmed, ":") {
			if inService && currentSvc.name != "" {
				services = append(services, currentSvc)
			}
			inServices = false
			inService = false
			continue
		}

		if !inServices {
			continue
		}

		// Service name (indent 2)
		if indent == 2 && strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			if inService && currentSvc.name != "" {
				services = append(services, currentSvc)
			}
			currentSvc = composeSvc{name: strings.TrimSuffix(trimmed, ":")}
			inService = true
			resetSubSections()
			continue
		}

		if !inService {
			continue
		}

		// Service-level keys (indent 4)
		if indent == 4 {
			if strings.HasPrefix(trimmed, "image:") {
				currentSvc.image = strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
				resetSubSections()
				continue
			}
			if trimmed == "depends_on:" {
				resetSubSections()
				inDependsOn = true
				continue
			}
			if trimmed == "environment:" {
				resetSubSections()
				inEnvironment = true
				continue
			}
			if trimmed == "ports:" {
				resetSubSections()
				inPorts = true
				continue
			}
			if trimmed == "volumes:" {
				resetSubSections()
				inVolumes = true
				continue
			}
			if trimmed == "networks:" {
				resetSubSections()
				inNetworks = true
				continue
			}
			if trimmed == "deploy:" {
				resetSubSections()
				inDeploy = true
				continue
			}
			if trimmed == "links:" {
				resetSubSections()
				inLinks = true
				continue
			}
			if trimmed == "healthcheck:" {
				resetSubSections()
				inHealthcheck = true
				continue
			}
			// Inline depends_on: [a, b]
			if strings.HasPrefix(trimmed, "depends_on:") && strings.Contains(trimmed, "[") {
				resetSubSections()
				listPart := trimmed[strings.Index(trimmed, "[")+1:]
				listPart = strings.TrimSuffix(listPart, "]")
				for _, d := range strings.Split(listPart, ",") {
					d = strings.TrimSpace(d)
					d = strings.Trim(d, `"'`)
					if d != "" {
						currentSvc.dependsOn = append(currentSvc.dependsOn, d)
					}
				}
				continue
			}
			resetSubSections()
			continue
		}

		// Sub-items (indent 6+)
		if indent >= 6 {
			if inDependsOn {
				if strings.HasPrefix(trimmed, "- ") {
					dep := strings.TrimPrefix(trimmed, "- ")
					dep = strings.TrimSpace(dep)
					dep = strings.Split(dep, ":")[0] // handle condition syntax
					currentSvc.dependsOn = append(currentSvc.dependsOn, dep)
				} else if indent == 6 && strings.HasSuffix(trimmed, ":") {
					// depends_on with condition: { service: { condition: ... } }
					dep := strings.TrimSuffix(trimmed, ":")
					currentSvc.dependsOn = append(currentSvc.dependsOn, dep)
				}
				continue
			}
			if inEnvironment {
				if strings.HasPrefix(trimmed, "- ") {
					envLine := strings.TrimPrefix(trimmed, "- ")
					envLine = strings.Trim(envLine, `"'`)
					currentSvc.envValues = append(currentSvc.envValues, envLine)
				} else if strings.Contains(trimmed, ":") {
					// KEY: value format
					parts := strings.SplitN(trimmed, ":", 2)
					if len(parts) == 2 {
						currentSvc.envValues = append(currentSvc.envValues, strings.TrimSpace(parts[0])+"="+strings.TrimSpace(parts[1]))
					}
				}
				continue
			}
			if inPorts {
				if strings.HasPrefix(trimmed, "- ") {
					port := strings.TrimPrefix(trimmed, "- ")
					port = strings.Trim(port, `"'`)
					currentSvc.ports = append(currentSvc.ports, port)
				}
				continue
			}
			if inVolumes {
				if strings.HasPrefix(trimmed, "- ") {
					vol := strings.TrimPrefix(trimmed, "- ")
					vol = strings.Trim(vol, `"'`)
					currentSvc.volumes = append(currentSvc.volumes, vol)
				}
				continue
			}
			if inNetworks {
				if strings.HasPrefix(trimmed, "- ") {
					net := strings.TrimPrefix(trimmed, "- ")
					currentSvc.networks = append(currentSvc.networks, strings.TrimSpace(net))
				}
				continue
			}
			if inLinks {
				if strings.HasPrefix(trimmed, "- ") {
					link := strings.TrimPrefix(trimmed, "- ")
					currentSvc.links = append(currentSvc.links, strings.TrimSpace(link))
				}
				continue
			}
			if inHealthcheck {
				if strings.HasPrefix(trimmed, "test:") {
					currentSvc.healthcheck = strings.TrimSpace(strings.TrimPrefix(trimmed, "test:"))
				}
				continue
			}
			if inDeploy {
				if strings.HasPrefix(trimmed, "replicas:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "replicas:"))
					n := 0
					fmt.Sscanf(val, "%d", &n)
					currentSvc.replicas = n
				}
				if trimmed == "resources:" {
					inDeployResources = true
				}
				if inDeployResources && trimmed == "limits:" {
					inDeployLimits = true
				}
				if inDeployLimits {
					if strings.HasPrefix(trimmed, "cpus:") {
						currentSvc.cpuLimit = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "cpus:")), `"'`)
					}
					if strings.HasPrefix(trimmed, "memory:") {
						currentSvc.memoryLimit = strings.TrimSpace(strings.TrimPrefix(trimmed, "memory:"))
					}
				}
				continue
			}
		}
	}

	if inService && currentSvc.name != "" {
		services = append(services, currentSvc)
	}
	return services
}

func inferServiceType(svc composeSvc) string {
	nameLower := strings.ToLower(svc.name)
	imageLower := strings.ToLower(svc.image)
	combined := nameLower + " " + imageLower

	// Database patterns
	dbPatterns := regexp.MustCompile(`postgres|mysql|mariadb|mongo|dynamodb|cockroach|couchdb|orientdb|neo4j|mssql|oracle`)
	if dbPatterns.MatchString(combined) {
		return "database"
	}

	// Cache patterns
	cachePatterns := regexp.MustCompile(`redis|memcached|valkey|dragonfly`)
	if cachePatterns.MatchString(combined) {
		return "cache"
	}

	// Queue patterns
	queuePatterns := regexp.MustCompile(`rabbitmq|activemq|nats|sqs|zeromq`)
	if queuePatterns.MatchString(combined) {
		return "queue"
	}

	// Stream patterns
	streamPatterns := regexp.MustCompile(`kafka|redpanda|pulsar|kinesis`)
	if streamPatterns.MatchString(combined) {
		return "stream"
	}

	// Load balancer / proxy
	lbPatterns := regexp.MustCompile(`nginx|haproxy|traefik|envoy|caddy|kong|ambassador`)
	if lbPatterns.MatchString(combined) {
		return "load-balancer"
	}

	// Monitoring / observability
	monPatterns := regexp.MustCompile(`prometheus|grafana|jaeger|zipkin|datadog|newrelic|elastic|kibana|fluentd|logstash|loki|tempo`)
	if monPatterns.MatchString(combined) {
		return "monitoring"
	}

	// Storage (object storage, minio)
	storagePatterns := regexp.MustCompile(`minio|localstack|s3|azurite`)
	if storagePatterns.MatchString(combined) {
		return "storage"
	}

	// Search engine
	searchPatterns := regexp.MustCompile(`elasticsearch|opensearch|solr|meilisearch|typesense`)
	if searchPatterns.MatchString(combined) {
		return "database"
	}

	// Notification
	notifPatterns := regexp.MustCompile(`mailhog|mailpit|smtp`)
	if notifPatterns.MatchString(combined) {
		return "notification"
	}

	return "service"
}
