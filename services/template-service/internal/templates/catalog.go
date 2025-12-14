package templates

import "moktopus/template-service/internal/model"

// All returns all available architecture templates
var All = []model.Template{
	MicroservicesEcommerce,
	EventDrivenCQRS,
	ServerlessAPI,
	ThreeTierWeb,
	StreamProcessing,
	MultiRegionActive,
	ServiceMeshIstio,
	MLInferencePipeline,
}

var MicroservicesEcommerce = model.Template{
	ID:          "microservices-ecommerce",
	Name:        "Microservices E-Commerce",
	Description: "Production-grade e-commerce platform with API gateway, multiple services, caching, and async communication",
	Category:    "microservices",
	Tags:        []string{"production", "ecommerce", "api-gateway", "cache", "queue"},
	Complexity:  "intermediate",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "api-gateway", Type: "api-gateway", Config: map[string]any{"label": "API Gateway", "latencyMs": 5, "provider": "aws", "service": "ALB"}},
			{ID: "user-service", Type: "service", Config: map[string]any{"label": "User Service", "latencyMs": 15, "replicas": 2}},
			{ID: "product-service", Type: "service", Config: map[string]any{"label": "Product Service", "latencyMs": 20, "replicas": 3}},
			{ID: "order-service", Type: "service", Config: map[string]any{"label": "Order Service", "latencyMs": 25, "replicas": 2}},
			{ID: "payment-service", Type: "service", Config: map[string]any{"label": "Payment Service", "latencyMs": 50, "replicas": 2}},
			{ID: "notification-service", Type: "service", Config: map[string]any{"label": "Notification Service", "latencyMs": 10, "replicas": 1}},
			{ID: "user-db", Type: "database", Config: map[string]any{"label": "User DB", "engine": "postgresql", "latencyMs": 8}},
			{ID: "product-db", Type: "database", Config: map[string]any{"label": "Product DB", "engine": "postgresql", "latencyMs": 8}},
			{ID: "order-db", Type: "database", Config: map[string]any{"label": "Order DB", "engine": "postgresql", "latencyMs": 8}},
			{ID: "product-cache", Type: "cache", Config: map[string]any{"label": "Product Cache", "engine": "redis", "latencyMs": 2}},
			{ID: "session-cache", Type: "cache", Config: map[string]any{"label": "Session Cache", "engine": "redis", "latencyMs": 1}},
			{ID: "order-queue", Type: "queue", Config: map[string]any{"label": "Order Events", "latencyMs": 5}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "api-gateway", Target: "user-service"},
			{ID: "e2", Source: "api-gateway", Target: "product-service"},
			{ID: "e3", Source: "api-gateway", Target: "order-service"},
			{ID: "e4", Source: "user-service", Target: "user-db"},
			{ID: "e5", Source: "user-service", Target: "session-cache"},
			{ID: "e6", Source: "product-service", Target: "product-db"},
			{ID: "e7", Source: "product-service", Target: "product-cache"},
			{ID: "e8", Source: "order-service", Target: "order-db"},
			{ID: "e9", Source: "order-service", Target: "payment-service"},
			{ID: "e10", Source: "order-service", Target: "order-queue"},
			{ID: "e11", Source: "order-queue", Target: "notification-service"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 60, RequestsPerSecond: 500, Description: "Moderate e-commerce traffic with mix of reads and writes"},
}

