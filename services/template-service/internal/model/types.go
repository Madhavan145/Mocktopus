package model

// Template is a pre-built architecture pattern
type Template struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Category      string        `json:"category"`
	Tags          []string      `json:"tags"`
	Complexity    string        `json:"complexity"` // "starter", "intermediate", "advanced"
	Provider      string        `json:"provider,omitempty"`
	Graph         Graph         `json:"graph"`
	TrafficPreset TrafficPreset `json:"trafficPreset"`
}

// Graph is the CTM topology
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// TrafficPreset has recommended simulation settings for the template
type TrafficPreset struct {
	DurationSeconds   int    `json:"durationSeconds"`
	RequestsPerSecond int    `json:"requestsPerSecond"`
	Description       string `json:"description"`
}

// Category groups templates
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}
