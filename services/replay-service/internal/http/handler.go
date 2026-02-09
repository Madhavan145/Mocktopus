package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"moktopus/replay-service/internal/model"
	"moktopus/replay-service/internal/store"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type replayHandler struct {
	store store.Store
}

func NewHandler(st store.Store) http.Handler {
	h := &replayHandler{store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/replay/reconstruct", h.reconstruct)
	mux.HandleFunc("/api/v1/replay/reconstruct", h.reconstructV1)
	mux.HandleFunc("/api/replay/stream", stream)
	mux.HandleFunc("/api/v1/replay/snapshots/", h.getSnapshots)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "replay-service", "status": "ok"})
}

func (h *replayHandler) reconstruct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.ReconstructRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	response := buildReplay(req)
	if h.store != nil && len(response.Snapshots) > 0 {
		_ = h.store.SaveSnapshots(req.RunID, response.Snapshots)
	}
	respondJSON(w, http.StatusOK, response)
}

func (h *replayHandler) reconstructV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req model.ReconstructRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	response := buildReplay(req)
	if h.store != nil && len(response.Snapshots) > 0 {
		_ = h.store.SaveSnapshots(req.RunID, response.Snapshots)
	}
	respondV1Data(w, http.StatusOK, response)
}

func (h *replayHandler) getSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/replay/snapshots/")
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

func stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	runID := r.URL.Query().Get("runId")
	if runID == "" {
		runID = "demo"
	}

	// Send initial connection acknowledgment per SRS message format.
	_ = conn.WriteJSON(map[string]any{
		"type":  "replay_update",
		"runId": runID,
		"payload": map[string]any{
			"status":  "connected",
			"message": "replay stream connected",
		},
	})

	// Simulate replay timeline frames (10 frames representing event progression).
	totalFrames := 10
	for frame := 1; frame <= totalFrames; frame++ {
		progress := float64(frame) / float64(totalFrames)
		msg := map[string]any{
			"type":  "replay_update",
			"runId": runID,
			"payload": map[string]any{
				"frame":      frame,
				"progress":   progress,
				"eventIndex": frame * 25,
				"nodeState": map[string]any{
					"_summary": map[string]any{
						"touches":   frame * 25,
						"failures":  frame * 2,
						"completed": frame * 20,
					},
				},
			},
		}
		if err := conn.WriteJSON(msg); err != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Send completion message.
	_ = conn.WriteJSON(map[string]any{
		"type":  "replay_update",
		"runId": runID,
		"payload": map[string]any{
			"status":   "completed",
			"progress": 1.0,
		},
	})
}

func buildReplay(req model.ReconstructRequest) model.ReconstructResponse {
	stride := req.SnapshotStride
	if stride <= 0 {
		stride = 1000
	}
	if req.RunID == "" {
		req.RunID = "unknown"
	}

	nodeState := map[string]model.NodeState{}
	snapshots := make([]model.Snapshot, 0, len(req.Events)/stride+1)

	for index, event := range req.Events {
		if event.NodeID != "" {
			current := nodeState[event.NodeID]
			current.Touches++
			switch event.Type {
			case "FAILED", "REQUEST_FAILED", "NODE_FAILURE":
				current.Failures++
			case "REQUEST_COMPLETE", "REQUEST_COMPLETED", "PROCESS_COMPLETE", "NODE_PROCESS_COMPLETE":
				current.Completed++
			case "QUEUED":
				current.Queued++
			}
			nodeState[event.NodeID] = current
		}

		if (index+1)%stride == 0 || index == len(req.Events)-1 {
			snapshots = append(snapshots, model.Snapshot{
				EventIndex: index,
				Timestamp:  event.TimeMs,
				NodeState:  cloneNodeState(nodeState),
			})
		}
	}

	return model.ReconstructResponse{
		RunID:           req.RunID,
		SnapshotStride:  stride,
		Snapshots:       snapshots,
		EventsProcessed: len(req.Events),
	}
}

func cloneNodeState(source map[string]model.NodeState) map[string]model.NodeState {
	cloned := make(map[string]model.NodeState, len(source))
	for nodeID, state := range source {
		cloned[nodeID] = state
	}
	return cloned
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
