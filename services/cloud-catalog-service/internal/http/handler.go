package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"moktopus/cloud-catalog-service/internal/catalog"
	"moktopus/cloud-catalog-service/internal/model"
)

type handler struct{}

func NewHandler() http.Handler {
	h := &handler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/v1/catalog/providers", h.providers)
	mux.HandleFunc("/api/v1/catalog/regions", h.regions)
	mux.HandleFunc("/api/v1/catalog/compute", h.compute)
	mux.HandleFunc("/api/v1/catalog/cache", h.cache)
	mux.HandleFunc("/api/v1/catalog/database", h.database)
	mux.HandleFunc("/api/v1/catalog/queue", h.queue)
	mux.HandleFunc("/api/v1/catalog/loadbalancer", h.loadbalancer)
	mux.HandleFunc("/api/v1/catalog/latency", h.latency)
	mux.HandleFunc("/api/v1/catalog/search", h.search)
	mux.HandleFunc("/api/v1/catalog/recommend", h.recommend)
	return mux
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "cloud-catalog-service", "status": "ok"})
}

func (h *handler) providers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	providers := []model.Provider{
		{ID: "aws", Name: "Amazon Web Services", Description: "AWS cloud infrastructure with 30+ regions worldwide"},
		{ID: "gcp", Name: "Google Cloud Platform", Description: "GCP with 37+ regions and integrated AI/ML services"},
		{ID: "azure", Name: "Microsoft Azure", Description: "Azure with 60+ regions and enterprise integration"},
	}
	respondData(w, http.StatusOK, providers)
}

func (h *handler) regions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	provider := r.URL.Query().Get("provider")
	if provider != "" {
		regions := catalog.RegionsByProvider(provider)
		if regions == nil {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "unknown provider: "+provider)
			return
		}
		respondData(w, http.StatusOK, regions)
		return
	}
	respondData(w, http.StatusOK, catalog.AllRegions())
}

func (h *handler) compute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	provider := r.URL.Query().Get("provider")
	family := r.URL.Query().Get("family")

	var instances []model.InstanceType
	if provider != "" {
		instances = catalog.ComputeByProvider(provider)
		if instances == nil {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "unknown provider")
			return
		}
	} else {
		instances = catalog.AllComputeInstances()
	}

	if family != "" {
		filtered := make([]model.InstanceType, 0)
		for _, inst := range instances {
			if strings.EqualFold(inst.Family, family) {
				filtered = append(filtered, inst)
			}
		}
		instances = filtered
	}

	respondData(w, http.StatusOK, model.CatalogResponse{
		Provider: provider,
		Service:  "compute",
		Count:    len(instances),
		Items:    instances,
	})
}

func (h *handler) cache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	provider := r.URL.Query().Get("provider")

	var configs []model.CacheConfig
	if provider != "" {
		configs = catalog.CacheByProvider(provider)
		if configs == nil {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "unknown provider")
			return
		}
	} else {
		configs = catalog.AllCacheInstances()
	}

	respondData(w, http.StatusOK, model.CatalogResponse{
		Provider: provider,
		Service:  "cache",
		Count:    len(configs),
		Items:    configs,
	})
}

func (h *handler) database(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	provider := r.URL.Query().Get("provider")
	engine := r.URL.Query().Get("engine")

	var configs []model.DatabaseConfig
	if provider != "" {
		configs = catalog.DatabaseByProvider(provider)
		if configs == nil {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "unknown provider")
			return
		}
	} else {
		configs = catalog.AllDatabaseInstances()
	}

	if engine != "" {
		filtered := make([]model.DatabaseConfig, 0)
		for _, db := range configs {
			if strings.EqualFold(db.Engine, engine) {
				filtered = append(filtered, db)
			}
		}
		configs = filtered
	}

	respondData(w, http.StatusOK, model.CatalogResponse{
		Provider: provider,
		Service:  "database",
		Count:    len(configs),
		Items:    configs,
	})
}

