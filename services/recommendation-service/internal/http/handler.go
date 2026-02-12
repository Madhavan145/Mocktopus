package http

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"moktopus/recommendation-service/internal/llm"
	"moktopus/recommendation-service/internal/model"
)

// LLMRegistry is set by main to give the handler access to the LLM provider registry.
var LLMRegistry *llm.Registry

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/recommendation/derive", derive)
	mux.HandleFunc("/api/v1/recommendation/derive", deriveV1)
	mux.HandleFunc("/api/v1/recommendation/configure-llm", configureLLM)
	mux.HandleFunc("/api/v1/recommendation/llm-status", llmStatus)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "recommendation-service", "status": "ok"})
}

func derive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.DeriveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, evaluate(req))
}

func deriveV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req model.DeriveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	// Try LLM-enhanced analysis if a provider is configured for this org
	orgID := r.Header.Get("X-Org-ID")
	useLLM := r.URL.Query().Get("llm") != "false" // opt-out via ?llm=false

	if useLLM && orgID != "" && LLMRegistry != nil {
		if provider := LLMRegistry.Get(orgID); provider != nil && provider.Available() {
			llmResp, err := runLLMAnalysis(r.Context(), provider, req)
			if err == nil {
				respondV1Data(w, http.StatusOK, llmResp)
				return
			}
			// Fall through to rule-based on LLM failure (graceful degradation)
		}
	}

	respondV1Data(w, http.StatusOK, evaluate(req))
}

func evaluate(req model.DeriveRequest) model.DeriveResponse {
	recs := make([]model.Recommendation, 0, 4)

	if req.Metrics.P95LatencyMs > 180 {
		recs = append(recs, model.Recommendation{
			ID:                      "rec-latency-cache",
			Problem:                 "High p95 latency",
			Reason:                  "Tail latency is elevated under current traffic profile",
			Fix:                     "Introduce cache layer for repeated reads",
			ExpectedLatencyDeltaPct: -28,
			ExpectedCostDeltaPct:    10,
			RiskScore:               0.72,
		})
	}

	if req.Metrics.QueueDepthMax > 120 || req.Metrics.QueueWaitMs > 100 {
		recs = append(recs, model.Recommendation{
			ID:                      "rec-queue-workers",
			Problem:                 "Queue saturation risk",
			Reason:                  "Observed queue depth and wait time indicate processing bottleneck",
			Fix:                     "Increase worker concurrency or scale worker pool",
			ExpectedLatencyDeltaPct: -22,
			ExpectedCostDeltaPct:    8,
			RiskScore:               0.78,
		})
	}

	if req.Metrics.CPUEstimate > 75 {
		recs = append(recs, model.Recommendation{
			ID:                      "rec-autoscale",
			Problem:                 "Sustained high CPU",
			Reason:                  "CPU estimate is above resilient operating threshold",
			Fix:                     "Enable autoscaling with conservative cooldown",
			ExpectedLatencyDeltaPct: -15,
			ExpectedCostDeltaPct:    12,
			RiskScore:               0.65,
		})
	}

	if req.Metrics.ErrorRate > 0.03 {
		recs = append(recs, model.Recommendation{
			ID:                      "rec-retry-policy",
			Problem:                 "Elevated failure rate",
			Reason:                  "Current retry profile may be insufficient for intermittent failures",
			Fix:                     "Tune retry policy and add circuit breaker on unstable dependency",
			ExpectedLatencyDeltaPct: -8,
			ExpectedCostDeltaPct:    4,
			RiskScore:               0.74,
		})
	}

	if len(recs) == 0 {
		recs = append(recs, model.Recommendation{
			ID:                      "rec-stable",
			Problem:                 "No immediate bottleneck detected",
			Reason:                  "Current metrics are within expected operating range",
			Fix:                     "Keep topology and monitor growth scenarios",
			ExpectedLatencyDeltaPct: 0,
			ExpectedCostDeltaPct:    0,
			RiskScore:               0.21,
		})
	}

	confidence := 0.75
	if len(req.Topology.Nodes) > 0 {
		known := 0
		for _, node := range req.Topology.Nodes {
			t := strings.ToLower(strings.TrimSpace(node.Type))
			if t != "" {
				known++
			}
		}
		coverage := float64(known) / float64(len(req.Topology.Nodes))
		confidence = math.Min(0.95, 0.6+coverage*0.35)
	}

	return model.DeriveResponse{
		ProjectID:       req.ProjectID,
		Confidence:      round(confidence * 100),
		Recommendations: recs,
	}
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
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

// --- LLM-enhanced recommendation logic ---

func runLLMAnalysis(ctx context.Context, provider llm.Provider, req model.DeriveRequest) (model.DeriveResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	userMsg := llm.BuildUserMessage(req.Topology, req.Metrics)
	completion, err := provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: llm.SystemPrompt,
		UserMessage:  userMsg,
		Temperature:  0.2,
		MaxTokens:    2048,
	})
	if err != nil {
		return model.DeriveResponse{}, err
	}

	parsed, err := llm.ParseLLMRecommendations(completion.Content)
	if err != nil {
		return model.DeriveResponse{}, err
	}

	recs := make([]model.Recommendation, 0, len(parsed))
	for i, p := range parsed {
		recs = append(recs, model.Recommendation{
			ID:                      fmt.Sprintf("llm-rec-%d", i+1),
			Problem:                 p.Problem,
			Reason:                  p.Reason,
			Fix:                     p.Fix,
			ExpectedLatencyDeltaPct: p.ExpectedLatencyDeltaPct,
			ExpectedCostDeltaPct:    p.ExpectedCostDeltaPct,
			RiskScore:               p.RiskScore,
		})
	}

	return model.DeriveResponse{
		ProjectID:       req.ProjectID,
		Confidence:      92.0, // LLM-driven analysis gets higher confidence
		Recommendations: recs,
		Source:          "llm:" + provider.Name(),
		TokensUsed:      completion.TokensUsed,
	}, nil
}

