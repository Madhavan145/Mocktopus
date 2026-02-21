package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"moktopus/chaos-service/internal/model"
	"moktopus/chaos-service/internal/store"
)

type handler struct {
	store *store.InMemoryStore
}

func NewHandler(repo *store.InMemoryStore) http.Handler {
	h := handler{store: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/chaos/scenarios", h.scenarios)
	mux.HandleFunc("/api/v1/chaos/scenarios", h.scenariosV1)
	mux.HandleFunc("/api/chaos/inject", h.inject)
	mux.HandleFunc("/api/v1/chaos/inject", h.injectV1)
	return mux
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "chaos-service", "status": "ok"})
}

func (h handler) scenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"scenarios": h.store.ListScenarios()})
}

func (h handler) scenariosV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	respondV1Data(w, http.StatusOK, map[string]any{"scenarios": h.store.ListScenarios()})
}

func (h handler) inject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request, ok := decodeInject(w, r)
	if !ok {
		return
	}

	response := h.buildInjection(request)
	respondJSON(w, http.StatusOK, response)
}

func (h handler) injectV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	request, ok := decodeInjectV1(w, r)
	if !ok {
		return
	}

	response := h.buildInjection(request)
	respondV1Data(w, http.StatusOK, response)
}

func (h handler) buildInjection(request model.InjectRequest) model.InjectResponse {
	scenarioID := strings.TrimSpace(request.ScenarioID)
	if scenarioID == "" {
		scenarioID = "steady"
	}

	scenario, ok := h.store.GetScenario(scenarioID)
	if ok {
		if request.FailureRate == 0 {
			request.FailureRate = scenario.FailureRate
		}
		if request.SlowdownMs == 0 {
			request.SlowdownMs = scenario.SlowdownMs
		}
		if request.PacketLossPct == 0 {
			request.PacketLossPct = scenario.PacketLossPct
		}
	}

	projectID := strings.TrimSpace(request.ProjectID)
	if projectID == "" {
		projectID = "demo"
	}

	return model.InjectResponse{
		ProjectID:   projectID,
		ScenarioID:  scenarioID,
		AppliedAt:   time.Now().UTC(),
		Status:      "applied",
		Description: "failure injection parameters ready for simulation orchestration",
	}
}

func decodeInject(w http.ResponseWriter, r *http.Request) (model.InjectRequest, bool) {
	var request model.InjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return model.InjectRequest{}, false
	}
	return request, true
}

func decodeInjectV1(w http.ResponseWriter, r *http.Request) (model.InjectRequest, bool) {
	var request model.InjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return model.InjectRequest{}, false
	}
	return request, true
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
