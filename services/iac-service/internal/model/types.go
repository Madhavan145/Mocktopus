package model

// Node is the CTM node representation (same as topology-service)
type Node struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

// Edge is the CTM edge representation
type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// Graph is the canonical topology model
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// ImportRequest is the payload for IaC import
type ImportRequest struct {
	Format  string `json:"format"`  // "terraform", "cloudformation", "pulumi", "docker-compose"
	Content string `json:"content"` // raw HCL/YAML/JSON content
}

// ImportResponse returns the parsed topology
type ImportResponse struct {
	Format        string   `json:"format"`
	ResourceCount int      `json:"resourceCount"`
	Graph         Graph    `json:"graph"`
	Warnings      []string `json:"warnings,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	Region        string   `json:"region,omitempty"`
}

// ExportRequest generates IaC from topology
type ExportRequest struct {
	Format   string `json:"format"`   // "terraform", "cloudformation", "docker-compose"
	Provider string `json:"provider"` // "aws", "gcp", "azure"
	Region   string `json:"region"`
	Graph    Graph  `json:"graph"`
}

// ExportResponse returns the generated IaC code
type ExportResponse struct {
	Format  string `json:"format"`
	Content string `json:"content"`
	Files   []File `json:"files,omitempty"`
}

// File is a single output file
type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// SupportedFormat describes an IaC format
type SupportedFormat struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Import      bool   `json:"import"`
	Export      bool   `json:"export"`
}
