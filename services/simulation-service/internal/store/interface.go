package store

import "moktopus/simulation-service/internal/model"

// Store provides persistence for simulation runs.
type Store interface {
	CreateRun(run *RunRecord) error
	UpdateRun(run *RunRecord) error
	GetRun(id string) (*RunRecord, error)
	ListRuns(projectID string, limit int) ([]RunRecord, error)
	SaveEvents(runID string, events []model.SimulationEvent) error
	GetEvents(runID string, limit int) ([]model.SimulationEvent, error)
	CountTodayByOrg(orgID string) int
	GetOrgMaxSimulations(orgID string) int
}

// RunRecord is the persisted representation of a simulation run.
type RunRecord struct {
	ID              string
	ProjectID       string
	Status          string // pending, running, completed, failed
	Seed            int64
	TrafficConfig   []byte // JSON
	TopologySnapshot []byte // JSON
	Result          []byte // JSON (model.RunResponse)
	ErrorMessage    string
	DurationMs      int64
	TotalRequests   int
	TotalCompleted  int
	TotalFailed     int
	StartedAt       *string
	CompletedAt     *string
	CreatedAt       string
}
