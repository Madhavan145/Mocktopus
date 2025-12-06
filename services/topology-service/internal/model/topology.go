package model

import "time"

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Node struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Config   map[string]any `json:"config"`
	Position *Position      `json:"position,omitempty"`
}

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings,omitempty"`
}

type SavedTopology struct {
	ProjectID string    `json:"projectId"`
	Graph     Graph     `json:"graph"`
	CTM       Graph     `json:"ctm"`
	UpdatedAt time.Time `json:"updatedAt"`
}
