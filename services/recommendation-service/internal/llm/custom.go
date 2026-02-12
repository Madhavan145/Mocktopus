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

// CustomProvider implements Provider for any OpenAI-compatible HTTP endpoint.
// Works with vLLM, LiteLLM, LocalAI, Azure OpenAI, or self-hosted endpoints.
type CustomProvider struct {
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

func NewCustom(cfg ProviderConfig) *CustomProvider {
	return &CustomProvider{
		endpoint: cfg.Endpoint,
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		client:   &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *CustomProvider) Name() string { return "custom" }

func (p *CustomProvider) Available() bool { return p.endpoint != "" }

func (p *CustomProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	temp := req.Temperature
	if temp == 0 {
		temp = 0.3
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	// Use OpenAI-compatible format (most common for custom endpoints)
	body := map[string]any{
		"model":       model,
		"temperature": temp,
		"max_tokens":  maxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.UserMessage},
		},
	}
	raw, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("custom endpoint returned %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message      struct{ Content string } `json:"message"`
			FinishReason string                   `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CompletionResponse{}, err
	}

	content := ""
	finishReason := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
		finishReason = result.Choices[0].FinishReason
	}
	return CompletionResponse{
		Content:      content,
		Model:        result.Model,
		TokensUsed:   result.Usage.TotalTokens,
		FinishReason: finishReason,
	}, nil
}
