package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"moktopus/template-service/internal/model"
	"moktopus/template-service/internal/templates"
)

type handler struct{}

func NewHandler() http.Handler {
	h := &handler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/v1/templates", h.listTemplates)
	mux.HandleFunc("/api/v1/templates/categories", h.categories)
	mux.HandleFunc("/api/v1/templates/", h.getTemplate)
	return mux
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "template-service", "status": "ok"})
}

func (h *handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	category := r.URL.Query().Get("category")
	complexity := r.URL.Query().Get("complexity")
	tag := r.URL.Query().Get("tag")

	result := make([]model.Template, 0)
	for _, tmpl := range templates.All {
		if category != "" && tmpl.Category != category {
			continue
		}
		if complexity != "" && tmpl.Complexity != complexity {
			continue
		}
		if tag != "" {
			found := false
			for _, t := range tmpl.Tags {
				if strings.EqualFold(t, tag) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, tmpl)
	}

	respondData(w, http.StatusOK, map[string]interface{}{
		"count":     len(result),
		"templates": result,
	})
}

func (h *handler) categories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	respondData(w, http.StatusOK, templates.Categories())
}

func (h *handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
	id = strings.TrimSpace(id)
	if id == "" || id == "categories" {
		respondError(w, http.StatusBadRequest, "INVALID_ID", "template ID required")
		return
	}

	for _, tmpl := range templates.All {
		if tmpl.ID == id {
			respondData(w, http.StatusOK, tmpl)
			return
		}
	}

	respondError(w, http.StatusNotFound, "NOT_FOUND", "template not found: "+id)
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
