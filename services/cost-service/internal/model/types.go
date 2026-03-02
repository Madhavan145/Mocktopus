package model

type TopologyNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type TopologyEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

type EstimateRequest struct {
	ProjectID string   `json:"projectId"`
	Topology  Topology `json:"topology"`
	Provider  string   `json:"provider,omitempty"` // "aws", "gcp", "azure" - for refined pricing
	Region    string   `json:"region,omitempty"`
}

type CostLineItem struct {
	ResourceType string  `json:"resourceType"`
	NodeID       string  `json:"nodeId,omitempty"`
	Label        string  `json:"label,omitempty"`
	InstanceType string  `json:"instanceType,omitempty"`
	Count        int     `json:"count"`
	UnitMonthly  float64 `json:"unitMonthly"`
	Subtotal     float64 `json:"subtotal"`
	Provider     string  `json:"provider,omitempty"`
	Tier         string  `json:"tier,omitempty"`
}

type EstimateResponse struct {
	ProjectID      string         `json:"projectId"`
	Currency       string         `json:"currency"`
	Total          float64        `json:"total"`
	MonthlyTotal   float64        `json:"monthlyTotal"`
	AnnualEstimate float64        `json:"annualEstimate"`
	Items          []CostLineItem `json:"items"`
	Breakdown      CostBreakdown  `json:"breakdown"`
	Comparison     []ProviderCost `json:"comparison,omitempty"`
}

// CostBreakdown groups costs by category
type CostBreakdown struct {
	Compute  float64 `json:"compute"`
	Database float64 `json:"database"`
	Cache    float64 `json:"cache"`
	Queue    float64 `json:"queue"`
	Network  float64 `json:"network"`
	Storage  float64 `json:"storage"`
	Other    float64 `json:"other"`
}

// ProviderCost shows equivalent cost on different providers
type ProviderCost struct {
	Provider string  `json:"provider"`
	Total    float64 `json:"total"`
	Savings  float64 `json:"savings"` // % vs most expensive
}
