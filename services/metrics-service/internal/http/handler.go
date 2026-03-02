package http

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"moktopus/metrics-service/internal/model"
	"moktopus/metrics-service/internal/store"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type metricsHandler struct {
	store store.Store
}

func NewHandler(st store.Store) http.Handler {
	h := &metricsHandler{store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/metrics/derive", h.derive)
	mux.HandleFunc("/api/v1/metrics/derive", h.deriveV1)
	mux.HandleFunc("/api/metrics/stream", h.stream)
	mux.HandleFunc("/api/v1/metrics/snapshots/", h.getSnapshots)
	mux.HandleFunc("/api/v1/metrics/nodes/", h.getNodeMetrics)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "metrics-service", "status": "ok"})
}

func (h *metricsHandler) derive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.DeriveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, deriveResponse(req.ProjectID, req.Metrics))
}

func (h *metricsHandler) deriveV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req model.DeriveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	respondV1Data(w, http.StatusOK, deriveResponse(req.ProjectID, req.Metrics))
}

func (h *metricsHandler) stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	projectID := r.URL.Query().Get("projectId")
	duration := 10
	requestsPerSecond := 25
	if projectID == "" {
		projectID = "demo"
	}

	for tick := 1; tick <= duration; tick++ {
		progress := float64(tick) / float64(duration)
		metrics := model.SimulationMetrics{
			RequestsTotal: duration * requestsPerSecond,
			Completed:     tick * requestsPerSecond,
			Failed:        int(float64(tick*requestsPerSecond) * 0.03),
			RPS:           round(float64(requestsPerSecond) * (0.85 + progress*0.15)),
			AvgLatencyMs:  round(45 + progress*18),
			P95LatencyMs:  round(70 + progress*35),
			ErrorRate:     round(0.01 + progress*0.03),
			QueueDepthMax: int(8 + progress*30),
			QueueWaitMs:   round(progress * 120),
			Throughput:    round(float64(requestsPerSecond) * (0.8 + progress*0.2)),
			CPUEstimate:   round(35 + progress*45),
		}

		derived := deriveResponse(projectID, metrics)
		snapshot := model.LiveSnapshot{
			ProjectID:       projectID,
			Tick:            tick,
			Progress:        round(progress),
			Summary:         metrics,
			BottleneckScore: derived.BottleneckScore,
			CapacityHint:    derived.CapacityHint,
		}

		if err := conn.WriteJSON(snapshot); err != nil {
			return
		}

		time.Sleep(350 * time.Millisecond)
	}
}

