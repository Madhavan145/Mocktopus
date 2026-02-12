package llm

import "context"

// Provider is the interface any LLM backend must satisfy.
// Implementations exist for OpenAI, Anthropic, Ollama, and custom HTTP endpoints.
type Provider interface {
	// Name returns the provider identifier (e.g. "openai", "anthropic", "ollama", "custom").
	Name() string
	// Complete sends the system prompt + user message and returns the response text.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	// Available returns true if the provider is configured and reachable.
	Available() bool
}

// CompletionRequest is the standard input for all providers.
type CompletionRequest struct {
	SystemPrompt string  `json:"systemPrompt"`
	UserMessage  string  `json:"userMessage"`
	Model        string  `json:"model"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"maxTokens"`
}

// CompletionResponse wraps the LLM output.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	TokensUsed   int    `json:"tokensUsed"`
	FinishReason string `json:"finishReason"`
}

// ProviderConfig holds connection details for an LLM provider.
type ProviderConfig struct {
	Type     string `json:"type"` // openai | anthropic | ollama | custom
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
	OrgID    string `json:"orgId"`
}
