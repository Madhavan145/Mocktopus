package store

import (
	"sync"
	"time"

	"moktopus/chaos-service/internal/model"
)

type InMemoryStore struct {
	mu        sync.RWMutex
	scenarios []model.Scenario
}

func NewInMemoryStore() *InMemoryStore {
	now := time.Now().UTC()
	return &InMemoryStore{
		scenarios: []model.Scenario{
			{ID: "steady", Name: "Steady State", FailureRate: 0.0, SlowdownMs: 0, PacketLossPct: 0.0, CreatedAt: now},
			{ID: "latency-spike", Name: "Latency Spike", FailureRate: 0.03, SlowdownMs: 120, PacketLossPct: 0.0, CreatedAt: now},
			{ID: "packet-loss", Name: "Packet Loss", FailureRate: 0.05, SlowdownMs: 40, PacketLossPct: 12.0, CreatedAt: now},
			{ID: "regional-issue", Name: "Regional Issue", FailureRate: 0.12, SlowdownMs: 250, PacketLossPct: 22.0, CreatedAt: now},
		},
	}
}

func (s *InMemoryStore) ListScenarios() []model.Scenario {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cloned := make([]model.Scenario, len(s.scenarios))
	copy(cloned, s.scenarios)
	return cloned
}

func (s *InMemoryStore) GetScenario(id string) (model.Scenario, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, scenario := range s.scenarios {
		if scenario.ID == id {
			return scenario, true
		}
	}
	return model.Scenario{}, false
}
