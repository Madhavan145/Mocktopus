package model

import "time"

type Scenario struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FailureRate   float64   `json:"failureRate"`
	SlowdownMs    int       `json:"slowdownMs"`
	PacketLossPct float64   `json:"packetLossPct"`
	CreatedAt     time.Time `json:"createdAt"`
}

type InjectRequest struct {
	ProjectID     string  `json:"projectId"`
	ScenarioID    string  `json:"scenarioId"`
	FailureRate   float64 `json:"failureRate"`
	SlowdownMs    int     `json:"slowdownMs"`
	PacketLossPct float64 `json:"packetLossPct"`
}

type InjectResponse struct {
	ProjectID   string    `json:"projectId"`
	ScenarioID  string    `json:"scenarioId"`
	AppliedAt   time.Time `json:"appliedAt"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}
