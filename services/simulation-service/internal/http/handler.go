package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"moktopus/simulation-service/internal/engine"
	"moktopus/simulation-service/internal/eventbus"
	"moktopus/simulation-service/internal/model"
	"moktopus/simulation-service/internal/store"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type handler struct {
	engine    engine.Engine
	runs      *runRegistry
	store     store.Store         // nil = no persistence
	publisher *eventbus.Publisher // nil = no event publishing
}

func NewHandler(sim engine.Engine, st store.Store, pub *eventbus.Publisher) http.Handler {
	h := handler{engine: sim, runs: newRunRegistry(), store: st, publisher: pub}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/simulation/run", h.run)
	mux.HandleFunc("/api/v1/simulation/run", h.runV1)
	mux.HandleFunc("/api/simulation/run-stream", h.runStream)
	mux.HandleFunc("/api/simulation/stream", h.wsStream)
	mux.HandleFunc("/api/simulation/runs", h.startRun)
	mux.HandleFunc("/api/v1/simulation/runs", h.startRunV1)
	mux.HandleFunc("/api/simulation/runs/", h.getRun)
	mux.HandleFunc("/api/v1/simulation/runs/", h.getRunV1)
	mux.HandleFunc("/api/v1/simulation/history/", h.listRuns)
	mux.HandleFunc("/api/v1/simulation/compare", h.compareRuns)
	mux.HandleFunc("/api/v1/simulation/traces/", h.getTraces)
	return mux
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "simulation-service", "status": "ok"})
}

func (h handler) run(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce daily simulation limit
	if h.store != nil {
		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			maxSims := h.store.GetOrgMaxSimulations(orgID)
			todayCount := h.store.CountTodayByOrg(orgID)
			if todayCount >= maxSims {
				http.Error(w, "daily simulation limit reached for your plan", http.StatusForbidden)
				return
			}
		}
	}

	var req model.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	result, err := h.engine.Run(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h handler) runV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	// Enforce daily simulation limit
	if h.store != nil {
		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			maxSims := h.store.GetOrgMaxSimulations(orgID)
			todayCount := h.store.CountTodayByOrg(orgID)
			if todayCount >= maxSims {
				respondV1Error(w, http.StatusForbidden, "SIMULATION_LIMIT_REACHED",
					"daily simulation limit reached for your plan")
				return
			}
		}
	}

	var req model.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	result, err := h.engine.Run(req)
	if err != nil {
		respondV1Error(w, http.StatusBadRequest, "RUN_REJECTED", err.Error())
		return
	}

	respondV1Data(w, http.StatusOK, result)
}

func (h handler) runStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	result, err := h.engine.RunWithProgress(req, func(snapshot model.ProgressSnapshot) {
		_ = encoder.Encode(model.StreamMessage{
			Type:     "snapshot",
			Snapshot: &snapshot,
		})
		flusher.Flush()
	})
	if err != nil {
		_ = encoder.Encode(model.StreamMessage{Type: "error", Error: err.Error()})
		flusher.Flush()
		return
	}

	_ = encoder.Encode(model.StreamMessage{Type: "result", Result: &result})
	flusher.Flush()
}

