package http

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"moktopus/simulation-service/internal/model"
)

type runRegistry struct {
	counter uint64
	mu      sync.RWMutex
	runs    map[string]*runState
}

type runState struct {
	id          string
	status      string
	startedAt   time.Time
	completedAt *time.Time
	snapshot    *model.ProgressSnapshot
	result      *model.RunResponse
	error       string
}

func newRunRegistry() *runRegistry {
	return &runRegistry{runs: map[string]*runState{}}
}

func (r *runRegistry) create() *runState {
	id := fmt.Sprintf("run-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&r.counter, 1))
	state := &runState{
		id:        id,
		status:    "running",
		startedAt: time.Now().UTC(),
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[id] = state
	return state
}

func (r *runRegistry) updateSnapshot(id string, snapshot model.ProgressSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.runs[id]; ok {
		state.snapshot = &snapshot
	}
}

func (r *runRegistry) complete(id string, result model.RunResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.runs[id]; ok {
		now := time.Now().UTC()
		state.status = "completed"
		state.result = &result
		state.completedAt = &now
	}
}

func (r *runRegistry) fail(id, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if state, ok := r.runs[id]; ok {
		now := time.Now().UTC()
		state.status = "failed"
		state.error = message
		state.completedAt = &now
	}
}

func (r *runRegistry) get(id string) (*runState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.runs[id]
	if !ok {
		return nil, false
	}
	copyState := *state
	return &copyState, true
}
