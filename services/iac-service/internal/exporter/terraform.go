package exporter

import (
	"fmt"
	"strings"

	"moktopus/iac-service/internal/model"
)

// ExportTerraform generates Terraform HCL from a CTM graph
func ExportTerraform(graph model.Graph, provider, region string) *model.ExportResponse {
	var sb strings.Builder

	// Provider block
	switch provider {
	case "aws":
		sb.WriteString(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

`)
	case "gcp":
		sb.WriteString(`terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

`)
	case "azure":
		sb.WriteString(`terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

`)
	}

	// Generate resources from nodes
	for _, node := range graph.Nodes {
		sb.WriteString(generateTerraformResource(node, provider, region))
		sb.WriteString("\n")
	}

	// Generate variables.tf
	variablesTf := generateVariablesTf(graph, provider, region)

	// Generate outputs.tf
	outputsTf := generateOutputsTf(graph, provider)

	// Generate terraform.tfvars.example
	tfvarsExample := generateTfvarsExample(provider, region)

	mainContent := sb.String()
	return &model.ExportResponse{
		Format:  "terraform",
		Content: mainContent,
		Files: []model.File{
			{Name: "main.tf", Content: mainContent},
			{Name: "variables.tf", Content: variablesTf},
			{Name: "outputs.tf", Content: outputsTf},
			{Name: "terraform.tfvars.example", Content: tfvarsExample},
		},
	}
}

func generateVariablesTf(graph model.Graph, provider, region string) string {
	var sb strings.Builder
	sb.WriteString(`variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "region" {
  description = "Cloud region for deployment"
  type        = string
  default     = "` + region + `"
}

`)

	if provider == "gcp" {
		sb.WriteString(`variable "project_id" {
  description = "GCP project ID"
  type        = string
}

`)
	}

	// Check if there are database nodes - add password var
	for _, node := range graph.Nodes {
		if node.Type == "database" {
			sb.WriteString(`variable "db_password" {
  description = "Database master password"
  type        = string
  sensitive   = true
}

`)
			break
		}
	}

	// Add instance type variables per node
	for _, node := range graph.Nodes {
		name := sanitizeName(node.ID)
		instanceType := getConfigString(node.Config, "instanceType", "")
		if instanceType == "" {
			instanceType = getConfigString(node.Config, "instanceClass", "")
		}
		if instanceType != "" {
			sb.WriteString(fmt.Sprintf(`variable "%s_instance_type" {
  description = "Instance type for %s"
  type        = string
  default     = "%s"
}

`, name, node.ID, instanceType))
		}
	}

	return sb.String()
}

func generateOutputsTf(graph model.Graph, provider string) string {
	var sb strings.Builder

	for _, node := range graph.Nodes {
		name := sanitizeName(node.ID)
		switch node.Type {
		case "load-balancer":
			if provider == "aws" {
				sb.WriteString(fmt.Sprintf(`output "%s_dns_name" {
  description = "DNS name of the load balancer"
  value       = aws_lb.%s.dns_name
}

`, name, name))
			}
		case "api-gateway":
			if provider == "aws" {
				sb.WriteString(fmt.Sprintf(`output "%s_endpoint" {
  description = "API Gateway endpoint URL"
  value       = aws_apigatewayv2_api.%s.api_endpoint
}

`, name, name))
			}
		case "database":
			if provider == "aws" {
				sb.WriteString(fmt.Sprintf(`output "%s_endpoint" {
  description = "Database connection endpoint"
  value       = aws_db_instance.%s.endpoint
  sensitive   = true
}

`, name, name))
			}
		case "storage":
			if provider == "aws" {
				sb.WriteString(fmt.Sprintf(`output "%s_bucket_name" {
  description = "S3 bucket name"
  value       = aws_s3_bucket.%s.id
}

`, name, name))
			}
		}
	}

	return sb.String()
}

func generateTfvarsExample(provider, region string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`# Terraform variables — copy to terraform.tfvars and fill in values
environment = "production"
region      = "%s"
`, region))

	if provider == "gcp" {
		sb.WriteString(`project_id  = "my-project-id"
`)
	}

	sb.WriteString(`
# Sensitive values (do not commit to version control)
# db_password = "your-secure-password-here"
`)
	return sb.String()
}

func generateTerraformResource(node model.Node, provider, region string) string {
	name := sanitizeName(node.ID)

	switch node.Type {
	case "service":
		return generateComputeResource(name, node, provider)
	case "database":
		return generateDatabaseResource(name, node, provider)
	case "cache":
		return generateCacheResource(name, node, provider)
	case "queue":
		return generateQueueResource(name, node, provider)
	case "load-balancer":
		return generateLBResource(name, node, provider)
	case "api-gateway":
		return generateAPIGatewayResource(name, node, provider)
	case "storage", "object-store":
		return generateStorageResource(name, node, provider)
	case "cdn":
		return generateCDNResource(name, node, provider)
	case "worker":
		return generateComputeResource(name, node, provider)
	case "serverless", "lambda", "function":
		return generateServerlessResource(name, node, provider)
	case "dns":
		return generateDNSResource(name, node, provider)
	case "firewall", "waf":
		return generateFirewallResource(name, node, provider)
	case "notification", "sns", "pubsub":
		return generateNotificationResource(name, node, provider)
	case "container", "kubernetes", "k8s":
		return generateContainerResource(name, node, provider)
	case "vpn":
		return generateVPNResource(name, node, provider)
	case "vpc":
		return generateVPCResource(name, node, provider)
	case "subnet":
		return generateSubnetResource(name, node, provider)
	case "internet-gateway":
		return generateIGWResource(name, node, provider)
	case "nat-gateway":
		return generateNATResource(name, node, provider)
	case "security-group":
		return generateSGResource(name, node, provider)
	case "route-table":
		return generateRouteTableResource(name, node, provider)
	case "elastic-ip":
		return generateEIPResource(name, node, provider)
	case "stream":
		return generateStreamResource(name, node, provider)
	case "monitoring":
		return generateMonitoringResource(name, node, provider)
	case "iam":
		return generateIAMResource(name, node, provider)
	default:
		return generateComputeResource(name, node, provider)
	}
}

func generateComputeResource(name string, node model.Node, provider string) string {
	instanceType := getConfigString(node.Config, "instanceType", "")

	switch provider {
	case "aws":
		if instanceType == "" {
			instanceType = "t3.medium"
		}
		return fmt.Sprintf(`resource "aws_ecs_service" "%s" {
  name            = "%s"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.%s.arn
  desired_count   = 2
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnets
    security_groups  = [aws_security_group.%s.id]
    assign_public_ip = false
  }
}

`, name, name, name, name)

	case "gcp":
		return fmt.Sprintf(`resource "google_cloud_run_service" "%s" {
  name     = "%s"
  location = var.region

  template {
    spec {
      containers {
        image = "gcr.io/${var.project}/%s:latest"
        resources {
          limits = {
            cpu    = "1000m"
            memory = "512Mi"
          }
        }
      }
    }
  }
}

`, name, name, name)

	case "azure":
		return fmt.Sprintf(`resource "azurerm_container_app" "%s" {
  name                         = "%s"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"

  template {
    container {
      name   = "%s"
      image  = "acr.azurecr.io/%s:latest"
      cpu    = 0.5
      memory = "1Gi"
    }
  }
}

`, name, name, name, name)
	}
	return ""
}

func generateDatabaseResource(name string, node model.Node, provider string) string {
	engine := getConfigString(node.Config, "engine", "postgresql")
	instanceClass := getConfigString(node.Config, "instanceClass", "")

	switch provider {
	case "aws":
		if instanceClass == "" {
			instanceClass = "db.t4g.medium"
		}
		return fmt.Sprintf(`resource "aws_db_instance" "%s" {
  identifier           = "%s"
  engine               = "%s"
  engine_version       = "16"
  instance_class       = "%s"
  allocated_storage    = 20
  max_allocated_storage = 100
  storage_type         = "gp3"
  db_name              = "%s"
  username             = "admin"
  password             = var.db_password
  skip_final_snapshot  = true
  multi_az             = true

  vpc_security_group_ids = [aws_security_group.db.id]
  db_subnet_group_name   = aws_db_subnet_group.main.name
}

`, name, name, engine, instanceClass, name)

	case "gcp":
		return fmt.Sprintf(`resource "google_sql_database_instance" "%s" {
  name             = "%s"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    tier = "db-custom-2-8192"
    ip_configuration {
      private_network = google_compute_network.main.id
    }
    backup_configuration {
      enabled = true
    }
  }
}

`, name, name)

	case "azure":
		return fmt.Sprintf(`resource "azurerm_postgresql_flexible_server" "%s" {
  name                   = "%s"
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = "16"
  sku_name               = "GP_Standard_D2s_v3"
  storage_mb             = 32768
  zone                   = "1"
  backup_retention_days  = 7
}

`, name, name)
	}
	return ""
}

func generateCacheResource(name string, node model.Node, provider string) string {
	nodeType := getConfigString(node.Config, "nodeType", "")

	switch provider {
	case "aws":
		if nodeType == "" {
			nodeType = "cache.t4g.medium"
		}
		return fmt.Sprintf(`resource "aws_elasticache_replication_group" "%s" {
  replication_group_id = "%s"
  description          = "%s Redis cluster"
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = "%s"
  num_cache_clusters   = 2
  port                 = 6379

  subnet_group_name  = aws_elasticache_subnet_group.main.name
  security_group_ids = [aws_security_group.cache.id]
}

`, name, name, name, nodeType)

	case "gcp":
		return fmt.Sprintf(`resource "google_redis_instance" "%s" {
  name           = "%s"
  tier           = "STANDARD_HA"
  memory_size_gb = 4
  region         = var.region
  redis_version  = "REDIS_7_0"

  authorized_network = google_compute_network.main.id
}

`, name, name)

	case "azure":
		return fmt.Sprintf(`resource "azurerm_redis_cache" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  capacity            = 2
  family              = "C"
  sku_name            = "Standard"
  enable_non_ssl_port = false
  minimum_tls_version = "1.2"
}

