package http

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"moktopus/admin-service/internal/model"
	"moktopus/admin-service/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) http.Handler {
	h := &Handler{store: s}
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", h.health)

	// Organizations
	mux.HandleFunc("/api/v1/admin/orgs", h.orgs)
	mux.HandleFunc("/api/v1/admin/orgs/", h.orgByID)
	mux.HandleFunc("/api/v1/admin/orgs/suspend", h.suspendOrg)
	mux.HandleFunc("/api/v1/admin/orgs/unsuspend", h.unsuspendOrg)

	// Members
	mux.HandleFunc("/api/v1/admin/orgs/members", h.members)
	mux.HandleFunc("/api/v1/admin/orgs/members/add", h.addMember)
	mux.HandleFunc("/api/v1/admin/orgs/members/remove", h.removeMember)

	// Users (platform admin)
	mux.HandleFunc("/api/v1/admin/users", h.users)
	mux.HandleFunc("/api/v1/admin/users/promote", h.promoteUser)

	// API Keys
	mux.HandleFunc("/api/v1/admin/api-keys", h.apiKeys)
	mux.HandleFunc("/api/v1/admin/api-keys/create", h.createAPIKey)
	mux.HandleFunc("/api/v1/admin/api-keys/revoke", h.revokeAPIKey)
	mux.HandleFunc("/api/v1/admin/api-keys/validate", h.validateAPIKey)

	// Audit Log
	mux.HandleFunc("/api/v1/admin/audit", h.auditLog)

	// Usage & Plans
	mux.HandleFunc("/api/v1/admin/usage", h.usage)
	mux.HandleFunc("/api/v1/admin/plans", h.plans)

	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "admin-service", "status": "ok"})
}

// --- Organizations ---

func (h *Handler) orgs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		orgs, err := h.store.ListOrgs(r.Context(), 100)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
			return
		}
		respondData(w, http.StatusOK, orgs)
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Name         string `json:"name"`
			Slug         string `json:"slug"`
			Plan         string `json:"plan"`
			BillingEmail string `json:"billingEmail"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid request body")
			return
		}
		if req.Name == "" || req.Slug == "" {
			respondError(w, http.StatusBadRequest, "MISSING_FIELDS", "name and slug required")
			return
		}
		plan := planByID(req.Plan)
		org := &model.Organization{
			ID:                   generateID(),
			Name:                 req.Name,
			Slug:                 strings.ToLower(strings.TrimSpace(req.Slug)),
			Plan:                 plan.ID,
			MaxSeats:             plan.MaxSeats,
			MaxProjects:          plan.MaxProjects,
			MaxSimulationsPerDay: plan.MaxSimulationsPerDay,
			BillingEmail:         req.BillingEmail,
			Settings:             map[string]any{},
		}
		if err := h.store.CreateOrg(r.Context(), org); err != nil {
			respondError(w, http.StatusConflict, "ORG_EXISTS", "organization slug already taken")
			return
		}
		respondData(w, http.StatusCreated, org)
		return
	}
	respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use GET or POST")
}

func (h *Handler) orgByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/orgs/")
	if id == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID", "org id required")
		return
	}
	if r.Method == http.MethodGet {
		org, err := h.store.GetOrg(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		respondData(w, http.StatusOK, org)
		return
	}
	if r.Method == http.MethodPut {
		var req struct {
			Plan        string `json:"plan"`
			MaxSeats    int    `json:"maxSeats"`
			MaxProjects int    `json:"maxProjects"`
			MaxSims     int    `json:"maxSimulationsPerDay"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid body")
			return
		}
		if err := h.store.UpdateOrgPlan(r.Context(), id, req.Plan, req.MaxSeats, req.MaxProjects, req.MaxSims); err != nil {
			respondError(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
			return
		}
		respondData(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}
	respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use GET or PUT")
}

