package store

import (
	"sync"

	"moktopus/topology-service/internal/model"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	items map[string]model.SavedTopology
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{items: map[string]model.SavedTopology{}}
}

func (s *InMemoryStore) Save(topology model.SavedTopology) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[topology.ProjectID] = topology
	return nil
}

func (s *InMemoryStore) Get(projectID string) (model.SavedTopology, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.items[projectID]
	return value, ok, nil
}