func (h *handler) queue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	provider := r.URL.Query().Get("provider")

	var configs []model.QueueConfig
	if provider != "" {
		configs = catalog.QueueByProvider(provider)
		if configs == nil {
			respondError(w, http.StatusBadRequest, "INVALID_PROVIDER", "unknown provider")
			return
		}
	} else {
		configs = catalog.AllQueueServices()
	}

	respondData(w, http.StatusOK, model.CatalogResponse{
		Provider: provider,
		Service:  "queue",
		Count:    len(configs),
		Items:    configs,
	})
}

func (h *handler) loadbalancer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	respondData(w, http.StatusOK, model.CatalogResponse{
		Service: "loadbalancer",
		Count:   len(catalog.AllLoadBalancers()),
		Items:   catalog.AllLoadBalancers(),
	})
}

func (h *handler) latency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		respondError(w, http.StatusBadRequest, "MISSING_PARAMS", "both 'from' and 'to' region params required")
		return
	}
	latency := catalog.GetInterRegionLatency(from, to)
	respondData(w, http.StatusOK, map[string]interface{}{
		"from":      from,
		"to":        to,
		"latencyMs": latency,
	})
}

func (h *handler) search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q == "" {
		respondError(w, http.StatusBadRequest, "MISSING_QUERY", "'q' parameter required")
		return
	}

	results := make(map[string]interface{})

	// Search compute
	var matchedCompute []model.InstanceType
	for _, inst := range catalog.AllComputeInstances() {
		if strings.Contains(strings.ToLower(inst.Name), q) || strings.Contains(strings.ToLower(inst.Service), q) || strings.Contains(strings.ToLower(inst.Family), q) {
			matchedCompute = append(matchedCompute, inst)
		}
	}
	if len(matchedCompute) > 0 {
		results["compute"] = matchedCompute
	}

	// Search cache
	var matchedCache []model.CacheConfig
	for _, c := range catalog.AllCacheInstances() {
		if strings.Contains(strings.ToLower(c.NodeType), q) || strings.Contains(strings.ToLower(c.Service), q) {
			matchedCache = append(matchedCache, c)
		}
	}
	if len(matchedCache) > 0 {
		results["cache"] = matchedCache
	}

	// Search database
	var matchedDB []model.DatabaseConfig
	for _, db := range catalog.AllDatabaseInstances() {
		if strings.Contains(strings.ToLower(db.InstanceClass), q) || strings.Contains(strings.ToLower(db.Service), q) || strings.Contains(strings.ToLower(db.Engine), q) {
			matchedDB = append(matchedDB, db)
		}
	}
	if len(matchedDB) > 0 {
		results["database"] = matchedDB
	}

	respondData(w, http.StatusOK, results)
}

func respondJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type apiEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondData(w http.ResponseWriter, code int, data interface{}) {
	respondJSON(w, code, apiEnvelope{Success: true, Data: data})
}

func respondError(w http.ResponseWriter, code int, errCode, message string) {
	respondJSON(w, code, apiEnvelope{Success: false, Error: &apiError{Code: errCode, Message: message}})
}

// ═══ RECOMMENDATION ENGINE ═══

type recommendRequest struct {
	Provider string           `json:"provider"`
	Nodes    []recommendNode  `json:"nodes"`
}

type recommendNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

type recommendation struct {
	NodeID        string   `json:"nodeId"`
	NodeType      string   `json:"nodeType"`
	Suggested     string   `json:"suggestedInstance"`
	Service       string   `json:"service"`
	Reasoning     string   `json:"reasoning"`
	Alternatives  []string `json:"alternatives,omitempty"`
	CostTier      string   `json:"costTier"` // low, medium, high
	Tips          []string `json:"tips,omitempty"`
}

