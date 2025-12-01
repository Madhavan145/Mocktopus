package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"moktopus/project-service/internal/model"
	"moktopus/project-service/internal/store"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type handler struct {
	store store.Store
}

func NewHandler(repo store.Store) http.Handler {
	h := handler{store: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/projects", h.projects)
	mux.HandleFunc("/api/v1/projects", h.projectsV1)
	mux.HandleFunc("/api/projects/", h.projectByID)
	mux.HandleFunc("/api/v1/projects/", h.projectByIDV1)
	mux.HandleFunc("/api/v1/team/invite", h.teamInviteV1)
	mux.HandleFunc("/api/v1/team/members", h.teamMembersV1)
	mux.HandleFunc("/api/projects/notifications/stream", h.notificationsStream)
	mux.HandleFunc("/api/v1/projects/notifications/stream", h.notificationsStreamV1)
	return mux
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "project-service", "status": "ok"})
}

func (h handler) projects(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, h.listProjects(r, orgID))
	case http.MethodPost:
		// Enforce project limit
		maxProjects := h.store.GetOrgMaxProjects(orgID)
		currentCount := h.store.CountByOrg(orgID)
		if currentCount >= maxProjects {
			http.Error(w, "project limit reached for your plan", http.StatusForbidden)
			return
		}
		request, ok := decodeUpsert(w, r)
		if !ok {
			return
		}
		request.OrgID = orgID
		request.OwnerID = r.Header.Get("X-User-ID")
		created := h.store.Upsert("", request)
		respondJSON(w, http.StatusCreated, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h handler) projectsV1(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	switch r.Method {
	case http.MethodGet:
		respondV1Data(w, http.StatusOK, h.listProjects(r, orgID))
	case http.MethodPost:
		// Enforce project limit
		maxProjects := h.store.GetOrgMaxProjects(orgID)
		currentCount := h.store.CountByOrg(orgID)
		if currentCount >= maxProjects {
			respondV1Error(w, http.StatusForbidden, "PROJECT_LIMIT_REACHED",
				fmt.Sprintf("project limit reached (%d/%d) for your plan", currentCount, maxProjects))
			return
		}
		request, ok := decodeUpsertV1(w, r)
		if !ok {
			return
		}
		request.OrgID = orgID
		request.OwnerID = r.Header.Get("X-User-ID")
		created := h.store.Upsert("", request)
		respondV1Data(w, http.StatusCreated, created)
	default:
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h handler) projectByID(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/projects/"))
	if projectID == "" || strings.Contains(projectID, "/") {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	orgID := r.Header.Get("X-Org-ID")

	switch r.Method {
	case http.MethodGet:
		project, ok := h.store.Get(projectID, orgID)
		if !ok {
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, project)
	case http.MethodPut:
		request, ok := decodeUpsert(w, r)
		if !ok {
			return
		}
		request.OrgID = orgID
		request.OwnerID = r.Header.Get("X-User-ID")
		saved := h.store.Upsert(projectID, request)
		respondJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		h.store.Delete(projectID, orgID)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h handler) projectByIDV1(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/projects/"))
	if projectID == "" || strings.Contains(projectID, "/") {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "invalid project id")
		return
	}
	orgID := r.Header.Get("X-Org-ID")

	switch r.Method {
	case http.MethodGet:
		project, ok := h.store.Get(projectID, orgID)
		if !ok {
			respondV1Error(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "project not found")
			return
		}
		respondV1Data(w, http.StatusOK, project)
	case http.MethodPut:
		request, ok := decodeUpsertV1(w, r)
		if !ok {
			return
		}
		request.OrgID = orgID
		request.OwnerID = r.Header.Get("X-User-ID")
		saved := h.store.Upsert(projectID, request)
		respondV1Data(w, http.StatusOK, saved)
	case http.MethodDelete:
		h.store.Delete(projectID, orgID)
		respondV1Data(w, http.StatusOK, map[string]string{"projectId": projectID, "status": "deleted"})
	default:
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h handler) teamMembersV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	if projectID == "" {
		projectID = "demo"
	}

	members := h.store.ListTeam(projectID)
	respondV1Data(w, http.StatusOK, map[string]any{
		"projectId": projectID,
		"members":   members,
	})
}

func (h handler) teamInviteV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var request struct {
		ProjectID string `json:"projectId"`
		Email     string `json:"email"`
		Role      string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	projectID := strings.TrimSpace(request.ProjectID)
	if projectID == "" {
		projectID = "demo"
	}

	member := h.store.AddTeamMember(projectID, model.TeamMember{Email: request.Email, Role: request.Role})
	respondV1Data(w, http.StatusCreated, map[string]any{
		"projectId": projectID,
		"member":    member,
		"status":    "invited",
	})
}

func (h handler) notificationsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		projectID = "unknown"
	}

	// Send initial connection confirmation per SRS message format.
	_ = conn.WriteJSON(map[string]any{
		"type":  "notification",
		"runId": "",
		"payload": map[string]any{
			"title":     "workspace.connected",
			"message":   "notifications stream active",
			"projectId": projectID,
		},
	})

	// Keep-alive loop: send heartbeat every 30s to detect dead connections.
	// In production, this would be replaced by an event-driven pub/sub push.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Listen for close from client side in a goroutine.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := conn.WriteJSON(map[string]any{
				"type":  "heartbeat",
				"runId": "",
				"payload": map[string]any{
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			}); err != nil {
				return
			}
		}
	}
}

func (h handler) notificationsStreamV1(w http.ResponseWriter, r *http.Request) {
	// V1 uses same WebSocket handler.
	h.notificationsStream(w, r)
}

func (h handler) listProjects(r *http.Request, orgID string) model.ProjectListResponse {
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return h.store.List(orgID, cursor, limit)
}

func decodeUpsert(w http.ResponseWriter, r *http.Request) (model.UpsertProjectRequest, bool) {
	var request model.UpsertProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return model.UpsertProjectRequest{}, false
	}
	return request, true
}

func decodeUpsertV1(w http.ResponseWriter, r *http.Request) (model.UpsertProjectRequest, bool) {
	var request model.UpsertProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return model.UpsertProjectRequest{}, false
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