var EventDrivenCQRS = model.Template{
	ID:          "event-driven-cqrs",
	Name:        "Event-Driven CQRS",
	Description: "Command Query Responsibility Segregation with event sourcing, separate read/write models, and eventual consistency",
	Category:    "event-driven",
	Tags:        []string{"cqrs", "event-sourcing", "kafka", "eventual-consistency"},
	Complexity:  "advanced",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "command-api", Type: "api-gateway", Config: map[string]any{"label": "Command API", "latencyMs": 5}},
			{ID: "query-api", Type: "api-gateway", Config: map[string]any{"label": "Query API", "latencyMs": 3}},
			{ID: "command-handler", Type: "service", Config: map[string]any{"label": "Command Handler", "latencyMs": 15, "replicas": 3}},
			{ID: "event-store", Type: "database", Config: map[string]any{"label": "Event Store", "engine": "postgresql", "latencyMs": 5}},
			{ID: "event-bus", Type: "queue", Config: map[string]any{"label": "Event Bus (Kafka)", "latencyMs": 3, "service": "MSK"}},
			{ID: "projection-worker", Type: "service", Config: map[string]any{"label": "Projection Worker", "latencyMs": 20, "replicas": 2}},
			{ID: "read-model-db", Type: "database", Config: map[string]any{"label": "Read Model DB", "engine": "postgresql", "latencyMs": 4}},
			{ID: "read-cache", Type: "cache", Config: map[string]any{"label": "Read Cache", "engine": "redis", "latencyMs": 1}},
			{ID: "query-service", Type: "service", Config: map[string]any{"label": "Query Service", "latencyMs": 8, "replicas": 4}},
			{ID: "saga-orchestrator", Type: "service", Config: map[string]any{"label": "Saga Orchestrator", "latencyMs": 10, "replicas": 2}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "command-api", Target: "command-handler"},
			{ID: "e2", Source: "command-handler", Target: "event-store"},
			{ID: "e3", Source: "command-handler", Target: "event-bus"},
			{ID: "e4", Source: "event-bus", Target: "projection-worker"},
			{ID: "e5", Source: "event-bus", Target: "saga-orchestrator"},
			{ID: "e6", Source: "projection-worker", Target: "read-model-db"},
			{ID: "e7", Source: "query-api", Target: "query-service"},
			{ID: "e8", Source: "query-service", Target: "read-cache"},
			{ID: "e9", Source: "query-service", Target: "read-model-db"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 120, RequestsPerSecond: 1000, Description: "High-throughput event stream with 80% reads, 20% writes"},
}

var ServerlessAPI = model.Template{
	ID:          "serverless-api",
	Name:        "Serverless REST API",
	Description: "Fully serverless API with Lambda/Cloud Functions, API Gateway, DynamoDB, and CDN for static assets",
	Category:    "serverless",
	Tags:        []string{"serverless", "lambda", "api-gateway", "dynamodb", "cdn"},
	Complexity:  "starter",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "cdn", Type: "cdn", Config: map[string]any{"label": "CloudFront CDN", "latencyMs": 2}},
			{ID: "api-gw", Type: "api-gateway", Config: map[string]any{"label": "API Gateway", "latencyMs": 10}},
			{ID: "auth-fn", Type: "service", Config: map[string]any{"label": "Auth Lambda", "latencyMs": 50, "runtime": "serverless", "coldStartMs": 200}},
			{ID: "crud-fn", Type: "service", Config: map[string]any{"label": "CRUD Lambda", "latencyMs": 30, "runtime": "serverless", "coldStartMs": 150}},
			{ID: "dynamo-db", Type: "database", Config: map[string]any{"label": "DynamoDB", "engine": "dynamodb", "latencyMs": 5}},
			{ID: "s3-assets", Type: "storage", Config: map[string]any{"label": "S3 Assets", "latencyMs": 10}},
			{ID: "cognito", Type: "service", Config: map[string]any{"label": "Cognito Auth", "latencyMs": 20}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "cdn", Target: "s3-assets"},
			{ID: "e2", Source: "cdn", Target: "api-gw"},
			{ID: "e3", Source: "api-gw", Target: "auth-fn"},
			{ID: "e4", Source: "api-gw", Target: "crud-fn"},
			{ID: "e5", Source: "auth-fn", Target: "cognito"},
			{ID: "e6", Source: "crud-fn", Target: "dynamo-db"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 30, RequestsPerSecond: 200, Description: "Bursty serverless traffic with cold starts"},
}

