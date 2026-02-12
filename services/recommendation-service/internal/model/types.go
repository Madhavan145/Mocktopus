package model

type TopologyNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
}

type SimulationMetrics struct {
	RequestsTotal int     `json:"requestsTotal"`
	Completed     int     `json:"completed"`
	Failed        int     `json:"failed"`
	RPS           float64 `json:"rps"`
	AvgLatencyMs  float64 `json:"avgLatencyMs"`
	P95LatencyMs  float64 `json:"p95LatencyMs"`
	ErrorRate     float64 `json:"errorRate"`
	QueueDepthMax int     `json:"queueDepthMax"`
	QueueWaitMs   float64 `json:"queueWaitMs"`
	Throughput    float64 `json:"throughput"`
	CPUEstimate   float64 `json:"cpuEstimate"`
}

type DeriveRequest struct {
	ProjectID string            `json:"projectId"`
	Topology  Topology          `json:"topology"`
	Metrics   SimulationMetrics `json:"metrics"`
}

type Recommendation struct {
	ID                      string  `json:"id"`
	Problem                 string  `json:"problem"`
	Reason                  string  `json:"reason"`
	Fix                     string  `json:"fix"`
	ExpectedLatencyDeltaPct float64 `json:"expectedLatencyDeltaPct"`
	ExpectedCostDeltaPct    float64 `json:"expectedCostDeltaPct"`
	RiskScore               float64 `json:"riskScore"`
}

type DeriveResponse struct {
	ProjectID       string           `json:"projectId"`
	Confidence      float64          `json:"confidence"`
	Recommendations []Recommendation `json:"recommendations"`
	Source          string           `json:"source,omitempty"`
	TokensUsed      int              `json:"tokensUsed,omitempty"`
}