func (h *Handler) suspendOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		OrgID string `json:"orgId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrgID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "orgId required")
		return
	}
	if err := h.store.SuspendOrg(r.Context(), req.OrgID); err != nil {
		respondError(w, http.StatusInternalServerError, "SUSPEND_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) unsuspendOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		OrgID string `json:"orgId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrgID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "orgId required")
		return
	}
	if err := h.store.UnsuspendOrg(r.Context(), req.OrgID); err != nil {
		respondError(w, http.StatusInternalServerError, "UNSUSPEND_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "active"})
}

// --- Members ---

func (h *Handler) members(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ORG", "orgId query param required")
		return
	}
	members, err := h.store.ListMembers(r.Context(), orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	respondData(w, http.StatusOK, members)
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		OrgID     string `json:"orgId"`
		UserID    string `json:"userId"`
		Role      string `json:"role"`
		InvitedBy string `json:"invitedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid body")
		return
	}
	if req.OrgID == "" || req.UserID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_FIELDS", "orgId and userId required")
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}
	if err := h.store.AddMember(r.Context(), req.OrgID, req.UserID, req.Role, req.InvitedBy); err != nil {
		respondError(w, http.StatusInternalServerError, "ADD_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		OrgID  string `json:"orgId"`
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrgID == "" || req.UserID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "orgId and userId required")
		return
	}
	if err := h.store.RemoveMember(r.Context(), req.OrgID, req.UserID); err != nil {
		respondError(w, http.StatusInternalServerError, "REMOVE_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "removed"})
}

// --- Users ---

func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context(), 200)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	respondData(w, http.StatusOK, users)
}

func (h *Handler) promoteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		UserID  string `json:"userId"`
		IsAdmin bool   `json:"isPlatformAdmin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "userId required")
		return
	}
	if err := h.store.SetPlatformAdmin(r.Context(), req.UserID, req.IsAdmin); err != nil {
		respondError(w, http.StatusInternalServerError, "PROMOTE_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "updated"})
}

// --- API Keys ---

func (h *Handler) apiKeys(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ORG", "orgId query param required")
		return
	}
	keys, err := h.store.ListAPIKeys(r.Context(), orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	respondData(w, http.StatusOK, keys)
}

func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		OrgID     string   `json:"orgId"`
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		CreatedBy string   `json:"createdBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrgID == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "orgId and name required")
		return
	}

	// Generate cryptographically secure API key
	rawKey := make([]byte, 32)
	if _, err := rand.Read(rawKey); err != nil {
		respondError(w, http.StatusInternalServerError, "KEY_GEN_FAILED", "failed to generate key")
		return
	}
	fullKey := "mkt_" + hex.EncodeToString(rawKey)
	keyHash := hashKey(fullKey)
	keyPrefix := fullKey[:12]

	key := &model.APIKey{
		ID:              generateID(),
		OrgID:           req.OrgID,
		Name:            req.Name,
		KeyPrefix:       keyPrefix,
		Scopes:          req.Scopes,
		RateLimitPerMin: 60,
		CreatedBy:       req.CreatedBy,
	}

	if err := h.store.CreateAPIKey(r.Context(), key, keyHash); err != nil {
		respondError(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}

	// Return the full key ONCE (never stored in plaintext)
	respondData(w, http.StatusCreated, map[string]any{
		"key":       fullKey,
		"keyPrefix": keyPrefix,
		"name":      key.Name,
		"id":        key.ID,
		"scopes":    key.Scopes,
		"warning":   "Store this key securely. It will not be shown again.",
	})
}

func (h *Handler) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		KeyID string `json:"keyId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.KeyID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "keyId required")
		return
	}
	if err := h.store.RevokeAPIKey(r.Context(), req.KeyID); err != nil {
		respondError(w, http.StatusInternalServerError, "REVOKE_FAILED", err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) validateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		respondError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "key required")
		return
	}
	keyHash := hashKey(req.Key)
	apiKey, err := h.store.ValidateAPIKey(r.Context(), keyHash)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "INVALID_KEY", "invalid or expired API key")
		return
	}
	respondData(w, http.StatusOK, apiKey)
}

// --- Audit ---

func (h *Handler) auditLog(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ORG", "orgId query param required")
		return
	}
	entries, err := h.store.ListAuditLog(r.Context(), orgID, 100)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	respondData(w, http.StatusOK, entries)
}

// --- Usage & Plans ---

func (h *Handler) usage(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("orgId")
	if orgID == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ORG", "orgId query param required")
		return
	}
	stats, err := h.store.GetUsageStats(r.Context(), orgID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	respondData(w, http.StatusOK, stats)
}

func (h *Handler) plans(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, model.AvailablePlans())
}

// --- Helpers ---

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func planByID(id string) model.Plan {
	for _, p := range model.AvailablePlans() {
		if p.ID == id {
			return p
		}
	}
	return model.AvailablePlans()[0] // default to trial
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondData(w http.ResponseWriter, status int, data any) {
	respondJSON(w, status, map[string]any{"success": true, "data": data, "error": nil})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{"success": false, "data": nil, "error": map[string]string{"code": code, "message": message}})
}