`, name, name)
	}
	return ""
}

func generateQueueResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_sqs_queue" "%s" {
  name                       = "%s"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 345600
  receive_wait_time_seconds  = 10
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_pubsub_topic" "%s" {
  name = "%s"
}

resource "google_pubsub_subscription" "%s_sub" {
  name  = "%s-sub"
  topic = google_pubsub_topic.%s.name
  ack_deadline_seconds = 20
}

`, name, name, name, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_servicebus_queue" "%s" {
  name         = "%s"
  namespace_id = azurerm_servicebus_namespace.main.id
}

`, name, name)
	}
	return ""
}

func generateLBResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_lb" "%s" {
  name               = "%s"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.lb.id]
  subnets            = var.public_subnets
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_global_forwarding_rule" "%s" {
  name       = "%s"
  target     = google_compute_target_http_proxy.%s.id
  port_range = "80"
}

`, name, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_lb" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard"
}

`, name, name)
	}
	return ""
}

func generateAPIGatewayResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_apigatewayv2_api" "%s" {
  name          = "%s"
  protocol_type = "HTTP"
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_api_gateway_api" "%s" {
  provider = google-beta
  api_id   = "%s"
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_api_management" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  publisher_name      = "Moktopus"
  publisher_email     = "admin@example.com"
  sku_name            = "Developer_1"
}

`, name, name)
	}
	return ""
}

func generateStorageResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_s3_bucket" "%s" {
  bucket = "%s"
}

resource "aws_s3_bucket_versioning" "%s" {
  bucket = aws_s3_bucket.%s.id
  versioning_configuration {
    status = "Enabled"
  }
}

`, name, name, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_storage_bucket" "%s" {
  name     = "%s"
  location = var.region
  uniform_bucket_level_access = true
  versioning {
    enabled = true
  }
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_storage_account" "%s" {
  name                     = "%s"
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

`, name, name)
	}
	return ""
}

func generateCDNResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_cloudfront_distribution" "%s" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "%s CDN distribution"
  default_root_object = "index.html"

  origin {
    domain_name = "origin.example.com"
    origin_id   = "%s-origin"
  }

  default_cache_behavior {
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "%s-origin"
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

`, name, name, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_backend_bucket" "%s" {
  name        = "%s"
  bucket_name = google_storage_bucket.%s.name
  enable_cdn  = true
}

`, name, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_cdn_profile" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard_Microsoft"
}

resource "azurerm_cdn_endpoint" "%s_endpoint" {
  name                = "%s-endpoint"
  profile_name        = azurerm_cdn_profile.%s.name
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  origin {
    name      = "origin"
    host_name = "origin.example.com"
  }
}