var ThreeTierWeb = model.Template{
	ID:          "three-tier-web",
	Name:        "Three-Tier Web Application",
	Description: "Classic web application with load balancer, application servers, and database with read replicas",
	Category:    "traditional",
	Tags:        []string{"web", "load-balancer", "read-replica", "traditional"},
	Complexity:  "starter",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "lb", Type: "load-balancer", Config: map[string]any{"label": "Load Balancer", "latencyMs": 2}},
			{ID: "web-1", Type: "service", Config: map[string]any{"label": "Web Server 1", "latencyMs": 20}},
			{ID: "web-2", Type: "service", Config: map[string]any{"label": "Web Server 2", "latencyMs": 20}},
			{ID: "web-3", Type: "service", Config: map[string]any{"label": "Web Server 3", "latencyMs": 20}},
			{ID: "primary-db", Type: "database", Config: map[string]any{"label": "Primary DB", "engine": "postgresql", "latencyMs": 8}},
			{ID: "read-replica", Type: "database", Config: map[string]any{"label": "Read Replica", "engine": "postgresql", "latencyMs": 6}},
			{ID: "session-store", Type: "cache", Config: map[string]any{"label": "Session Store (Redis)", "engine": "redis", "latencyMs": 1}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "lb", Target: "web-1"},
			{ID: "e2", Source: "lb", Target: "web-2"},
			{ID: "e3", Source: "lb", Target: "web-3"},
			{ID: "e4", Source: "web-1", Target: "primary-db"},
			{ID: "e5", Source: "web-2", Target: "read-replica"},
			{ID: "e6", Source: "web-3", Target: "read-replica"},
			{ID: "e7", Source: "web-1", Target: "session-store"},
			{ID: "e8", Source: "web-2", Target: "session-store"},
			{ID: "e9", Source: "web-3", Target: "session-store"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 60, RequestsPerSecond: 300, Description: "Steady web traffic evenly distributed across servers"},
}

var StreamProcessing = model.Template{
	ID:          "stream-processing",
	Name:        "Real-Time Stream Processing",
	Description: "High-throughput data pipeline with Kafka, stream processors, and time-series storage for analytics",
	Category:    "data",
	Tags:        []string{"kafka", "streaming", "analytics", "time-series"},
	Complexity:  "advanced",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "ingress-lb", Type: "load-balancer", Config: map[string]any{"label": "Ingress LB", "latencyMs": 2}},
			{ID: "collector-1", Type: "service", Config: map[string]any{"label": "Data Collector 1", "latencyMs": 5, "replicas": 3}},
			{ID: "collector-2", Type: "service", Config: map[string]any{"label": "Data Collector 2", "latencyMs": 5, "replicas": 3}},
			{ID: "kafka", Type: "queue", Config: map[string]any{"label": "Kafka Cluster", "latencyMs": 3, "partitions": 12}},
			{ID: "stream-processor", Type: "service", Config: map[string]any{"label": "Stream Processor (Flink)", "latencyMs": 15, "replicas": 4}},
			{ID: "enrichment-svc", Type: "service", Config: map[string]any{"label": "Enrichment Service", "latencyMs": 10, "replicas": 2}},
			{ID: "timeseries-db", Type: "database", Config: map[string]any{"label": "TimescaleDB", "engine": "postgresql", "latencyMs": 4}},
			{ID: "analytics-cache", Type: "cache", Config: map[string]any{"label": "Analytics Cache", "engine": "redis", "latencyMs": 1}},
			{ID: "dashboard-api", Type: "service", Config: map[string]any{"label": "Dashboard API", "latencyMs": 10, "replicas": 2}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "ingress-lb", Target: "collector-1"},
			{ID: "e2", Source: "ingress-lb", Target: "collector-2"},
			{ID: "e3", Source: "collector-1", Target: "kafka"},
			{ID: "e4", Source: "collector-2", Target: "kafka"},
			{ID: "e5", Source: "kafka", Target: "stream-processor"},
			{ID: "e6", Source: "stream-processor", Target: "enrichment-svc"},
			{ID: "e7", Source: "stream-processor", Target: "timeseries-db"},
			{ID: "e8", Source: "dashboard-api", Target: "timeseries-db"},
			{ID: "e9", Source: "dashboard-api", Target: "analytics-cache"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 120, RequestsPerSecond: 5000, Description: "High-volume event ingestion (5K events/sec)"},
}

