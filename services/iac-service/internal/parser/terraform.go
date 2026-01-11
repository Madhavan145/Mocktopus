package parser

import (
	"fmt"
	"regexp"
	"strings"

	"moktopus/iac-service/internal/model"
)

// parsedResource holds extracted data for a single Terraform resource block.
type parsedResource struct {
	resourceType string
	name         string
	blockContent string
}

// extractBlock extracts the content between balanced braces starting from offset.
func extractBlock(content string, offset int) (string, int) {
	start := strings.Index(content[offset:], "{")
	if start == -1 {
		return "", -1
	}
	start += offset
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(content); i++ {
		ch := content[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				return content[start+1 : i], i + 1
			}
		}
	}
	return content[start+1:], len(content)
}

// ParseTerraform parses Terraform HCL content into a CTM graph.
func ParseTerraform(content string) (*model.ImportResponse, error) {
	resp := &model.ImportResponse{
		Format:   "terraform",
		Warnings: []string{},
	}

	// Pre-processing: strip comments and resolve variables
	content = stripTerraformComments(content)
	variables := extractVariableDefaults(content)
	content = resolveVariableReferences(content, variables)

	nodes := []model.Node{}
	edges := []model.Edge{}
	nodeIDs := map[string]bool{}
	edgeSet := map[string]bool{}

	// Parse provider
	providerRe := regexp.MustCompile(`provider\s+"(aws|google|azurerm)"`)
	if matches := providerRe.FindStringSubmatch(content); len(matches) > 1 {
		switch matches[1] {
		case "aws":
			resp.Provider = "aws"
		case "google":
			resp.Provider = "gcp"
		case "azurerm":
			resp.Provider = "azure"
		}
	}

	// Parse region
	regionRe := regexp.MustCompile(`region\s*=\s*"([^"]+)"`)
	if matches := regionRe.FindStringSubmatch(content); len(matches) > 1 {
		resp.Region = matches[1]
	}

	// Extract all resource blocks
	resourceRe := regexp.MustCompile(`resource\s+"([^"]+)"\s+"([^"]+)"`)
	resourceMatches := resourceRe.FindAllStringSubmatchIndex(content, -1)

	resources := make([]parsedResource, 0, len(resourceMatches))
	for _, match := range resourceMatches {
		resourceType := content[match[2]:match[3]]
		resourceName := content[match[4]:match[5]]
		blockContent, _ := extractBlock(content, match[0])
		if blockContent == "" {
			continue
		}
		resources = append(resources, parsedResource{
			resourceType: resourceType,
			name:         resourceName,
			blockContent: blockContent,
		})
	}

	// Phase 1: Create nodes
	for _, res := range resources {
		nodeType, config := mapTerraformResource(res.resourceType, res.name, res.blockContent)
		if nodeType == "" {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("unsupported resource: %s.%s", res.resourceType, res.name))
			continue
		}
		if !nodeIDs[res.name] {
			nodes = append(nodes, model.Node{
				ID:     res.name,
				Type:   nodeType,
				Config: config,
			})
			nodeIDs[res.name] = true
		}
	}

	// Phase 2: Detect edges
	dependsRe := regexp.MustCompile(`depends_on\s*=\s*\[([^\]]+)\]`)
	refRe := regexp.MustCompile(`(?:aws|google|azurerm)_\w+\.(\w+)(?:\.\w+)?`)

	for _, res := range resources {
		sourceID := res.name
		if !nodeIDs[sourceID] {
			continue
		}

		for _, depMatch := range dependsRe.FindAllStringSubmatch(res.blockContent, -1) {
			deps := strings.Split(depMatch[1], ",")
			for _, dep := range deps {
				dep = strings.TrimSpace(dep)
				dep = strings.Trim(dep, `"`)
				parts := strings.Split(dep, ".")
				if len(parts) >= 2 {
					targetID := parts[1]
					if nodeIDs[targetID] && targetID != sourceID {
						edgeKey := sourceID + "->" + targetID
						if !edgeSet[edgeKey] {
							edgeSet[edgeKey] = true
							edges = append(edges, model.Edge{
								ID:     fmt.Sprintf("dep-%s-%s", sourceID, targetID),
								Source: sourceID,
								Target: targetID,
							})
						}
					}
				}
			}
		}

		for _, refMatch := range refRe.FindAllStringSubmatch(res.blockContent, -1) {
			targetID := refMatch[1]
			if nodeIDs[targetID] && targetID != sourceID {
				edgeKey := sourceID + "->" + targetID
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edges = append(edges, model.Edge{
						ID:     fmt.Sprintf("ref-%s-%s", sourceID, targetID),
						Source: sourceID,
						Target: targetID,
					})
				}
			}
		}
	}

	resp.Graph = model.Graph{Nodes: nodes, Edges: edges}
	resp.ResourceCount = len(nodes)
	return resp, nil
}