`, name, name, name, name, name)
	}
	return ""
}

func generateServerlessResource(name string, node model.Node, provider string) string {
	runtime := getConfigString(node.Config, "runtime", "nodejs18.x")
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_lambda_function" "%s" {
  function_name = "%s"
  role          = aws_iam_role.lambda_exec.arn
  handler       = "index.handler"
  runtime       = "%s"
  memory_size   = 256
  timeout       = 30

  environment {
    variables = {
      ENVIRONMENT = "production"
    }
  }
}

`, name, name, runtime)
	case "gcp":
		return fmt.Sprintf(`resource "google_cloudfunctions2_function" "%s" {
  name     = "%s"
  location = var.region

  build_config {
    runtime     = "nodejs18"
    entry_point = "handler"
  }

  service_config {
    max_instance_count = 100
    available_memory   = "256M"
    timeout_seconds    = 30
  }
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_linux_function_app" "%s" {
  name                = "%s"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  service_plan_id     = azurerm_service_plan.main.id

  site_config {
    application_stack {
      node_version = "18"
    }
  }
}

`, name, name)
	}
	return ""
}

func generateDNSResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_route53_zone" "%s" {
  name = "%s.example.com"
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_dns_managed_zone" "%s" {
  name     = "%s"
  dns_name = "%s.example.com."
}

`, name, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_dns_zone" "%s" {
  name                = "%s.example.com"
  resource_group_name = azurerm_resource_group.main.name
}

`, name, name)
	}
	return ""
}

func generateFirewallResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_wafv2_web_acl" "%s" {
  name  = "%s"
  scope = "REGIONAL"

  default_action {
    allow {}
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "%s"
    sampled_requests_enabled   = true
  }
}

`, name, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_security_policy" "%s" {
  name = "%s"

  rule {
    action   = "allow"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
  }
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_web_application_firewall_policy" "%s" {
  name                = "%s"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location

  managed_rules {
    managed_rule_set {
      version = "3.2"
    }
  }

  policy_settings {
    enabled = true
    mode    = "Prevention"
  }
}

`, name, name)
	}
	return ""
}

func generateNotificationResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_sns_topic" "%s" {
  name = "%s"
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_pubsub_topic" "%s" {
  name = "%s"
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_notification_hub" "%s" {
  name                = "%s"
  namespace_name      = azurerm_notification_hub_namespace.main.name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
}

`, name, name)
	}
	return ""
}

func generateContainerResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_eks_cluster" "%s" {
  name     = "%s"
  role_arn = aws_iam_role.eks.arn

  vpc_config {
    subnet_ids = var.private_subnets
  }
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_container_cluster" "%s" {
  name     = "%s"
  location = var.region

  initial_node_count = 3

  node_config {
    machine_type = "e2-medium"
  }
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_kubernetes_cluster" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = "%s"

  default_node_pool {
    name       = "default"
    node_count = 3
    vm_size    = "Standard_D2_v2"
  }

  identity {
    type = "SystemAssigned"
  }
}

`, name, name, name)
	}
	return ""
}

func generateVPNResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_vpn_gateway" "%s" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "%s"
  }
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_vpn_gateway" "%s" {
  name    = "%s"
  network = google_compute_network.main.id
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_virtual_network_gateway" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  type                = "Vpn"
  sku                 = "VpnGw1"

  ip_configuration {
    public_ip_address_id = azurerm_public_ip.vpn.id
    subnet_id            = azurerm_subnet.gateway.id
  }
}

`, name, name)
	}
	return ""
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return strings.ToLower(name)
}

func generateVPCResource(name string, node model.Node, provider string) string {
	cidr := getConfigString(node.Config, "cidrBlock", "10.0.0.0/16")
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_vpc" "%s" {
  cidr_block           = "%s"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "%s"
  }
}

`, name, cidr, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_network" "%s" {
  name                    = "%s"
  auto_create_subnetworks = false
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_virtual_network" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  address_space       = ["%s"]
}