// --- LLM Configuration Endpoints ---

func configureLLM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var cfg llm.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	if cfg.OrgID == "" {
		respondV1Error(w, http.StatusBadRequest, "MISSING_ORG_ID", "orgId is required")
		return
	}
	if cfg.Type == "" {
		respondV1Error(w, http.StatusBadRequest, "MISSING_TYPE", "type is required (openai, anthropic, ollama, custom)")
		return
	}

	if LLMRegistry == nil {
		respondV1Error(w, http.StatusInternalServerError, "REGISTRY_NOT_INITIALIZED", "LLM registry not available")
		return
	}

	if err := LLMRegistry.Configure(cfg); err != nil {
		respondV1Error(w, http.StatusBadRequest, "CONFIG_ERROR", err.Error())
		return
	}

	respondV1Data(w, http.StatusOK, map[string]any{
		"orgId":    cfg.OrgID,
		"provider": cfg.Type,
		"model":    cfg.Model,
		"status":   "configured",
	})
}

func llmStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		respondV1Error(w, http.StatusBadRequest, "MISSING_ORG_ID", "orgId query param required")
		return
	}

	if LLMRegistry == nil {
		respondV1Data(w, http.StatusOK, map[string]any{"configured": false})
		return
	}

	cfg, ok := LLMRegistry.GetConfig(orgID)
	if !ok {
		respondV1Data(w, http.StatusOK, map[string]any{"configured": false})
		return
	}

	provider := LLMRegistry.Get(orgID)
	available := provider != nil && provider.Available()

	respondV1Data(w, http.StatusOK, map[string]any{
		"configured": true,
		"provider":   cfg.Type,
		"model":      cfg.Model,
		"endpoint":   cfg.Endpoint,
		"available":  available,
	})
}