func tfExtract(block, pattern string) string {
	re := regexp.MustCompile(pattern)
	if m := re.FindStringSubmatch(block); len(m) > 1 {
		return m[1]
	}
	return ""
}

func mapTerraformResource(resourceType, name, block string) (string, map[string]any) {
	config := map[string]any{
		"label":       name,
		"iacResource": resourceType,
		"provider":    "aws",
	}
	if strings.HasPrefix(resourceType, "google_") {
		config["provider"] = "gcp"
	} else if strings.HasPrefix(resourceType, "azurerm_") {
		config["provider"] = "azure"
	}

	switch {
	// ═══ NETWORKING ═══
	case resourceType == "aws_vpc":
		if v := tfExtract(block, `cidr_block\s*=\s*"([^"]+)"`); v != "" {
			config["cidrBlock"] = v
		}
		return "vpc", config
	case resourceType == "aws_subnet":
		if v := tfExtract(block, `cidr_block\s*=\s*"([^"]+)"`); v != "" {
			config["cidrBlock"] = v
		}
		if v := tfExtract(block, `availability_zone\s*=\s*"([^"]+)"`); v != "" {
			config["availabilityZone"] = v
		}
		if v := tfExtract(block, `map_public_ip_on_launch\s*=\s*(true|false)`); v != "" {
			config["public"] = v == "true"
		}
		return "subnet", config
	case resourceType == "aws_internet_gateway":
		return "internet-gateway", config
	case resourceType == "aws_nat_gateway":
		return "nat-gateway", config
	case resourceType == "aws_route_table" || resourceType == "aws_route" || resourceType == "aws_route_table_association":
		return "route-table", config
	case resourceType == "aws_security_group":
		if v := tfExtract(block, `name\s*=\s*"([^"]+)"`); v != "" {
			config["sgName"] = v
		}
		return "security-group", config
	case resourceType == "aws_security_group_rule":
		return "security-group", config
	case resourceType == "aws_eip":
		return "elastic-ip", config
	case resourceType == "google_compute_network":
		return "vpc", config
	case resourceType == "google_compute_subnetwork":
		if v := tfExtract(block, `ip_cidr_range\s*=\s*"([^"]+)"`); v != "" {
			config["cidrBlock"] = v
		}
		return "subnet", config
	case resourceType == "google_compute_router" || resourceType == "google_compute_router_nat":
		return "nat-gateway", config
	case resourceType == "google_compute_firewall":
		return "security-group", config
	case resourceType == "azurerm_virtual_network":
		return "vpc", config
	case resourceType == "azurerm_subnet":
		return "subnet", config
	case resourceType == "azurerm_network_security_group":
		return "security-group", config
	case resourceType == "azurerm_public_ip":
		return "elastic-ip", config
	case resourceType == "azurerm_nat_gateway":
		return "nat-gateway", config

	// ═══ COMPUTE ═══
	case resourceType == "aws_instance":
		if v := tfExtract(block, `instance_type\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		if v := tfExtract(block, `ami\s*=\s*"([^"]+)"`); v != "" {
			config["ami"] = v
		}
		return "service", config
	case resourceType == "aws_ecs_cluster":
		if v := tfExtract(block, `name\s*=\s*"([^"]+)"`); v != "" {
			config["clusterName"] = v
		}
		return "service", config
	case resourceType == "aws_ecs_service":
		if v := tfExtract(block, `launch_type\s*=\s*"([^"]+)"`); v != "" {
			config["launchType"] = v
		}
		if v := tfExtract(block, `desired_count\s*=\s*(\d+)`); v != "" {
			config["desiredCount"] = v
		}
		if v := tfExtract(block, `name\s*=\s*"([^"]+)"`); v != "" {
			config["serviceName"] = v
		}
		return "service", config
	case resourceType == "aws_ecs_task_definition":
		if v := tfExtract(block, `cpu\s*=\s*"?(\d+)"?`); v != "" {
			config["cpu"] = v
		}
		if v := tfExtract(block, `memory\s*=\s*"?(\d+)"?`); v != "" {
			config["memory"] = v
		}
		if v := tfExtract(block, `network_mode\s*=\s*"([^"]+)"`); v != "" {
			config["networkMode"] = v
		}
		return "service", config
	case resourceType == "aws_autoscaling_group":
		if v := tfExtract(block, `min_size\s*=\s*(\d+)`); v != "" {
			config["minSize"] = v
		}
		if v := tfExtract(block, `max_size\s*=\s*(\d+)`); v != "" {
			config["maxSize"] = v
		}
		if v := tfExtract(block, `desired_capacity\s*=\s*(\d+)`); v != "" {
			config["desiredCapacity"] = v
		}
		return "service", config
	case resourceType == "aws_launch_template":
		if v := tfExtract(block, `instance_type\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "service", config
	case resourceType == "google_compute_instance":
		if v := tfExtract(block, `machine_type\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "service", config
	case resourceType == "google_cloud_run_service" || resourceType == "google_cloud_run_v2_service":
		return "service", config
	case resourceType == "google_compute_instance_group_manager":
		return "service", config
	case resourceType == "azurerm_linux_virtual_machine" || resourceType == "azurerm_windows_virtual_machine":
		if v := tfExtract(block, `size\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "service", config
	case resourceType == "azurerm_container_app":
		return "service", config

	// ═══ SERVERLESS ═══
	case resourceType == "aws_lambda_function":
		config["runtime"] = "serverless"
		if v := tfExtract(block, `runtime\s*=\s*"([^"]+)"`); v != "" {
			config["lambdaRuntime"] = v
		}
		if v := tfExtract(block, `memory_size\s*=\s*(\d+)`); v != "" {
			config["memorySize"] = v
		}
		if v := tfExtract(block, `timeout\s*=\s*(\d+)`); v != "" {
			config["timeout"] = v
		}
		return "serverless", config
	case resourceType == "aws_lambda_event_source_mapping":
		return "serverless", config
	case resourceType == "google_cloudfunctions2_function" || resourceType == "google_cloudfunctions_function":
		config["runtime"] = "serverless"
		return "serverless", config
	case resourceType == "azurerm_linux_function_app" || resourceType == "azurerm_windows_function_app":
		config["runtime"] = "serverless"
		return "serverless", config

	// ═══ DATABASE ═══
	case resourceType == "aws_db_instance" || resourceType == "aws_rds_cluster":
		if v := tfExtract(block, `engine\s*=\s*"([^"]+)"`); v != "" {
			config["engine"] = v
		}
		if v := tfExtract(block, `engine_version\s*=\s*"([^"]+)"`); v != "" {
			config["engineVersion"] = v
		}
		if v := tfExtract(block, `instance_class\s*=\s*"([^"]+)"`); v != "" {
			config["instanceClass"] = v
		}
		if v := tfExtract(block, `multi_az\s*=\s*(true|false)`); v != "" {
			config["multiAz"] = v == "true"
		}
		if v := tfExtract(block, `storage_type\s*=\s*"([^"]+)"`); v != "" {
			config["storageType"] = v
		}
		if v := tfExtract(block, `allocated_storage\s*=\s*(\d+)`); v != "" {
			config["allocatedStorage"] = v
		}
		return "database", config
	case resourceType == "aws_rds_cluster_instance":
		if v := tfExtract(block, `instance_class\s*=\s*"([^"]+)"`); v != "" {
			config["instanceClass"] = v
		}
		return "database", config
	case resourceType == "aws_dynamodb_table":
		if v := tfExtract(block, `billing_mode\s*=\s*"([^"]+)"`); v != "" {
			config["billingMode"] = v
		}
		return "database", config
	case resourceType == "google_sql_database_instance":
		if v := tfExtract(block, `database_version\s*=\s*"([^"]+)"`); v != "" {
			config["engine"] = v
		}
		if v := tfExtract(block, `tier\s*=\s*"([^"]+)"`); v != "" {
			config["instanceClass"] = v
		}
		return "database", config
	case resourceType == "google_spanner_instance" || resourceType == "google_firestore_database":
		return "database", config
	case resourceType == "azurerm_postgresql_flexible_server" || resourceType == "azurerm_mysql_flexible_server":
		if v := tfExtract(block, `sku_name\s*=\s*"([^"]+)"`); v != "" {
			config["instanceClass"] = v
		}
		return "database", config
	case resourceType == "azurerm_cosmosdb_account":
		return "database", config
	case resourceType == "azurerm_mssql_server" || resourceType == "azurerm_mssql_database":
		return "database", config

	// ═══ CACHE ═══
	case resourceType == "aws_elasticache_replication_group" || resourceType == "aws_elasticache_cluster":
		if v := tfExtract(block, `node_type\s*=\s*"([^"]+)"`); v != "" {
			config["nodeType"] = v
		}
		if v := tfExtract(block, `engine\s*=\s*"([^"]+)"`); v != "" {
			config["engine"] = v
		}
		if v := tfExtract(block, `num_cache_clusters\s*=\s*(\d+)`); v != "" {
			config["numClusters"] = v
		}
		return "cache", config
	case resourceType == "google_redis_instance":
		if v := tfExtract(block, `memory_size_gb\s*=\s*(\d+)`); v != "" {
			config["memorySizeGb"] = v
		}
		return "cache", config
	case resourceType == "azurerm_redis_cache":
		if v := tfExtract(block, `sku_name\s*=\s*"([^"]+)"`); v != "" {
			config["sku"] = v
		}
		return "cache", config

	// ═══ QUEUE / MESSAGING ═══
	case resourceType == "aws_sqs_queue":
		if v := tfExtract(block, `visibility_timeout_seconds\s*=\s*(\d+)`); v != "" {
			config["visibilityTimeout"] = v
		}
		if v := tfExtract(block, `fifo_queue\s*=\s*(true|false)`); v != "" {
			config["fifo"] = v == "true"
		}
		return "queue", config
	case resourceType == "aws_sns_topic":
		if v := tfExtract(block, `name\s*=\s*"([^"]+)"`); v != "" {
			config["topicName"] = v
		}
		return "notification", config
	case resourceType == "aws_sns_topic_subscription":
		return "notification", config
	case resourceType == "aws_kinesis_stream":
		if v := tfExtract(block, `shard_count\s*=\s*(\d+)`); v != "" {
			config["shardCount"] = v
		}
		return "stream", config
	case resourceType == "aws_msk_cluster":
		return "stream", config
	case resourceType == "google_pubsub_topic":
		return "notification", config
	case resourceType == "google_pubsub_subscription":
		return "queue", config
	case resourceType == "azurerm_servicebus_namespace" || resourceType == "azurerm_servicebus_queue":
		return "queue", config
	case resourceType == "azurerm_servicebus_topic":
		return "notification", config
	case resourceType == "azurerm_eventhub_namespace" || resourceType == "azurerm_eventhub":
		return "stream", config

	// ═══ LOAD BALANCER ═══
	case resourceType == "aws_lb" || resourceType == "aws_alb":
		if v := tfExtract(block, `load_balancer_type\s*=\s*"([^"]+)"`); v != "" {
			config["lbType"] = v
		}
		if v := tfExtract(block, `internal\s*=\s*(true|false)`); v != "" {
			config["internal"] = v == "true"
		}
		return "load-balancer", config
	case resourceType == "aws_lb_target_group" || resourceType == "aws_alb_target_group":
		if v := tfExtract(block, `port\s*=\s*(\d+)`); v != "" {
			config["port"] = v
		}
		if v := tfExtract(block, `protocol\s*=\s*"([^"]+)"`); v != "" {
			config["protocol"] = v
		}
		return "load-balancer", config
	case resourceType == "aws_lb_listener" || resourceType == "aws_alb_listener":
		if v := tfExtract(block, `port\s*=\s*"?(\d+)"?`); v != "" {
			config["port"] = v
		}
		return "load-balancer", config
	case strings.HasPrefix(resourceType, "google_compute_forwarding_rule"):
		return "load-balancer", config
	case resourceType == "google_compute_backend_service" || resourceType == "google_compute_url_map":
		return "load-balancer", config
	case resourceType == "azurerm_lb" || resourceType == "azurerm_application_gateway":
		return "load-balancer", config

	// ═══ API GATEWAY ═══
	case strings.HasPrefix(resourceType, "aws_apigatewayv2") || strings.HasPrefix(resourceType, "aws_api_gateway"):
		return "api-gateway", config
	case resourceType == "google_api_gateway_api":
		return "api-gateway", config
	case resourceType == "azurerm_api_management":
		return "api-gateway", config

	// ═══ CDN ═══
	case strings.HasPrefix(resourceType, "aws_cloudfront"):
		return "cdn", config
	case resourceType == "google_compute_backend_bucket":
		return "cdn", config
	case strings.HasPrefix(resourceType, "azurerm_cdn"):
		return "cdn", config

	// ═══ STORAGE ═══
	case resourceType == "aws_s3_bucket" || resourceType == "aws_s3_bucket_versioning" || resourceType == "aws_s3_bucket_policy":
		return "storage", config
	case resourceType == "google_storage_bucket":
		return "storage", config
	case resourceType == "azurerm_storage_account" || resourceType == "azurerm_storage_container":
		return "storage", config

	// ═══ DNS ═══
	case resourceType == "aws_route53_zone" || resourceType == "aws_route53_record":
		return "dns", config
	case resourceType == "google_dns_managed_zone" || resourceType == "google_dns_record_set":
		return "dns", config
	case resourceType == "azurerm_dns_zone" || resourceType == "azurerm_private_dns_zone":
		return "dns", config

	// ═══ WAF/FIREWALL ═══
	case strings.HasPrefix(resourceType, "aws_wafv2") || strings.HasPrefix(resourceType, "aws_waf"):
		return "firewall", config
	case resourceType == "google_compute_security_policy":
		return "firewall", config
	case resourceType == "azurerm_web_application_firewall_policy":
		return "firewall", config

	// ═══ CONTAINER / K8S ═══
	case resourceType == "aws_eks_cluster":
		return "kubernetes", config
	case resourceType == "aws_eks_node_group":
		if v := tfExtract(block, `instance_types\s*=\s*\["([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "kubernetes", config
	case resourceType == "google_container_cluster" || resourceType == "google_container_node_pool":
		if v := tfExtract(block, `machine_type\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "kubernetes", config
	case resourceType == "azurerm_kubernetes_cluster":
		if v := tfExtract(block, `vm_size\s*=\s*"([^"]+)"`); v != "" {
			config["instanceType"] = v
		}
		return "kubernetes", config

	// ═══ VPN ═══
	case resourceType == "aws_vpn_gateway" || resourceType == "aws_vpn_connection":
		return "vpn", config
	case resourceType == "google_compute_vpn_gateway" || resourceType == "google_compute_vpn_tunnel":
		return "vpn", config
	case resourceType == "azurerm_virtual_network_gateway":
		return "vpn", config

	// ═══ MONITORING ═══
	case resourceType == "aws_cloudwatch_log_group" || resourceType == "aws_cloudwatch_metric_alarm":
		return "monitoring", config
	case resourceType == "google_monitoring_alert_policy":
		return "monitoring", config
	case resourceType == "azurerm_log_analytics_workspace":
		return "monitoring", config

	// ═══ IAM ═══
	case resourceType == "aws_iam_role" || resourceType == "aws_iam_policy" ||
		resourceType == "aws_iam_role_policy_attachment" || resourceType == "aws_iam_instance_profile":
		return "iam", config
	case resourceType == "google_service_account" || resourceType == "google_project_iam_member":
		return "iam", config
	case resourceType == "azurerm_role_assignment":
		return "iam", config

	default:
		return "service", config
	}
}