func (h handler) startRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce daily simulation limit
	if h.store != nil {
		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			maxSims := h.store.GetOrgMaxSimulations(orgID)
			todayCount := h.store.CountTodayByOrg(orgID)
			if todayCount >= maxSims {
				http.Error(w, "daily simulation limit reached for your plan", http.StatusForbidden)
				return
			}
		}
	}

	var req model.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	state := h.runs.create()

	// Persist to DB if store available
	if h.store != nil {
		trafficJSON, _ := json.Marshal(req.Traffic)
		topoJSON, _ := json.Marshal(req.Topology)
		_ = h.store.CreateRun(&store.RunRecord{
			ID:               state.id,
			ProjectID:        req.ProjectID,
			Status:           "running",
			Seed:             req.Traffic.Randomness.Seed,
			TrafficConfig:    trafficJSON,
			TopologySnapshot: topoJSON,
		})
	}

	go func(runID string, request model.RunRequest) {
		startedAt := time.Now().UTC().Format(time.RFC3339)

		// Publish start event
		if h.publisher != nil {
			payload, _ := json.Marshal(map[string]any{"seed": request.Traffic.Randomness.Seed})
			_ = h.publisher.Publish(context.Background(), eventbus.Event{
				Type: eventbus.SimulationStarted, Source: "simulation-service",
				RunID: runID, ProjectID: request.ProjectID, Payload: payload,
			})
		}

		result, err := h.engine.RunWithProgress(request, func(snapshot model.ProgressSnapshot) {
			h.runs.updateSnapshot(runID, snapshot)
		})
		if err != nil {
			h.runs.fail(runID, err.Error())
			if h.store != nil {
				completedAt := time.Now().UTC().Format(time.RFC3339)
				_ = h.store.UpdateRun(&store.RunRecord{
					ID: runID, Status: "failed", ErrorMessage: err.Error(),
					StartedAt: &startedAt, CompletedAt: &completedAt,
				})
			}
			if h.publisher != nil {
				payload, _ := json.Marshal(map[string]string{"error": err.Error()})
				_ = h.publisher.Publish(context.Background(), eventbus.Event{
					Type: eventbus.SimulationFailed, Source: "simulation-service",
					RunID: runID, ProjectID: request.ProjectID, Payload: payload,
				})
			}
			return
		}
		h.runs.complete(runID, result)
		if h.store != nil {
			completedAt := time.Now().UTC().Format(time.RFC3339)
			resultJSON, _ := json.Marshal(result)
			_ = h.store.UpdateRun(&store.RunRecord{
				ID: runID, Status: "completed", Result: resultJSON,
				DurationMs:    int64(result.Metadata.VirtualDurationMs),
				TotalRequests: result.Metrics.RequestsTotal, TotalCompleted: result.Metrics.Completed,
				TotalFailed: result.Metrics.Failed,
				StartedAt:   &startedAt, CompletedAt: &completedAt,
			})
			_ = h.store.SaveEvents(runID, result.Events)
		}
		// Publish completion event
		if h.publisher != nil {
			payload, _ := json.Marshal(result.Metrics)
			_ = h.publisher.Publish(context.Background(), eventbus.Event{
				Type: eventbus.SimulationCompleted, Source: "simulation-service",
				RunID: runID, ProjectID: request.ProjectID, Payload: payload,
			})
		}
	}(state.id, req)

	respondJSON(w, http.StatusAccepted, model.StartRunResponse{RunID: state.id, Status: state.status})
}

