package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider implements Provider for local Ollama instances.
type OllamaProvider struct {
	endpoint string
	model    string
	client   *http.Client
}

func NewOllama(cfg ProviderConfig) *OllamaProvider {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) Available() bool {
	// Ollama doesn't need an API key — just needs to be reachable
	return p.endpoint != ""
}

func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	body := map[string]any{
		"model":  model,
		"stream": false,
		"system": req.SystemPrompt,
		"prompt": req.UserMessage,
		"options": map[string]any{
			"temperature": req.Temperature,
		},
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/api/generate", bytes.NewReader(raw))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Response string `json:"response"`
		Model    string `json:"model"`
		Done     bool   `json:"done"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CompletionResponse{}, err
	}

	finishReason := "incomplete"
	if result.Done {
		finishReason = "stop"
	}
	return CompletionResponse{
		Content:      result.Response,
		Model:        result.Model,
		TokensUsed:   0, // Ollama doesn't report tokens in non-streaming mode by default
		FinishReason: finishReason,
	}, nil
}