// stripTerraformComments removes single-line (#, //) and block (/* */) comments from HCL content.
func stripTerraformComments(content string) string {
	// Remove block comments
	for {
		start := strings.Index(content, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(content[start:], "*/")
		if end == -1 {
			content = content[:start]
			break
		}
		content = content[:start] + content[start+end+2:]
	}

	// Remove single-line comments (# and //)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Don't strip inside strings
		inString := false
		for j := 0; j < len(line); j++ {
			if line[j] == '"' && (j == 0 || line[j-1] != '\\') {
				inString = !inString
			}
			if !inString {
				if line[j] == '#' {
					lines[i] = line[:j]
					break
				}
				if j < len(line)-1 && line[j] == '/' && line[j+1] == '/' {
					lines[i] = line[:j]
					break
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

// extractVariableDefaults parses variable blocks and extracts their default values.
func extractVariableDefaults(content string) map[string]string {
	vars := map[string]string{}
	varRe := regexp.MustCompile(`variable\s+"([^"]+)"`)
	matches := varRe.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		varName := content[match[2]:match[3]]
		block, _ := extractBlock(content, match[0])
		if block == "" {
			continue
		}
		// Extract default value
		defaultRe := regexp.MustCompile(`default\s*=\s*"([^"]*)"`)
		if m := defaultRe.FindStringSubmatch(block); len(m) > 1 {
			vars[varName] = m[1]
		}
	}
	return vars
}

// resolveVariableReferences replaces var.X references with their default values.
func resolveVariableReferences(content string, vars map[string]string) string {
	// Replace var.name patterns
	varRefRe := regexp.MustCompile(`var\.(\w+)`)
	content = varRefRe.ReplaceAllStringFunc(content, func(match string) string {
		name := strings.TrimPrefix(match, "var.")
		if val, ok := vars[name]; ok {
			return `"` + val + `"`
		}
		return match
	})

	// Replace ${var.name} inside strings
	interpRe := regexp.MustCompile(`\$\{var\.(\w+)\}`)
	content = interpRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := interpRe.FindStringSubmatch(match)
		if len(inner) > 1 {
			if val, ok := vars[inner[1]]; ok {
				return val
			}
		}
		return match
	})

	return content
}