func (h handler) startRunV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	// Enforce daily simulation limit
	if h.store != nil {
		orgID := r.Header.Get("X-Org-ID")
		if orgID != "" {
			maxSims := h.store.GetOrgMaxSimulations(orgID)
			todayCount := h.store.CountTodayByOrg(orgID)
			if todayCount >= maxSims {
				respondV1Error(w, http.StatusForbidden, "SIMULATION_LIMIT_REACHED",
					"daily simulation limit reached for your plan")
				return
			}
		}
	}

	var req model.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	state := h.runs.create()

	if h.store != nil {
		trafficJSON, _ := json.Marshal(req.Traffic)
		topoJSON, _ := json.Marshal(req.Topology)
		_ = h.store.CreateRun(&store.RunRecord{
			ID:               state.id,
			ProjectID:        req.ProjectID,
			Status:           "running",
			Seed:             req.Traffic.Randomness.Seed,
			TrafficConfig:    trafficJSON,
			TopologySnapshot: topoJSON,
		})
	}

	go func(runID string, request model.RunRequest) {
		startedAt := time.Now().UTC().Format(time.RFC3339)

		if h.publisher != nil {
			payload, _ := json.Marshal(map[string]any{"seed": request.Traffic.Randomness.Seed})
			_ = h.publisher.Publish(context.Background(), eventbus.Event{
				Type: eventbus.SimulationStarted, Source: "simulation-service",
				RunID: runID, ProjectID: request.ProjectID, Payload: payload,
			})
		}

		result, err := h.engine.RunWithProgress(request, func(snapshot model.ProgressSnapshot) {
			h.runs.updateSnapshot(runID, snapshot)
		})
		if err != nil {
			h.runs.fail(runID, err.Error())
			if h.store != nil {
				completedAt := time.Now().UTC().Format(time.RFC3339)
				_ = h.store.UpdateRun(&store.RunRecord{
					ID: runID, Status: "failed", ErrorMessage: err.Error(),
					StartedAt: &startedAt, CompletedAt: &completedAt,
				})
			}
			if h.publisher != nil {
				payload, _ := json.Marshal(map[string]string{"error": err.Error()})
				_ = h.publisher.Publish(context.Background(), eventbus.Event{
					Type: eventbus.SimulationFailed, Source: "simulation-service",
					RunID: runID, ProjectID: request.ProjectID, Payload: payload,
				})
			}
			return
		}
		h.runs.complete(runID, result)
		if h.store != nil {
			completedAt := time.Now().UTC().Format(time.RFC3339)
			resultJSON, _ := json.Marshal(result)
			_ = h.store.UpdateRun(&store.RunRecord{
				ID: runID, Status: "completed", Result: resultJSON,
				DurationMs:    int64(result.Metadata.VirtualDurationMs),
				TotalRequests: result.Metrics.RequestsTotal, TotalCompleted: result.Metrics.Completed,
				TotalFailed: result.Metrics.Failed,
				StartedAt:   &startedAt, CompletedAt: &completedAt,
			})
			_ = h.store.SaveEvents(runID, result.Events)
		}
		if h.publisher != nil {
			payload, _ := json.Marshal(result.Metrics)
			_ = h.publisher.Publish(context.Background(), eventbus.Event{
				Type: eventbus.SimulationCompleted, Source: "simulation-service",
				RunID: runID, ProjectID: request.ProjectID, Payload: payload,
			})
		}
	}(state.id, req)

	respondV1Data(w, http.StatusAccepted, model.StartRunResponse{RunID: state.id, Status: state.status})
}

func (h handler) getRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/simulation/runs/")
	runID = strings.TrimSpace(runID)
	if runID == "" || strings.Contains(runID, "/") {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	state, ok := h.runs.get(runID)
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	response := model.RunStateResponse{
		RunID:     state.id,
		Status:    state.status,
		StartedAt: state.startedAt.Format("2006-01-02T15:04:05.000Z07:00"),
		Snapshot:  state.snapshot,
		Result:    state.result,
		Error:     state.error,
	}
	if state.completedAt != nil {
		response.CompletedAt = state.completedAt.Format("2006-01-02T15:04:05.000Z07:00")
	}

	respondJSON(w, http.StatusOK, response)
}

func (h handler) getRunV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/simulation/runs/")
	runID = strings.TrimSpace(runID)
	if runID == "" || strings.Contains(runID, "/") {
		respondV1Error(w, http.StatusBadRequest, "INVALID_RUN_ID", "invalid run id")
		return
	}

	state, ok := h.runs.get(runID)
	if !ok {
		respondV1Error(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found")
		return
	}

	response := model.RunStateResponse{
		RunID:     state.id,
		Status:    state.status,
		StartedAt: state.startedAt.Format("2006-01-02T15:04:05.000Z07:00"),
		Snapshot:  state.snapshot,
		Result:    state.result,
		Error:     state.error,
	}
	if state.completedAt != nil {
		response.CompletedAt = state.completedAt.Format("2006-01-02T15:04:05.000Z07:00")
	}

	respondV1Data(w, http.StatusOK, response)
}

func (h handler) listRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	projectID := strings.TrimPrefix(r.URL.Path, "/api/v1/simulation/history/")
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "project id required")
		return
	}

	if h.store == nil {
		respondV1Data(w, http.StatusOK, []any{})
		return
	}

	runs, err := h.store.ListRuns(projectID, 50)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list runs")
		return
	}

	respondV1Data(w, http.StatusOK, runs)
}

