package model

type SimulationMetrics struct {
	RequestsTotal int     `json:"requestsTotal"`
	Completed     int     `json:"completed"`
	Failed        int     `json:"failed"`
	RPS           float64 `json:"rps"`
	AvgLatencyMs  float64 `json:"avgLatencyMs"`
	P50LatencyMs  float64 `json:"p50LatencyMs"`
	P95LatencyMs  float64 `json:"p95LatencyMs"`
	P99LatencyMs  float64 `json:"p99LatencyMs"`
	P999LatencyMs float64 `json:"p999LatencyMs"`
	ErrorRate     float64 `json:"errorRate"`
	QueueDepthMax int     `json:"queueDepthMax"`
	QueueWaitMs   float64 `json:"queueWaitMs"`
	Throughput    float64 `json:"throughput"`
	CPUEstimate   float64 `json:"cpuEstimate"`
	// Reliability engineering
	Goodput           float64 `json:"goodput"`
	Availability      float64 `json:"availability"`
	AvailabilityNines float64 `json:"availabilityNines"`
	MTTR_Ms           float64 `json:"mttrMs"`
	MTTF_Ms           float64 `json:"mttfMs"`
	MTBF_Ms           float64 `json:"mtbfMs"`
	ApdexScore        float64 `json:"apdexScore"`
	ApdexT            float64 `json:"apdexT"`
	SLOBurnRate       float64 `json:"sloBurnRate"`
	// Distribution
	LatencyStdDev    float64 `json:"latencyStdDev"`
	LatencyJitter    float64 `json:"latencyJitter"`
	LatencyHistogram [12]int `json:"latencyHistogram"`
	// Saturation
	MemoryEstimate      float64 `json:"memoryEstimate"`
	ConcurrencyPeak     int     `json:"concurrencyPeak"`
	CostPerRequest      float64 `json:"costPerRequest"`
	ThroughputPerDollar float64 `json:"throughputPerDollar"`
	// Enhanced
	CircuitBreakerTrips     int     `json:"circuitBreakerTrips,omitempty"`
	AutoscaleEvents         int     `json:"autoscaleEvents,omitempty"`
	ConnectionPoolExhausted int     `json:"connectionPoolExhausted,omitempty"`
	BackpressureShed        int     `json:"backpressureShed,omitempty"`
	BrownoutActivations     int     `json:"brownoutActivations,omitempty"`
	PeakLoadFactor          float64 `json:"peakLoadFactor,omitempty"`
}

type DeriveRequest struct {
	ProjectID string            `json:"projectId"`
	Metrics   SimulationMetrics `json:"metrics"`
}

type StreamRequest struct {
	ProjectID         string `json:"projectId"`
	DurationSeconds   int    `json:"durationSeconds"`
	RequestsPerSecond int    `json:"requestsPerSecond"`
}

type LiveSnapshot struct {
	ProjectID       string            `json:"projectId"`
	Tick            int               `json:"tick"`
	Progress        float64           `json:"progress"`
	Summary         SimulationMetrics `json:"summary"`
	BottleneckScore float64           `json:"bottleneckScore"`
	CapacityHint    string            `json:"capacityHint"`
}

type TimeseriesPoint struct {
	TimeIndex int     `json:"timeIndex"`
	RPS       float64 `json:"rps"`
	LatencyMs float64 `json:"latencyMs"`
	ErrorRate float64 `json:"errorRate"`
}

type DeriveResponse struct {
	ProjectID       string            `json:"projectId"`
	Summary         SimulationMetrics `json:"summary"`
	BottleneckScore float64           `json:"bottleneckScore"`
	CapacityHint    string            `json:"capacityHint"`
	Series          []TimeseriesPoint `json:"series"`
	// Derived assessments
	SLOCompliance    float64 `json:"sloCompliance"`    // % of requests meeting SLO
	ScalabilityGrade string  `json:"scalabilityGrade"` // A-F
	ReliabilityTier  string  `json:"reliabilityTier"`  // e.g. "Three Nines", "Four Nines"
	CostEfficiency   string  `json:"costEfficiency"`   // "Optimal", "Moderate", "Expensive"
	LatencyGrade     string  `json:"latencyGrade"`     // A-F based on P99/P50 ratio
}

// MetricSnapshot is a persisted time-series metric point for a simulation run.
type MetricSnapshot struct {
	RunID       string  `json:"runId"`
	TimestampMs int64   `json:"timestampMs"`
	RPS         float64 `json:"rps"`
	LatencyP50  float64 `json:"latencyP50"`
	LatencyP95  float64 `json:"latencyP95"`
	LatencyP99  float64 `json:"latencyP99"`
	ErrorRate   float64 `json:"errorRate"`
	Throughput  float64 `json:"throughput"`
	CPUEstimate float64 `json:"cpuEstimate"`
	QueueDepth  int     `json:"queueDepth"`
}

// NodeMetric is a per-node metric snapshot.
type NodeMetric struct {
	RunID          string  `json:"runId"`
	NodeID         string  `json:"nodeId"`
	NodeLabel      string  `json:"nodeLabel,omitempty"`
	NodeType       string  `json:"nodeType,omitempty"`
	TimestampMs    int64   `json:"timestampMs"`
	RequestsIn     int     `json:"requestsIn"`
	RequestsOut    int     `json:"requestsOut"`
	RequestsFailed int     `json:"requestsFailed"`
	RequestsQueued int     `json:"requestsQueued"`
	LatencyAvg     float64 `json:"latencyAvg"`
	LatencyP95     float64 `json:"latencyP95"`
	CPUUsage       float64 `json:"cpuUsage"`
	MemoryUsage    float64 `json:"memoryUsage"`
	QueueDepth     int     `json:"queueDepth"`
	ErrorRate      float64 `json:"errorRate"`
	Saturation     float64 `json:"saturation"`
}