func deriveResponse(projectID string, metrics model.SimulationMetrics) model.DeriveResponse {
	bottleneckScore := math.Min(100,
		metrics.CPUEstimate*0.4+
			metrics.ErrorRate*100*0.25+
			float64(metrics.QueueDepthMax)*0.015+
			metrics.P95LatencyMs*0.015+
			metrics.LatencyJitter*20+
			metrics.SLOBurnRate*10,
	)

	hint := "Healthy under current load"
	if bottleneckScore > 80 {
		hint = "Critical bottleneck — immediate horizontal scaling or load shedding required"
	} else if bottleneckScore > 60 {
		hint = "High bottleneck risk — add capacity, enable circuit breakers, or introduce queue buffering"
	} else if bottleneckScore > 40 {
		hint = "Moderate load pressure — monitor scaling thresholds and review SLO burn rate"
	} else if bottleneckScore > 20 {
		hint = "Low risk — system operating within healthy margins"
	}

	series := make([]model.TimeseriesPoint, 0, 12)
	for i := 0; i < 12; i++ {
		factor := 0.75 + float64(i)*0.05
		series = append(series, model.TimeseriesPoint{
			TimeIndex: i,
			RPS:       round(metrics.RPS * factor),
			LatencyMs: round(metrics.AvgLatencyMs * (2 - factor)),
			ErrorRate: round(metrics.ErrorRate * (1 + float64(i)*0.03)),
		})
	}

	// SLO compliance: % of requests under SLO target (99.9%)
	sloCompliance := 0.0
	if metrics.Availability > 0 {
		sloCompliance = math.Min(100, metrics.Availability*100)
	}

	// Scalability grade based on queue saturation + CPU + error rate
	scalabilityGrade := gradeScalability(metrics)

	// Reliability tier from availability nines
	reliabilityTier := "Below Two Nines"
	if metrics.AvailabilityNines >= 4 {
		reliabilityTier = "Four Nines (99.99%)"
	} else if metrics.AvailabilityNines >= 3 {
		reliabilityTier = "Three Nines (99.9%)"
	} else if metrics.AvailabilityNines >= 2 {
		reliabilityTier = "Two Nines (99%)"
	} else if metrics.AvailabilityNines >= 1 {
		reliabilityTier = "One Nine (90%)"
	}

	// Cost efficiency assessment
	costEfficiency := "Optimal"
	if metrics.CostPerRequest > 0.0001 {
		costEfficiency = "Expensive"
	} else if metrics.CostPerRequest > 0.00001 {
		costEfficiency = "Moderate"
	}

	// Latency grade: based on P99/P50 tail amplification ratio
	latencyGrade := "A"
	if metrics.P50LatencyMs > 0 {
		ratio := metrics.P99LatencyMs / metrics.P50LatencyMs
		if ratio > 20 {
			latencyGrade = "F"
		} else if ratio > 10 {
			latencyGrade = "D"
		} else if ratio > 5 {
			latencyGrade = "C"
		} else if ratio > 3 {
			latencyGrade = "B"
		}
	}

	return model.DeriveResponse{
		ProjectID:        projectID,
		Summary:          metrics,
		BottleneckScore:  round(bottleneckScore),
		CapacityHint:     hint,
		Series:           series,
		SLOCompliance:    round(sloCompliance),
		ScalabilityGrade: scalabilityGrade,
		ReliabilityTier:  reliabilityTier,
		CostEfficiency:   costEfficiency,
		LatencyGrade:     latencyGrade,
	}
}

func gradeScalability(m model.SimulationMetrics) string {
	score := 100.0
	// Penalize high CPU
	if m.CPUEstimate > 80 {
		score -= (m.CPUEstimate - 80) * 2
	}
	// Penalize high error rate
	score -= m.ErrorRate * 200
	// Penalize large queues relative to throughput
	if m.Throughput > 0 {
		queueRatio := float64(m.QueueDepthMax) / m.Throughput
		if queueRatio > 1 {
			score -= queueRatio * 10
		}
	}
	// Penalize SLO burn
	if m.SLOBurnRate > 1 {
		score -= (m.SLOBurnRate - 1) * 20
	}

	if score >= 90 {
		return "A"
	} else if score >= 75 {
		return "B"
	} else if score >= 60 {
		return "C"
	} else if score >= 40 {
		return "D"
	}
	return "F"
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

func round(value float64) float64 {
	return math.Round(value*100) / 100
}

func (h *metricsHandler) getSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/snapshots/")
	runID = strings.TrimSpace(runID)
	if runID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_RUN_ID", "run id required")
		return
	}

	if h.store == nil {
		respondV1Data(w, http.StatusOK, []any{})
		return
	}

	snapshots, err := h.store.GetSnapshots(runID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get snapshots")
		return
	}
	respondV1Data(w, http.StatusOK, snapshots)
}

func (h *metricsHandler) getNodeMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/nodes/")
	runID = strings.TrimSpace(runID)
	if runID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_RUN_ID", "run id required")
		return
	}

	if h.store == nil {
		respondV1Data(w, http.StatusOK, []any{})
		return
	}

	metrics, err := h.store.GetNodeMetrics(runID)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get node metrics")
		return
	}
	respondV1Data(w, http.StatusOK, metrics)
}