`, name, name, cidr)
	}
	return ""
}

func generateSubnetResource(name string, node model.Node, provider string) string {
	cidr := getConfigString(node.Config, "cidrBlock", "10.0.1.0/24")
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_subnet" "%s" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "%s"
  availability_zone = "%s"

  tags = {
    Name = "%s"
  }
}

`, name, cidr, getConfigString(node.Config, "availabilityZone", "us-east-1a"), name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_subnetwork" "%s" {
  name          = "%s"
  ip_cidr_range = "%s"
  network       = google_compute_network.main.id
  region        = var.region
}

`, name, name, cidr)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_subnet" "%s" {
  name                 = "%s"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["%s"]
}

`, name, name, cidr)
	}
	return ""
}

func generateIGWResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_internet_gateway" "%s" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "%s"
  }
}

`, name, name)
	case "gcp":
		return "# GCP uses default internet gateway\n\n"
	case "azure":
		return "# Azure uses implicit internet gateway\n\n"
	}
	return ""
}

func generateNATResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_nat_gateway" "%s" {
  allocation_id = aws_eip.%s.id
  subnet_id     = aws_subnet.public.id

  tags = {
    Name = "%s"
  }
}

`, name, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_router_nat" "%s" {
  name                               = "%s"
  router                             = google_compute_router.main.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_nat_gateway" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku_name            = "Standard"
}

`, name, name)
	}
	return ""
}

func generateSGResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_security_group" "%s" {
  name   = "%s"
  vpc_id = aws_vpc.main.id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "%s"
  }
}

`, name, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_firewall" "%s" {
  name    = "%s"
  network = google_compute_network.main.name

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_network_security_group" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  security_rule {
    name                       = "AllowHTTPS"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

`, name, name)
	}
	return ""
}

func generateRouteTableResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_route_table" "%s" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "%s"
  }
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_route" "%s" {
  name         = "%s"
  dest_range   = "0.0.0.0/0"
  network      = google_compute_network.main.name
  next_hop_gateway = "default-internet-gateway"
  priority     = 1000
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_route_table" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

`, name, name)
	}
	return ""
}

func generateEIPResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_eip" "%s" {
  domain = "vpc"

  tags = {
    Name = "%s"
  }
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_compute_address" "%s" {
  name   = "%s"
  region = var.region
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_public_ip" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

`, name, name)
	}
	return ""
}

func generateStreamResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		shardCount := getConfigString(node.Config, "shardCount", "1")
		return fmt.Sprintf(`resource "aws_kinesis_stream" "%s" {
  name             = "%s"
  shard_count      = %s
  retention_period = 24
}

`, name, name, shardCount)
	case "gcp":
		return fmt.Sprintf(`resource "google_pubsub_topic" "%s" {
  name = "%s"
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_eventhub_namespace" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard"
  capacity            = 1
}

`, name, name)
	}
	return ""
}

func generateMonitoringResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_cloudwatch_log_group" "%s" {
  name              = "/app/%s"
  retention_in_days = 30
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_monitoring_alert_policy" "%s" {
  display_name = "%s"
  combiner     = "OR"

  conditions {
    display_name = "condition"
    condition_threshold {
      filter          = "resource.type=\"gce_instance\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
      duration        = "60s"
    }
  }
}

`, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_log_analytics_workspace" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

`, name, name)
	}
	return ""
}

func generateIAMResource(name string, node model.Node, provider string) string {
	switch provider {
	case "aws":
		return fmt.Sprintf(`resource "aws_iam_role" "%s" {
  name = "%s"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

`, name, name)
	case "gcp":
		return fmt.Sprintf(`resource "google_service_account" "%s" {
  account_id   = "%s"
  display_name = "%s"
}

`, name, name, name)
	case "azure":
		return fmt.Sprintf(`resource "azurerm_user_assigned_identity" "%s" {
  name                = "%s"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

`, name, name)
	}
	return ""
}

func getConfigString(config map[string]any, key, fallback string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}