var MultiRegionActive = model.Template{
	ID:          "multi-region-active-active",
	Name:        "Multi-Region Active-Active",
	Description: "Globally distributed active-active setup with regional load balancers, cross-region replication, and conflict resolution",
	Category:    "distributed",
	Tags:        []string{"multi-region", "active-active", "global", "replication"},
	Complexity:  "advanced",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "global-lb", Type: "load-balancer", Config: map[string]any{"label": "Global LB (Route53)", "latencyMs": 1}},
			{ID: "us-east-service", Type: "service", Config: map[string]any{"label": "US-East Service", "latencyMs": 15, "region": "us-east-1", "replicas": 3}},
			{ID: "eu-west-service", Type: "service", Config: map[string]any{"label": "EU-West Service", "latencyMs": 15, "region": "eu-west-1", "replicas": 3}},
			{ID: "us-east-db", Type: "database", Config: map[string]any{"label": "US-East Aurora", "engine": "aurora-postgresql", "region": "us-east-1", "latencyMs": 5}},
			{ID: "eu-west-db", Type: "database", Config: map[string]any{"label": "EU-West Aurora", "engine": "aurora-postgresql", "region": "eu-west-1", "latencyMs": 5}},
			{ID: "us-east-cache", Type: "cache", Config: map[string]any{"label": "US-East Redis", "region": "us-east-1", "latencyMs": 1}},
			{ID: "eu-west-cache", Type: "cache", Config: map[string]any{"label": "EU-West Redis", "region": "eu-west-1", "latencyMs": 1}},
			{ID: "replication-stream", Type: "queue", Config: map[string]any{"label": "Cross-Region Replication", "latencyMs": 75}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "global-lb", Target: "us-east-service"},
			{ID: "e2", Source: "global-lb", Target: "eu-west-service"},
			{ID: "e3", Source: "us-east-service", Target: "us-east-db"},
			{ID: "e4", Source: "us-east-service", Target: "us-east-cache"},
			{ID: "e5", Source: "eu-west-service", Target: "eu-west-db"},
			{ID: "e6", Source: "eu-west-service", Target: "eu-west-cache"},
			{ID: "e7", Source: "us-east-db", Target: "replication-stream"},
			{ID: "e8", Source: "replication-stream", Target: "eu-west-db"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 90, RequestsPerSecond: 2000, Description: "Global traffic split 60/40 across regions with cross-region replication lag"},
}

var ServiceMeshIstio = model.Template{
	ID:          "service-mesh-istio",
	Name:        "Service Mesh (Istio/Envoy)",
	Description: "Microservices with Istio service mesh - sidecar proxies, mTLS, circuit breakers, retries, and observability",
	Category:    "service-mesh",
	Tags:        []string{"istio", "envoy", "sidecar", "mtls", "circuit-breaker"},
	Complexity:  "advanced",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "istio-ingress", Type: "api-gateway", Config: map[string]any{"label": "Istio Ingress Gateway", "latencyMs": 3, "sidecar": true}},
			{ID: "frontend-svc", Type: "service", Config: map[string]any{"label": "Frontend Service", "latencyMs": 10, "sidecar": true, "sidecarLatencyMs": 1.5}},
			{ID: "product-svc", Type: "service", Config: map[string]any{"label": "Product Service", "latencyMs": 15, "sidecar": true, "sidecarLatencyMs": 1.5}},
			{ID: "inventory-svc", Type: "service", Config: map[string]any{"label": "Inventory Service", "latencyMs": 12, "sidecar": true, "sidecarLatencyMs": 1.5}},
			{ID: "pricing-svc", Type: "service", Config: map[string]any{"label": "Pricing Service", "latencyMs": 8, "sidecar": true, "sidecarLatencyMs": 1.5}},
			{ID: "product-db", Type: "database", Config: map[string]any{"label": "Product DB", "latencyMs": 5}},
			{ID: "inventory-db", Type: "database", Config: map[string]any{"label": "Inventory DB", "latencyMs": 5}},
			{ID: "pricing-cache", Type: "cache", Config: map[string]any{"label": "Pricing Cache", "latencyMs": 1}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "istio-ingress", Target: "frontend-svc"},
			{ID: "e2", Source: "frontend-svc", Target: "product-svc"},
			{ID: "e3", Source: "frontend-svc", Target: "inventory-svc"},
			{ID: "e4", Source: "product-svc", Target: "pricing-svc"},
			{ID: "e5", Source: "product-svc", Target: "product-db"},
			{ID: "e6", Source: "inventory-svc", Target: "inventory-db"},
			{ID: "e7", Source: "pricing-svc", Target: "pricing-cache"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 60, RequestsPerSecond: 800, Description: "Service mesh traffic with mTLS overhead and sidecar latency"},
}

