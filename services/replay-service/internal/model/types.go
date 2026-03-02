package model

type ReplayEvent struct {
	TimeMs    float64 `json:"timeMs"`
	Type      string  `json:"type"`
	RequestID int     `json:"requestId"`
	NodeID    string  `json:"nodeId,omitempty"`
	Detail    string  `json:"detail,omitempty"`
}

type ReconstructRequest struct {
	RunID          string       `json:"runId"`
	SnapshotStride int          `json:"snapshotStride"`
	Events         []ReplayEvent `json:"events"`
}

type NodeState struct {
	Touches   int `json:"touches"`
	Failures  int `json:"failures"`
	Completed int `json:"completed"`
	Queued    int `json:"queued"`
}

type Snapshot struct {
	EventIndex int                `json:"eventIndex"`
	Timestamp  float64            `json:"timestamp"`
	NodeState  map[string]NodeState `json:"nodeState"`
}

type ReconstructResponse struct {
	RunID           string     `json:"runId"`
	SnapshotStride  int        `json:"snapshotStride"`
	Snapshots       []Snapshot `json:"snapshots"`
	EventsProcessed int        `json:"eventsProcessed"`
}