func (h *handler) recommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req recommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid JSON payload")
		return
	}

	if req.Provider == "" {
		req.Provider = "aws"
	}

	recommendations := make([]recommendation, 0, len(req.Nodes))
	for _, node := range req.Nodes {
		rec := recommendForNode(node, req.Provider)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	respondData(w, http.StatusOK, map[string]interface{}{
		"provider":        req.Provider,
		"recommendations": recommendations,
		"totalNodes":      len(req.Nodes),
	})
}

func recommendForNode(node recommendNode, provider string) *recommendation {
	switch node.Type {
	case "service", "worker":
		return recommendCompute(node, provider)
	case "database":
		return recommendDatabase(node, provider)
	case "cache":
		return recommendCache(node, provider)
	case "queue":
		return recommendQueue(node, provider)
	case "load-balancer":
		return recommendLB(node, provider)
	case "serverless":
		return &recommendation{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Suggested: "Lambda / Cloud Functions",
			Service:   "serverless",
			Reasoning: "Serverless is ideal for event-driven workloads with variable traffic",
			CostTier:  "low",
			Tips: []string{
				"Set memory between 128-512MB for most API handlers",
				"Use provisioned concurrency for latency-sensitive functions",
				"Enable X-Ray tracing for observability",
			},
		}
	case "stream":
		return recommendStream(node, provider)
	case "monitoring":
		return &recommendation{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Suggested: selectByProvider(provider, "CloudWatch + X-Ray", "Cloud Monitoring + Cloud Trace", "Azure Monitor + Application Insights"),
			Service:   "monitoring",
			Reasoning: "Full-stack observability for metrics, logs, and traces",
			CostTier:  "medium",
			Tips: []string{
				"Set log retention to 30 days to balance cost and debuggability",
				"Use custom metrics for business KPIs",
				"Set up alerts for p99 latency and error rates",
			},
		}
	case "cdn":
		return &recommendation{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Suggested: selectByProvider(provider, "CloudFront", "Cloud CDN", "Azure CDN"),
			Service:   "cdn",
			Reasoning: "CDN reduces latency for static assets by serving from edge locations",
			CostTier:  "low",
			Tips: []string{
				"Enable compression (gzip/brotli) for text assets",
				"Set cache TTL based on content update frequency",
				"Use origin shield to reduce origin load",
			},
		}
	default:
		return nil
	}
}

func recommendCompute(node recommendNode, provider string) *recommendation {
	// Check if config has instance type already
	existing := getStr(node.Config, "instanceType")
	if existing == "" {
		existing = getStr(node.Config, "instanceClass")
	}

	var suggested, reasoning, costTier string
	var alternatives []string
	var tips []string

	switch provider {
	case "aws":
		suggested = "t3.medium"
		alternatives = []string{"t3.large", "m6i.large", "c6i.large"}
		reasoning = "t3.medium provides burstable compute suitable for most microservices. Upgrade to m6i for sustained workloads."
		costTier = "low"
		tips = []string{
			"Use Graviton (t4g/m7g) for 20% better price-performance",
			"Enable auto-scaling based on CPU/request count",
			"Consider Spot instances for stateless workers",
		}
		if existing != "" {
			suggested = existing
			reasoning = "Keeping existing instance type " + existing
		}
	case "gcp":
		suggested = "e2-medium"
		alternatives = []string{"e2-standard-2", "n2-standard-2", "c2-standard-4"}
		reasoning = "e2-medium offers cost-efficient compute for general workloads"
		costTier = "low"
		tips = []string{
			"Use preemptible VMs for batch processing",
			"Enable autoscaler with CPU target of 60%",
		}
	case "azure":
		suggested = "Standard_B2s"
		alternatives = []string{"Standard_D2s_v5", "Standard_D4s_v5"}
		reasoning = "B-series provides burstable compute, upgrade to D-series for consistent workloads"
		costTier = "low"
		tips = []string{
			"Use Azure Spot VMs for interruptible workloads",
			"Enable autoscale rules on VMSS",
		}
	}

	return &recommendation{
		NodeID:       node.ID,
		NodeType:     node.Type,
		Suggested:    suggested,
		Service:      "compute",
		Reasoning:    reasoning,
		Alternatives: alternatives,
		CostTier:     costTier,
		Tips:         tips,
	}
}