var MLInferencePipeline = model.Template{
	ID:          "ml-inference-pipeline",
	Name:        "ML Model Inference Pipeline",
	Description: "Real-time ML serving with model gateway, feature store, A/B testing, and model versioning",
	Category:    "ml",
	Tags:        []string{"ml", "inference", "feature-store", "ab-testing", "gpu"},
	Complexity:  "intermediate",
	Graph: model.Graph{
		Nodes: []model.Node{
			{ID: "api-lb", Type: "load-balancer", Config: map[string]any{"label": "API Load Balancer", "latencyMs": 2}},
			{ID: "model-gateway", Type: "service", Config: map[string]any{"label": "Model Gateway", "latencyMs": 5, "replicas": 2}},
			{ID: "feature-store", Type: "cache", Config: map[string]any{"label": "Feature Store (Redis)", "engine": "redis", "latencyMs": 2}},
			{ID: "model-a", Type: "service", Config: map[string]any{"label": "Model A (Production)", "latencyMs": 45, "replicas": 4, "gpu": true}},
			{ID: "model-b", Type: "service", Config: map[string]any{"label": "Model B (Canary)", "latencyMs": 55, "replicas": 1, "gpu": true}},
			{ID: "prediction-log", Type: "queue", Config: map[string]any{"label": "Prediction Log (Kafka)", "latencyMs": 3}},
			{ID: "metrics-db", Type: "database", Config: map[string]any{"label": "Metrics DB", "engine": "postgresql", "latencyMs": 6}},
			{ID: "model-registry", Type: "storage", Config: map[string]any{"label": "Model Registry (S3)", "latencyMs": 20}},
		},
		Edges: []model.Edge{
			{ID: "e1", Source: "api-lb", Target: "model-gateway"},
			{ID: "e2", Source: "model-gateway", Target: "feature-store"},
			{ID: "e3", Source: "model-gateway", Target: "model-a"},
			{ID: "e4", Source: "model-gateway", Target: "model-b"},
			{ID: "e5", Source: "model-a", Target: "prediction-log"},
			{ID: "e6", Source: "model-b", Target: "prediction-log"},
			{ID: "e7", Source: "prediction-log", Target: "metrics-db"},
			{ID: "e8", Source: "model-gateway", Target: "model-registry"},
		},
	},
	TrafficPreset: model.TrafficPreset{DurationSeconds: 60, RequestsPerSecond: 1000, Description: "ML inference load with 95/5 A/B traffic split"},
}

// Categories returns all template categories
func Categories() []model.Category {
	catMap := map[string]int{}
	for _, t := range All {
		catMap[t.Category]++
	}

	cats := []model.Category{
		{ID: "microservices", Name: "Microservices", Description: "Distributed service architectures"},
		{ID: "event-driven", Name: "Event-Driven", Description: "Event sourcing and CQRS patterns"},
		{ID: "serverless", Name: "Serverless", Description: "FaaS and managed service architectures"},
		{ID: "traditional", Name: "Traditional", Description: "Classic N-tier architectures"},
		{ID: "data", Name: "Data Pipeline", Description: "Stream processing and analytics"},
		{ID: "distributed", Name: "Distributed", Description: "Multi-region and globally distributed"},
		{ID: "service-mesh", Name: "Service Mesh", Description: "Istio, Linkerd, and sidecar patterns"},
		{ID: "ml", Name: "Machine Learning", Description: "ML serving and inference pipelines"},
	}

	for i := range cats {
		cats[i].Count = catMap[cats[i].ID]
	}
	return cats
}
