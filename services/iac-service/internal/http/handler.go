package http

import (
	"encoding/json"
	"io"
	"net/http"

	"moktopus/iac-service/internal/exporter"
	"moktopus/iac-service/internal/model"
	"moktopus/iac-service/internal/parser"
)

type handler struct{}

func NewHandler() http.Handler {
	h := &handler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/v1/iac/formats", h.formats)
	mux.HandleFunc("/api/v1/iac/import", h.importIaC)
	mux.HandleFunc("/api/v1/iac/export", h.exportIaC)
	return mux
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "iac-service", "status": "ok"})
}

func (h *handler) formats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	formats := []model.SupportedFormat{
		{ID: "terraform", Name: "Terraform (HCL)", Description: "HashiCorp Configuration Language - import from .tf files", Import: true, Export: true},
		{ID: "cloudformation", Name: "AWS CloudFormation", Description: "AWS CloudFormation JSON/YAML templates", Import: true, Export: false},
		{ID: "docker-compose", Name: "Docker Compose", Description: "Docker Compose YAML service definitions", Import: true, Export: false},
	}
	respondData(w, http.StatusOK, formats)
}

func (h *handler) importIaC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		respondError(w, http.StatusBadRequest, "READ_ERROR", "failed to read request body")
		return
	}

	var req model.ImportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON payload")
		return
	}

	if req.Content == "" {
		respondError(w, http.StatusBadRequest, "EMPTY_CONTENT", "content field is required")
		return
	}

	var resp *model.ImportResponse
	switch req.Format {
	case "terraform":
		resp, err = parser.ParseTerraform(req.Content)
	case "cloudformation":
		resp, err = parser.ParseCloudFormation(req.Content)
	case "docker-compose":
		resp, err = parser.ParseDockerCompose(req.Content)
	default:
		respondError(w, http.StatusBadRequest, "UNSUPPORTED_FORMAT", "format must be: terraform, cloudformation, or docker-compose")
		return
	}

	if err != nil {
		respondError(w, http.StatusUnprocessableEntity, "PARSE_ERROR", err.Error())
		return
	}

	respondData(w, http.StatusOK, resp)
}

func (h *handler) exportIaC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondError(w, http.StatusBadRequest, "READ_ERROR", "failed to read request body")
		return
	}

	var req model.ExportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON payload")
		return
	}

	if len(req.Graph.Nodes) == 0 {
		respondError(w, http.StatusBadRequest, "EMPTY_GRAPH", "graph must have at least one node")
		return
	}

	if req.Provider == "" {
		req.Provider = "aws"
	}
	if req.Region == "" {
		req.Region = "us-east-1"
	}

	var resp *model.ExportResponse
	switch req.Format {
	case "terraform":
		resp = exporter.ExportTerraform(req.Graph, req.Provider, req.Region)
	case "cloudformation":
		resp = exporter.ExportCloudFormation(req.Graph, req.Region)
	case "docker-compose":
		resp = exporter.ExportDockerCompose(req.Graph)
	default:
		respondError(w, http.StatusBadRequest, "UNSUPPORTED_EXPORT", "export supports: terraform, cloudformation, docker-compose")
		return
	}

	respondData(w, http.StatusOK, resp)
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