// wsStream provides a WebSocket endpoint for real-time simulation progress.
// The gateway proxies /ws/simulation_updates to this endpoint.
// Message format per SRS: {"type": "simulation_update", "runId": "...", "payload": {...}}
func (h handler) wsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Read incoming command to start streaming for a specific run.
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}

	var cmd struct {
		Action    string              `json:"action"`
		RunID     string              `json:"runId,omitempty"`
		ProjectID string              `json:"projectId,omitempty"`
		Topology  model.Topology      `json:"topology,omitempty"`
		Traffic   model.TrafficConfig `json:"traffic,omitempty"`
	}
	if err := json.Unmarshal(msg, &cmd); err != nil {
		_ = conn.WriteJSON(map[string]any{
			"type":    "error",
			"payload": map[string]string{"message": "invalid command"},
		})
		return
	}

	switch cmd.Action {
	case "subscribe":
		// Subscribe to updates for an existing running simulation.
		h.subscribeToRun(conn, cmd.RunID)
	case "run":
		// Start a new simulation and stream progress via WebSocket.
		h.runViaWebSocket(conn, cmd.ProjectID, cmd.Topology, cmd.Traffic)
	default:
		_ = conn.WriteJSON(map[string]any{
			"type":    "error",
			"payload": map[string]string{"message": "unknown action, use 'subscribe' or 'run'"},
		})
	}
}

func (h handler) subscribeToRun(conn *websocket.Conn, runID string) {
	if runID == "" {
		_ = conn.WriteJSON(map[string]any{
			"type":    "error",
			"payload": map[string]string{"message": "runId required"},
		})
		return
	}

	// Poll run state and push updates until completed.
	for {
		state, ok := h.runs.get(runID)
		if !ok {
			_ = conn.WriteJSON(map[string]any{
				"type":  "simulation_update",
				"runId": runID,
				"payload": map[string]any{
					"status": "not_found",
				},
			})
			return
		}

		msg := map[string]any{
			"type":  "simulation_update",
			"runId": runID,
			"payload": map[string]any{
				"status": state.status,
			},
		}
		if state.snapshot != nil {
			msg["payload"] = map[string]any{
				"status":   state.status,
				"progress": state.snapshot.Progress,
				"metrics":  state.snapshot.Metrics,
			}
		}

		if err := conn.WriteJSON(msg); err != nil {
			return
		}

		if state.status == "completed" || state.status == "failed" {
			if state.result != nil {
				_ = conn.WriteJSON(map[string]any{
					"type":  "simulation_update",
					"runId": runID,
					"payload": map[string]any{
						"status":  state.status,
						"metrics": state.result.Metrics,
					},
				})
			}
			return
		}

		time.Sleep(300 * time.Millisecond)
	}
}

func (h handler) runViaWebSocket(conn *websocket.Conn, projectID string, topology model.Topology, traffic model.TrafficConfig) {
	req := model.RunRequest{
		ProjectID: projectID,
		Topology:  topology,
		Traffic:   traffic,
	}

	runID := h.runs.create().id

	_ = conn.WriteJSON(map[string]any{
		"type":  "simulation_update",
		"runId": runID,
		"payload": map[string]any{
			"status": "started",
		},
	})

	result, err := h.engine.RunWithProgress(req, func(snapshot model.ProgressSnapshot) {
		_ = conn.WriteJSON(map[string]any{
			"type":  "simulation_update",
			"runId": runID,
			"payload": map[string]any{
				"status":   "running",
				"progress": snapshot.Progress,
				"metrics":  snapshot.Metrics,
			},
		})
	})
	if err != nil {
		_ = conn.WriteJSON(map[string]any{
			"type":  "simulation_update",
			"runId": runID,
			"payload": map[string]any{
				"status": "failed",
				"error":  err.Error(),
			},
		})
		return
	}

	_ = conn.WriteJSON(map[string]any{
		"type":  "simulation_update",
		"runId": runID,
		"payload": map[string]any{
			"status":  "completed",
			"metrics": result.Metrics,
		},
	})
}

