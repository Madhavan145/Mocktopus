package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"moktopus/topology-service/internal/model"
	"moktopus/topology-service/internal/store"
)

type handler struct {
	store store.Store
}

type saveTopologyRequest struct {
	Graph model.Graph `json:"graph"`
}

func NewHandler(repo store.Store) http.Handler {
	h := handler{store: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/topology/validate", h.validate)
	mux.HandleFunc("/api/v1/topology/validate", h.validateV1)
	mux.HandleFunc("/api/topology/ctm", h.ctm)
	mux.HandleFunc("/api/v1/topology/ctm", h.ctmV1)
	mux.HandleFunc("/api/topology/projects/", h.projectTopology)
	mux.HandleFunc("/api/v1/topology/projects/", h.projectTopologyV1)
	return mux
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "topology-service", "status": "ok"})
}

func (h handler) validate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	graph, ok := decodeGraph(w, r)
	if !ok {
		return
	}

	result := validateGraph(graph)
	respondJSON(w, http.StatusOK, result)
}

func (h handler) validateV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	graph, ok := decodeGraphV1(w, r)
	if !ok {
		return
	}

	result := validateGraph(graph)
	respondV1Data(w, http.StatusOK, result)
}

func (h handler) ctm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	graph, ok := decodeGraph(w, r)
	if !ok {
		return
	}

	result := validateGraph(graph)
	if !result.Valid {
		respondJSON(w, http.StatusBadRequest, result)
		return
	}

	respondJSON(w, http.StatusOK, canonicalize(graph))
}

func (h handler) ctmV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	graph, ok := decodeGraphV1(w, r)
	if !ok {
		return
	}

	result := validateGraph(graph)
	if !result.Valid {
		respondV1Error(w, http.StatusBadRequest, "INVALID_GRAPH", strings.Join(result.Errors, "; "))
		return
	}

	respondV1Data(w, http.StatusOK, canonicalize(graph))
}