func recommendDatabase(node recommendNode, provider string) *recommendation {
	engine := getStr(node.Config, "engine")
	if engine == "" {
		engine = "postgres"
	}

	var suggested, reasoning, costTier string
	var alternatives []string
	var tips []string

	switch provider {
	case "aws":
		suggested = "db.t4g.medium"
		alternatives = []string{"db.t4g.large", "db.r6g.large", "db.r6g.xlarge"}
		reasoning = "Graviton-based t4g provides best price-performance for typical DB workloads"
		costTier = "medium"
		tips = []string{
			"Enable Multi-AZ for production workloads",
			"Use RDS Proxy for connection pooling with serverless",
			"Enable Performance Insights for query analysis",
			"Consider Aurora Serverless v2 for variable workloads",
		}
	case "gcp":
		suggested = "db-custom-2-8192"
		alternatives = []string{"db-custom-4-16384", "db-custom-8-32768"}
		reasoning = "Custom machine type lets you right-size CPU and memory independently"
		costTier = "medium"
		tips = []string{
			"Enable high availability for production",
			"Use Cloud SQL Insights for query optimization",
		}
	case "azure":
		suggested = "GP_Standard_D2s_v3"
		alternatives = []string{"GP_Standard_D4s_v3", "BC_Standard_D4s_v3"}
		reasoning = "General Purpose tier balances compute and memory for most workloads"
		costTier = "medium"
		tips = []string{
			"Enable geo-redundant backup for DR",
			"Use connection pooling with PgBouncer (built-in)",
		}
	}

	return &recommendation{
		NodeID:       node.ID,
		NodeType:     node.Type,
		Suggested:    suggested + " (" + engine + ")",
		Service:      "database",
		Reasoning:    reasoning,
		Alternatives: alternatives,
		CostTier:     costTier,
		Tips:         tips,
	}
}

func recommendCache(node recommendNode, provider string) *recommendation {
	var suggested, reasoning, costTier string
	var alternatives []string
	var tips []string

	switch provider {
	case "aws":
		suggested = "cache.t4g.medium"
		alternatives = []string{"cache.t4g.small", "cache.r6g.large", "cache.r7g.large"}
		reasoning = "t4g.medium provides 3.09 GB memory suitable for most caching needs"
		costTier = "low"
		tips = []string{
			"Use Redis cluster mode for horizontal scaling",
			"Enable automatic failover with 2+ replicas",
			"Set maxmemory-policy to allkeys-lru for cache use cases",
			"Consider ElastiCache Serverless for unpredictable workloads",
		}
	case "gcp":
		suggested = "STANDARD_HA (4GB)"
		alternatives = []string{"BASIC (1GB)", "STANDARD_HA (8GB)"}
		reasoning = "Standard HA tier provides automatic failover"
		costTier = "low"
		tips = []string{"Enable AUTH for production", "Use Memorystore for Redis for sub-ms latency"}
	case "azure":
		suggested = "Standard C2"
		alternatives = []string{"Standard C1", "Premium P1"}
		reasoning = "Standard tier with 2.5GB provides replication and SLA"
		costTier = "low"
		tips = []string{"Use Premium tier for persistence and geo-replication", "Enable non-SSL port only in VNet"}
	}

	return &recommendation{
		NodeID:       node.ID,
		NodeType:     node.Type,
		Suggested:    suggested,
		Service:      "cache",
		Reasoning:    reasoning,
		Alternatives: alternatives,
		CostTier:     costTier,
		Tips:         tips,
	}
}

