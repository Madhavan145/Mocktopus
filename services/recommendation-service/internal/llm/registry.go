package llm

import (
	"fmt"
	"sync"
)

// Registry manages LLM providers per organization.
// It acts as a runtime store; providers can be configured via API at runtime.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider // keyed by orgId
	configs   map[string]ProviderConfig
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		configs:   make(map[string]ProviderConfig),
	}
}

// Configure sets the LLM provider for a given organization.
func (r *Registry) Configure(cfg ProviderConfig) error {
	provider, err := buildProvider(cfg)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[cfg.OrgID] = provider
	r.configs[cfg.OrgID] = cfg
	return nil
}

// Get returns the provider for the given org, or nil if none is configured.
func (r *Registry) Get(orgID string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[orgID]
}

// GetConfig returns the provider config for an org.
func (r *Registry) GetConfig(orgID string) (ProviderConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.configs[orgID]
	return cfg, ok
}

// Remove deletes the provider for an org.
func (r *Registry) Remove(orgID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, orgID)
	delete(r.configs, orgID)
}

// ListConfigured returns all org IDs that have a provider configured.
func (r *Registry) ListConfigured() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

func buildProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "openai":
		return NewOpenAI(cfg), nil
	case "anthropic":
		return NewAnthropic(cfg), nil
	case "ollama":
		return NewOllama(cfg), nil
	case "custom":
		return NewCustom(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", cfg.Type)
	}
}