func (h handler) projectTopology(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimPrefix(r.URL.Path, "/api/topology/projects/")
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || strings.Contains(projectID, "/") {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		topology, ok, err := h.store.Get(projectID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !ok {
			if projectID == "demo" {
				seeded := defaultTopology(projectID)
				_ = h.store.Save(seeded)
				respondJSON(w, http.StatusOK, seeded)
				return
			}
			http.Error(w, "topology not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, topology)
	case http.MethodPut:
		var req saveTopologyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		result := validateGraph(req.Graph)
		if !result.Valid {
			respondJSON(w, http.StatusBadRequest, result)
			return
		}

		saved := model.SavedTopology{
			ProjectID: projectID,
			Graph:     req.Graph,
			CTM:       canonicalize(req.Graph),
			UpdatedAt: time.Now().UTC(),
		}
		if err := h.store.Save(saved); err != nil {
			http.Error(w, "failed to save topology", http.StatusInternalServerError)
			return
		}
		respondJSON(w, http.StatusOK, saved)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h handler) projectTopologyV1(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimPrefix(r.URL.Path, "/api/v1/topology/projects/")
	projectID = strings.TrimSpace(projectID)
	if projectID == "" || strings.Contains(projectID, "/") {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "invalid project id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		topology, ok, err := h.store.Get(projectID)
		if err != nil {
			respondV1Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
			return
		}
		if !ok {
			if projectID == "demo" {
				seeded := defaultTopology(projectID)
				_ = h.store.Save(seeded)
				respondV1Data(w, http.StatusOK, seeded)
				return
			}
			respondV1Error(w, http.StatusNotFound, "TOPOLOGY_NOT_FOUND", "topology not found")
			return
		}
		respondV1Data(w, http.StatusOK, topology)
	case http.MethodPut:
		var req saveTopologyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
			return
		}
		result := validateGraph(req.Graph)
		if !result.Valid {
			respondV1Error(w, http.StatusBadRequest, "INVALID_GRAPH", strings.Join(result.Errors, "; "))
			return
		}

		saved := model.SavedTopology{
			ProjectID: projectID,
			Graph:     req.Graph,
			CTM:       canonicalize(req.Graph),
			UpdatedAt: time.Now().UTC(),
		}
		if err := h.store.Save(saved); err != nil {
			respondV1Error(w, http.StatusInternalServerError, "SAVE_FAILED", "failed to save topology")
			return
		}
		respondV1Data(w, http.StatusOK, saved)
	default:
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func decodeGraph(w http.ResponseWriter, r *http.Request) (model.Graph, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read payload", http.StatusBadRequest)
		return model.Graph{}, false
	}
	if len(body) == 0 {
		http.Error(w, "payload required", http.StatusBadRequest)
		return model.Graph{}, false
	}

	var graph model.Graph
	if err := json.Unmarshal(body, &graph); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return model.Graph{}, false
	}
	return graph, true
}

func decodeGraphV1(w http.ResponseWriter, r *http.Request) (model.Graph, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondV1Error(w, http.StatusBadRequest, "PAYLOAD_READ_FAILED", "unable to read payload")
		return model.Graph{}, false
	}
	if len(body) == 0 {
		respondV1Error(w, http.StatusBadRequest, "PAYLOAD_REQUIRED", "payload required")
		return model.Graph{}, false
	}

	var graph model.Graph
	if err := json.Unmarshal(body, &graph); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return model.Graph{}, false
	}
	return graph, true
}

func validateGraph(graph model.Graph) model.ValidationResult {
	errors := make([]string, 0)
	warnings := make([]string, 0)

	if len(graph.Nodes) == 0 {
		errors = append(errors, "at least one node is required")
	}

	nodeIDs := map[string]struct{}{}
	nodeTypes := map[string]string{} // id -> type
	for _, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		nType := strings.TrimSpace(node.Type)
		if id == "" {
			errors = append(errors, "node id is required")
			continue
		}
		if nType == "" {
			errors = append(errors, "node type is required for node "+id)
		}
		if _, exists := nodeIDs[id]; exists {
			errors = append(errors, "duplicate node id: "+id)
			continue
		}
		nodeIDs[id] = struct{}{}
		nodeTypes[id] = nType
	}

	edgeIDs := map[string]struct{}{}
	outgoing := map[string][]string{} // source -> []target
	incoming := map[string][]string{} // target -> []source
	for _, edge := range graph.Edges {
		if strings.TrimSpace(edge.ID) == "" {
			errors = append(errors, "edge id is required")
			continue
		}
		if _, exists := edgeIDs[edge.ID]; exists {
			errors = append(errors, "duplicate edge id: "+edge.ID)
			continue
		}
		edgeIDs[edge.ID] = struct{}{}

		if _, ok := nodeIDs[edge.Source]; !ok {
			errors = append(errors, "edge source not found: "+edge.Source)
		}
		if _, ok := nodeIDs[edge.Target]; !ok {
			errors = append(errors, "edge target not found: "+edge.Target)
		}
		if edge.Source == edge.Target {
			errors = append(errors, "self loop edge is not allowed: "+edge.ID)
		}
		outgoing[edge.Source] = append(outgoing[edge.Source], edge.Target)
		incoming[edge.Target] = append(incoming[edge.Target], edge.Source)
	}

	// ═══ ARCHITECTURAL VALIDATION (warnings, not errors) ═══

	// Orphan nodes: no edges at all
	for _, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		if len(outgoing[id]) == 0 && len(incoming[id]) == 0 && len(graph.Nodes) > 1 {
			warnings = append(warnings, "orphan node with no connections: "+id)
		}
	}

	// Public-facing database anti-pattern: database with no incoming from a service/LB
	for _, node := range graph.Nodes {
		if node.Type != "database" {
			continue
		}
		id := strings.TrimSpace(node.ID)
		hasServiceIncoming := false
		for _, src := range incoming[id] {
			srcType := nodeTypes[src]
			if srcType == "service" || srcType == "serverless" || srcType == "load-balancer" || srcType == "queue" {
				hasServiceIncoming = true
				break
			}
		}
		// If database has no incoming from service-level nodes, but has incoming from entry-points
		for _, src := range incoming[id] {
			srcType := nodeTypes[src]
			if (srcType == "api-gateway" || srcType == "cdn") && !hasServiceIncoming {
				warnings = append(warnings, "database '"+id+"' appears directly exposed to entry-point '"+src+"' without a service layer")
			}
		}
	}

	// Missing load-balancer for replicated services
	for _, node := range graph.Nodes {
		if node.Type != "service" && node.Type != "worker" {
			continue
		}
		replicas := 0
		if v, ok := node.Config["replicas"]; ok {
			switch val := v.(type) {
			case float64:
				replicas = int(val)
			case int:
				replicas = val
			}
		}
		if v, ok := node.Config["desiredCount"]; ok {
			switch val := v.(type) {
			case string:
				if val != "1" && val != "0" && val != "" {
					replicas = 2
				}
			case float64:
				replicas = int(val)
			}
		}
		if replicas > 1 {
			id := strings.TrimSpace(node.ID)
			hasLB := false
			for _, src := range incoming[id] {
				if nodeTypes[src] == "load-balancer" {
					hasLB = true
					break
				}
			}
			if !hasLB {
				warnings = append(warnings, "service '"+id+"' has multiple replicas but no load-balancer routing to it")
			}
		}
	}

	// Single point of failure: critical infrastructure with no redundancy indication
	criticalTypes := map[string]bool{"database": true, "cache": true, "queue": true}
	for _, node := range graph.Nodes {
		if !criticalTypes[node.Type] {
			continue
		}
		id := strings.TrimSpace(node.ID)
		// Check if multi-AZ or replication is configured
		hasRedundancy := false
		if v, ok := node.Config["multiAz"]; ok && v == true {
			hasRedundancy = true
		}
		if v, ok := node.Config["replication"]; ok && v == true {
			hasRedundancy = true
		}
		if v, ok := node.Config["numClusters"]; ok {
			if s, ok := v.(string); ok && s != "1" && s != "0" {
				hasRedundancy = true
			}
			if f, ok := v.(float64); ok && f > 1 {
				hasRedundancy = true
			}
		}
		if !hasRedundancy && len(incoming[id]) > 0 {
			warnings = append(warnings, node.Type+" '"+id+"' may be a single point of failure (no redundancy configured)")
		}
	}

	// Detect cycles (simple DFS-based cycle detection)
	visited := map[string]int{} // 0=unvisited, 1=in-progress, 2=done
	var hasCycle bool
	var cycleNode string
	var dfs func(node string)
	dfs = func(node string) {
		if hasCycle {
			return
		}
		visited[node] = 1
		for _, target := range outgoing[node] {
			if visited[target] == 1 {
				hasCycle = true
				cycleNode = target
				return
			}
			if visited[target] == 0 {
				dfs(target)
			}
		}
		visited[node] = 2
	}
	for id := range nodeIDs {
		if visited[id] == 0 {
			dfs(id)
		}
	}
	if hasCycle {
		warnings = append(warnings, "cycle detected in topology graph involving node '"+cycleNode+"' - this may indicate circular dependencies")
	}

	// Unreachable nodes: nodes that can't be reached from any entry point
	entryPoints := []string{}
	for id := range nodeIDs {
		if len(incoming[id]) == 0 && len(outgoing[id]) > 0 {
			entryPoints = append(entryPoints, id)
		}
	}
	if len(entryPoints) > 0 && len(graph.Nodes) > 2 {
		reachable := map[string]bool{}
		queue := make([]string, 0, len(entryPoints))
		queue = append(queue, entryPoints...)
		for _, ep := range entryPoints {
			reachable[ep] = true
		}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, target := range outgoing[current] {
				if !reachable[target] {
					reachable[target] = true
					queue = append(queue, target)
				}
			}
		}
		for id := range nodeIDs {
			if !reachable[id] && len(incoming[id]) == 0 && len(outgoing[id]) == 0 {
				continue // already reported as orphan
			}
			if !reachable[id] && (len(incoming[id]) > 0 || len(outgoing[id]) > 0) {
				warnings = append(warnings, "node '"+id+"' is not reachable from any entry point")
			}
		}
	}

	// Services without monitoring
	hasMonitoring := false
	for _, node := range graph.Nodes {
		if node.Type == "monitoring" {
			hasMonitoring = true
			break
		}
	}
	serviceCount := 0
	for _, node := range graph.Nodes {
		if node.Type == "service" || node.Type == "serverless" {
			serviceCount++
		}
	}
	if serviceCount > 0 && !hasMonitoring {
		warnings = append(warnings, "topology has services but no monitoring/observability component")
	}

	return model.ValidationResult{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

func canonicalize(graph model.Graph) model.Graph {
	nodes := make([]model.Node, 0, len(graph.Nodes))
	edges := make([]model.Edge, 0, len(graph.Edges))

	for _, node := range graph.Nodes {
		nodes = append(nodes, model.Node{
			ID:     strings.TrimSpace(node.ID),
			Type:   strings.TrimSpace(node.Type),
			Config: node.Config,
		})
	}

	for _, edge := range graph.Edges {
		edges = append(edges, model.Edge{
			ID:     strings.TrimSpace(edge.ID),
			Source: strings.TrimSpace(edge.Source),
			Target: strings.TrimSpace(edge.Target),
		})
	}

	return model.Graph{Nodes: nodes, Edges: edges}
}

func defaultTopology(projectID string) model.SavedTopology {
	graph := model.Graph{
		Nodes: []model.Node{
			{ID: "gateway-1", Type: "api-gateway", Config: map[string]any{"cpuLimit": 1.0, "replicas": 1.0, "processingTimeMs": 8.0, "failureRate": 0.001, "autoscaling": true}},
			{ID: "lb-1", Type: "load-balancer", Config: map[string]any{"algorithm": "least-connections"}},
			{ID: "service-1", Type: "service", Config: map[string]any{"cpuLimit": 2.0, "replicas": 2.0, "processingTimeMs": 30.0, "failureRate": 0.002, "autoscaling": true}},
			{ID: "cache-1", Type: "cache", Config: map[string]any{"latencyMs": 3.0, "failureChance": 0.003}},
			{ID: "queue-1", Type: "queue", Config: map[string]any{"maxDepth": 1000.0, "processingRate": 220.0, "retryPolicy": "exponential"}},
			{ID: "database-1", Type: "database", Config: map[string]any{"latencyMs": 18.0, "connectionPool": 60.0, "failureChance": 0.001, "replication": true}},
			{ID: "object-store-1", Type: "object-store", Config: map[string]any{"latencyMs": 25.0, "replicationFactor": 3.0, "durability": "high"}},
		},
		Edges: []model.Edge{
			{ID: "e-gateway-lb", Source: "gateway-1", Target: "lb-1"},
			{ID: "e-lb-service", Source: "lb-1", Target: "service-1"},
			{ID: "e-service-cache", Source: "service-1", Target: "cache-1"},
			{ID: "e-cache-queue", Source: "cache-1", Target: "queue-1"},
			{ID: "e-queue-db", Source: "queue-1", Target: "database-1"},
			{ID: "e-db-obj", Source: "database-1", Target: "object-store-1"},
		},
	}

	return model.SavedTopology{
		ProjectID: projectID,
		Graph:     graph,
		CTM:       canonicalize(graph),
		UpdatedAt: time.Now().UTC(),
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type apiEnvelope struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Error   *apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondV1Data(w http.ResponseWriter, status int, payload any) {
	respondJSON(w, status, apiEnvelope{Success: true, Data: payload, Error: nil})
}

func respondV1Error(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, apiEnvelope{Success: false, Data: nil, Error: &apiError{Code: code, Message: message}})
}