func recommendQueue(node recommendNode, provider string) *recommendation {
	var suggested, reasoning, costTier string
	var tips []string

	switch provider {
	case "aws":
		suggested = "SQS Standard"
		reasoning = "SQS Standard provides nearly unlimited throughput with at-least-once delivery"
		costTier = "low"
		tips = []string{
			"Use FIFO queues only when ordering is required (lower throughput)",
			"Set visibility timeout to 6x your processing time",
			"Enable dead-letter queue for failed messages",
			"Use long polling (20s) to reduce empty receives",
		}
	case "gcp":
		suggested = "Cloud Pub/Sub"
		reasoning = "Pub/Sub provides global messaging with exactly-once delivery"
		costTier = "low"
		tips = []string{"Use ordering keys only when needed", "Set ack deadline based on processing time"}
	case "azure":
		suggested = "Service Bus Standard"
		reasoning = "Service Bus provides enterprise messaging with topics and subscriptions"
		costTier = "low"
		tips = []string{"Use sessions for FIFO guarantees", "Enable dead-lettering for poison messages"}
	}

	return &recommendation{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Suggested: suggested,
		Service:   "queue",
		Reasoning: reasoning,
		CostTier:  costTier,
		Tips:      tips,
	}
}

func recommendLB(node recommendNode, provider string) *recommendation {
	var suggested, reasoning, costTier string
	var tips []string

	switch provider {
	case "aws":
		suggested = "Application Load Balancer"
		reasoning = "ALB provides L7 routing with path/host-based rules, WebSocket support, and gRPC"
		costTier = "low"
		tips = []string{
			"Use target group health checks with short intervals",
			"Enable access logs for debugging",
			"Use NLB for TCP/UDP or extreme performance needs",
		}
	case "gcp":
		suggested = "Global HTTP(S) Load Balancer"
		reasoning = "Global LB provides multi-region anycast with automatic failover"
		costTier = "low"
		tips = []string{"Use Cloud Armor for DDoS protection", "Enable CDN on backend services"}
	case "azure":
		suggested = "Application Gateway v2"
		reasoning = "App Gateway provides L7 routing with WAF integration"
		costTier = "medium"
		tips = []string{"Enable autoscaling to handle traffic spikes", "Use Azure Front Door for global routing"}
	}

	return &recommendation{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Suggested: suggested,
		Service:   "load-balancer",
		Reasoning: reasoning,
		CostTier:  costTier,
		Tips:      tips,
	}
}

func recommendStream(node recommendNode, provider string) *recommendation {
	var suggested, reasoning, costTier string
	var tips []string

	switch provider {
	case "aws":
		suggested = "Kinesis Data Streams (On-Demand)"
		reasoning = "On-demand mode auto-scales shards without capacity planning"
		costTier = "medium"
		tips = []string{
			"Use enhanced fan-out for multiple consumers",
			"Consider MSK (Managed Kafka) for complex event processing",
			"Set retention to 24h unless replay is needed",
		}
	case "gcp":
		suggested = "Cloud Pub/Sub"
		reasoning = "Pub/Sub handles streaming with automatic scaling"
		costTier = "low"
		tips = []string{"Use BigQuery subscriptions for analytics", "Enable message ordering when needed"}
	case "azure":
		suggested = "Event Hubs Standard"
		reasoning = "Event Hubs provides Kafka-compatible streaming at scale"
		costTier = "medium"
		tips = []string{"Use capture for long-term storage", "Consider Premium for dedicated throughput"}
	}

	return &recommendation{
		NodeID:    node.ID,
		NodeType:  node.Type,
		Suggested: suggested,
		Service:   "stream",
		Reasoning: reasoning,
		CostTier:  costTier,
		Tips:      tips,
	}
}

func selectByProvider(provider, aws, gcp, azure string) string {
	switch provider {
	case "aws":
		return aws
	case "gcp":
		return gcp
	case "azure":
		return azure
	}
	return aws
}

func getStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