// compareRuns implements the Architecture Comparison Engine (SRS Section 90).
// POST /api/v1/simulation/compare
// Body: { "runIds": ["run-a", "run-b"] }
// Returns a structured comparison with verdict.
func (h handler) compareRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	if h.store == nil {
		respondV1Error(w, http.StatusServiceUnavailable, "NO_STORE", "persistence not available")
		return
	}

	var req struct {
		RunIDs []string `json:"runIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request body")
		return
	}
	if len(req.RunIDs) < 2 {
		respondV1Error(w, http.StatusBadRequest, "INSUFFICIENT_RUNS", "at least 2 run IDs required")
		return
	}
	if len(req.RunIDs) > 5 {
		respondV1Error(w, http.StatusBadRequest, "TOO_MANY_RUNS", "maximum 5 runs for comparison")
		return
	}

	type runMetrics = compareRunMetrics

	runs := make([]runMetrics, 0, len(req.RunIDs))
	for _, runID := range req.RunIDs {
		record, err := h.store.GetRun(runID)
		if err != nil || record == nil {
			respondV1Error(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found: "+runID)
			return
		}
		if record.Status != "completed" {
			respondV1Error(w, http.StatusBadRequest, "RUN_INCOMPLETE", "run not completed: "+runID)
			return
		}
		var result model.RunResponse
		if err := json.Unmarshal(record.Result, &result); err != nil {
			respondV1Error(w, http.StatusInternalServerError, "PARSE_ERROR", "failed to parse run result")
			return
		}
		runs = append(runs, runMetrics{RunID: runID, Metrics: result.Metrics})
	}

	// Build comparison dimensions.
	dimensions := []comparisonDimension{
		buildDimension("Throughput", runs, func(m model.Metrics) float64 { return m.Throughput }, "rps", true),
		buildDimension("Avg Latency", runs, func(m model.Metrics) float64 { return m.AvgLatencyMs }, "ms", false),
		buildDimension("P95 Latency", runs, func(m model.Metrics) float64 { return m.P95LatencyMs }, "ms", false),
		buildDimension("P99 Latency", runs, func(m model.Metrics) float64 { return m.P99LatencyMs }, "ms", false),
		buildDimension("Error Rate", runs, func(m model.Metrics) float64 { return m.ErrorRate }, "%", false),
		buildDimension("Queue Wait", runs, func(m model.Metrics) float64 { return m.QueueWaitMs }, "ms", false),
		buildDimension("CPU Estimate", runs, func(m model.Metrics) float64 { return m.CPUEstimate }, "%", false),
	}

	// Generate verdict: which run is recommended.
	scores := make(map[string]int)
	for _, dim := range dimensions {
		if dim.Best != "" {
			scores[dim.Best]++
		}
	}
	recommended := ""
	bestScore := 0
	for runID, score := range scores {
		if score > bestScore {
			bestScore = score
			recommended = runID
		}
	}

	// Build reason string.
	var reasons []string
	if recommended != "" {
		for _, dim := range dimensions {
			if dim.Best == recommended {
				reasons = append(reasons, "Best "+dim.Name)
			}
		}
	}

	// Tradeoffs: dimensions where the recommended is NOT best.
	var tradeoffs []string
	if recommended != "" {
		for _, dim := range dimensions {
			if dim.Best != "" && dim.Best != recommended {
				tradeoffs = append(tradeoffs, dim.Name+" favors "+dim.Best)
			}
		}
	}

	verdict := map[string]any{
		"recommended": recommended,
		"score":       scores,
		"reasons":     reasons,
		"tradeoffs":   tradeoffs,
	}

	respondV1Data(w, http.StatusOK, map[string]any{
		"runs":       runs,
		"dimensions": dimensions,
		"verdict":    verdict,
	})
}

type compareRunMetrics struct {
	RunID   string        `json:"runId"`
	Metrics model.Metrics `json:"metrics"`
}

type comparisonDimension struct {
	Name   string             `json:"name"`
	Values map[string]float64 `json:"values"`
	Best   string             `json:"best"`
	Unit   string             `json:"unit"`
}

func buildDimension(name string, runs []compareRunMetrics, extractor func(model.Metrics) float64, unit string, higherIsBetter bool) comparisonDimension {
	values := make(map[string]float64)
	best := ""
	var bestVal float64
	for i, run := range runs {
		val := extractor(run.Metrics)
		values[run.RunID] = val
		if i == 0 {
			bestVal = val
			best = run.RunID
		} else {
			if higherIsBetter && val > bestVal {
				bestVal = val
				best = run.RunID
			} else if !higherIsBetter && val < bestVal {
				bestVal = val
				best = run.RunID
			}
		}
	}
	return comparisonDimension{Name: name, Values: values, Best: best, Unit: unit}
}

// getTraces extracts individual request traces from simulation events for waterfall visualization.
func (h handler) getTraces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	// Extract run ID from path: /api/v1/simulation/traces/{runId}
	path := r.URL.Path
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) < 2 {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PATH", "run ID required")
		return
	}
	runID := parts[len(parts)-1]

	entry, ok := h.runs.get(runID)
	if !ok || entry == nil {
		respondV1Error(w, http.StatusNotFound, "RUN_NOT_FOUND", "simulation run not found")
		return
	}
	if entry.result == nil {
		respondV1Error(w, http.StatusConflict, "RUN_IN_PROGRESS", "simulation still running")
		return
	}

	// Parse query params
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	statusFilter := r.URL.Query().Get("status") // "completed", "failed", or empty for all

	// Build traces from events
	traces := buildRequestTraces(entry.result.Events, limit, statusFilter)

	respondV1Data(w, http.StatusOK, map[string]any{
		"runId":      runID,
		"traceCount": len(traces),
		"traces":     traces,
	})
}

type requestTrace struct {
	RequestID    int             `json:"requestId"`
	Status       string          `json:"status"` // completed, failed
	TotalLatency float64         `json:"totalLatencyMs"`
	Path         []traceSegment  `json:"path"`
}

type traceSegment struct {
	NodeID   string  `json:"nodeId"`
	EnterMs  float64 `json:"enterMs"`
	ExitMs   float64 `json:"exitMs"`
	Duration float64 `json:"durationMs"`
	Status   string  `json:"status"` // ok, failed, queued
}

func buildRequestTraces(events []model.SimulationEvent, limit int, statusFilter string) []requestTrace {
	// Group events by request ID
	type reqEvents struct {
		events []model.SimulationEvent
		status string
	}
	byRequest := map[int]*reqEvents{}

	for _, ev := range events {
		if _, ok := byRequest[ev.RequestID]; !ok {
			byRequest[ev.RequestID] = &reqEvents{status: "in-progress"}
		}
		re := byRequest[ev.RequestID]
		re.events = append(re.events, ev)
		if ev.Type == "REQUEST_COMPLETE" {
			re.status = "completed"
		} else if ev.Type == "REQUEST_FAILED" {
			re.status = "failed"
		}
	}

	// Build traces
	traces := make([]requestTrace, 0, limit)
	for reqID, re := range byRequest {
		if statusFilter != "" && re.status != statusFilter {
			continue
		}
		if len(traces) >= limit {
			break
		}

		trace := requestTrace{
			RequestID: reqID,
			Status:    re.status,
		}

		// Build path from events
		var segments []traceSegment
		var currentNode string
		var enterTime float64

		for _, ev := range re.events {
			switch ev.Type {
			case "REQUEST_RECEIVED":
				// Start of request
			case "PROCESS_START":
				currentNode = ev.NodeID
				enterTime = ev.TimeMs
			case "PROCESS_COMPLETE":
				if currentNode != "" {
					segments = append(segments, traceSegment{
						NodeID:   currentNode,
						EnterMs:  enterTime,
						ExitMs:   ev.TimeMs,
						Duration: ev.TimeMs - enterTime,
						Status:   "ok",
					})
				}
				currentNode = ""
			case "FAILED":
				if currentNode != "" {
					segments = append(segments, traceSegment{
						NodeID:   ev.NodeID,
						EnterMs:  enterTime,
						ExitMs:   ev.TimeMs,
						Duration: ev.TimeMs - enterTime,
						Status:   "failed",
					})
				}
				currentNode = ""
			case "QUEUED":
				if ev.NodeID != "" {
					segments = append(segments, traceSegment{
						NodeID:   ev.NodeID,
						EnterMs:  ev.TimeMs,
						ExitMs:   ev.TimeMs,
						Duration: 0,
						Status:   "queued",
					})
				}
			}
		}

		trace.Path = segments
		if len(segments) > 0 {
			trace.TotalLatency = segments[len(segments)-1].ExitMs - re.events[0].TimeMs
		}

		traces = append(traces, trace)
	}

	return traces
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
