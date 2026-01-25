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

type TrafficConfig struct {
	DurationSeconds      int              `json:"durationSeconds"`
	RequestsPerSecond    int              `json:"requestsPerSecond"`
	MaxRetries           int              `json:"maxRetries"`
	BaseNetworkLatencyMs float64          `json:"baseNetworkLatencyMs"`
	RetryPenaltyMs       float64          `json:"retryPenaltyMs"`
	Randomness           RandomnessConfig `json:"randomness"`
	// Enhanced simulation features
	MultiRegion    *MultiRegionConfig    `json:"multiRegion,omitempty"`
	ServiceMesh    *ServiceMeshConfig    `json:"serviceMesh,omitempty"`
	Autoscaling    *AutoscalingConfig    `json:"autoscaling,omitempty"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuitBreaker,omitempty"`
	// Production realism features
	ConnectionPool *ConnectionPoolConfig `json:"connectionPool,omitempty"`
	Backpressure   *BackpressureConfig   `json:"backpressure,omitempty"`
	Brownout       *BrownoutConfig       `json:"brownout,omitempty"`
}

// MultiRegionConfig enables cross-region latency modeling
type MultiRegionConfig struct {
	Enabled              bool    `json:"enabled"`
	DefaultRegion        string  `json:"defaultRegion"`
	CrossRegionLatencyMs float64 `json:"crossRegionLatencyMs,omitempty"` // override auto-calculated
}

// ServiceMeshConfig models sidecar proxy overhead
type ServiceMeshConfig struct {
	Enabled          bool    `json:"enabled"`
	SidecarLatencyMs float64 `json:"sidecarLatencyMs"` // per-hop sidecar overhead (typically 1-3ms)
	MTLSOverheadMs   float64 `json:"mtlsOverheadMs"`   // mTLS handshake (typically 0.5-2ms)
	RetryBudget      float64 `json:"retryBudget"`      // % of requests that can be retried (e.g., 0.2 = 20%)
}

// AutoscalingConfig models auto-scaling behavior
type AutoscalingConfig struct {
	Enabled            bool    `json:"enabled"`
	ScaleUpThreshold   float64 `json:"scaleUpThreshold"`   // CPU % to trigger scale up (e.g., 0.7)
	ScaleDownThreshold float64 `json:"scaleDownThreshold"` // CPU % to trigger scale down
	ScaleUpDelayMs     float64 `json:"scaleUpDelayMs"`     // time to provision new instance
	MaxReplicas        int     `json:"maxReplicas"`
	CooldownMs         float64 `json:"cooldownMs"` // cooldown between scaling events
}

// CircuitBreakerConfig models circuit breaker pattern
type CircuitBreakerConfig struct {
	Enabled          bool    `json:"enabled"`
	FailureThreshold int     `json:"failureThreshold"` // consecutive failures to trip
	RecoveryTimeMs   float64 `json:"recoveryTimeMs"`   // time before half-open
	HalfOpenRequests int     `json:"halfOpenRequests"` // requests to test in half-open
}

// ConnectionPoolConfig models database/service connection pool exhaustion
type ConnectionPoolConfig struct {
	Enabled        bool    `json:"enabled"`
	MaxConnections int     `json:"maxConnections"` // max connections in pool
	AcquireTimeout float64 `json:"acquireTimeout"` // ms to wait for connection before failing
	IdleTimeoutMs  float64 `json:"idleTimeoutMs"`  // ms before idle connection is reclaimed
	LeakRate       float64 `json:"leakRate"`       // probability of a leaked connection (0-1)
}

// BackpressureConfig models cascading load propagation between dependent nodes
type BackpressureConfig struct {
	Enabled             bool    `json:"enabled"`
	PropagationFactor   float64 `json:"propagationFactor"`   // how much upstream latency increases when downstream is slow (e.g. 0.5 = 50% of downstream delay propagates)
	ShedThreshold       float64 `json:"shedThreshold"`       // load factor (0-1) at which node starts shedding requests
	ShedRate            float64 `json:"shedRate"`            // probability of shedding when above threshold
	QueueBackpressureMs float64 `json:"queueBackpressureMs"` // additional delay per queued request propagated upstream
}

// BrownoutConfig models partial degradation under sustained high load
type BrownoutConfig struct {
	Enabled          bool     `json:"enabled"`
	LoadThreshold    float64  `json:"loadThreshold"`    // load factor (0-1) at which brownout begins
	MaxSlowdownPct   float64  `json:"maxSlowdownPct"`   // max latency increase (e.g. 300 = 3x slower at full load)
	ErrorRateAtPeak  float64  `json:"errorRateAtPeak"`  // additional error rate at full brownout (e.g. 0.1 = +10%)
	RecoveryRateMs   float64  `json:"recoveryRateMs"`   // ms to recover 1% of capacity
	DegradedFeatures []string `json:"degradedFeatures"` // optional: which features degrade first
}

type RandomnessConfig struct {
	Enabled         bool    `json:"enabled"`
	Seed            int64   `json:"seed,omitempty"`
	ArrivalJitter   float64 `json:"arrivalJitter"`
	LatencyJitter   float64 `json:"latencyJitter"`
	FailureVariance float64 `json:"failureVariance"`
}

type RunRequest struct {
	ProjectID string           `json:"projectId,omitempty"`
	Topology  Topology         `json:"topology"`
	Traffic   TrafficConfig    `json:"traffic"`
	Chaos     []ChaosInjection `json:"chaos,omitempty"`
}

// ChaosInjection defines a failure injection to apply during simulation.
// Matches chaos-service scenario output format.
type ChaosInjection struct {
	Type       string  `json:"type"`       // "node-crash", "slowdown", "packet-loss", "region-failure"
	Target     string  `json:"target"`     // node ID or "*" for all
	StartMs    float64 `json:"startMs"`    // when to inject (0 = from start)
	DurationMs float64 `json:"durationMs"` // how long (0 = permanent)
	// Type-specific params
	Multiplier    float64 `json:"multiplier,omitempty"`    // slowdown multiplier (e.g., 3.0)
	FailureRate   float64 `json:"failureRate,omitempty"`   // override failure rate during injection
	PacketLossPct float64 `json:"packetLossPct,omitempty"` // % of requests dropped
}

type SimulationEvent struct {
	TimeMs    float64 `json:"timeMs"`
	Type      string  `json:"type"`
	RequestID int     `json:"requestId"`
	NodeID    string  `json:"nodeId,omitempty"`
	Detail    string  `json:"detail,omitempty"`
}

type Metrics struct {
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
	// Reliability engineering metrics
	Goodput           float64 `json:"goodput"`           // successful req/s excluding retries
	Availability      float64 `json:"availability"`      // completed / total (0-1)
	AvailabilityNines float64 `json:"availabilityNines"` // -log10(1 - availability)
	MTTR_Ms           float64 `json:"mttrMs"`            // mean time to recovery
	MTTF_Ms           float64 `json:"mttfMs"`            // mean time to failure
	MTBF_Ms           float64 `json:"mtbfMs"`            // mean time between failures
	ApdexScore        float64 `json:"apdexScore"`        // Apdex with T=100ms
	ApdexT            float64 `json:"apdexT"`            // Apdex threshold used (ms)
	SLOBurnRate       float64 `json:"sloBurnRate"`       // burn rate against 99.9% SLO
	// Distribution metrics
	LatencyStdDev    float64 `json:"latencyStdDev"`
	LatencyJitter    float64 `json:"latencyJitter"`    // coefficient of variation
	LatencyHistogram [12]int `json:"latencyHistogram"` // buckets: 0-5,5-10,10-25,25-50,50-100,100-250,250-500,500-1000,1000-2500,2500-5000,5000-10000,10000+
	// Saturation and resource metrics
	MemoryEstimate      float64 `json:"memoryEstimate"`
	ConcurrencyPeak     int     `json:"concurrencyPeak"`
	CostPerRequest      float64 `json:"costPerRequest"`      // derived from cost estimate / completed
	ThroughputPerDollar float64 `json:"throughputPerDollar"` // inverse cost efficiency
	// Enhanced metrics
	CrossRegionLatencyMs float64            `json:"crossRegionLatencyMs,omitempty"`
	SidecarOverheadMs    float64            `json:"sidecarOverheadMs,omitempty"`
	CircuitBreakerTrips  int                `json:"circuitBreakerTrips,omitempty"`
	AutoscaleEvents      int                `json:"autoscaleEvents,omitempty"`
	RegionMetrics        map[string]float64 `json:"regionMetrics,omitempty"`
	// Production realism metrics
	ConnectionPoolExhausted int     `json:"connectionPoolExhausted,omitempty"`
	BackpressureShed        int     `json:"backpressureShed,omitempty"`
	BrownoutActivations     int     `json:"brownoutActivations,omitempty"`
	PeakLoadFactor          float64 `json:"peakLoadFactor,omitempty"`
	// Industry-standard resilience engineering metrics
	BlastRadius          *BlastRadiusAnalysis `json:"blastRadius,omitempty"`
	CascadeChain         []CascadeFailure     `json:"cascadeChain,omitempty"`
	ErrorBudget          *ErrorBudgetStatus   `json:"errorBudget,omitempty"`
	SteadyStateDeviation float64              `json:"steadyStateDeviation,omitempty"` // % deviation from steady-state baseline
	CriticalPath         []CriticalPathNode   `json:"criticalPath,omitempty"`
	ResilienceScore      float64              `json:"resilienceScore,omitempty"` // 0-100 composite resilience grade
}

type RunResponse struct {
	Metrics        Metrics           `json:"metrics"`
	Events         []SimulationEvent `json:"events"`
	Metadata       RunMetadata       `json:"metadata"`
	CapacityReport *CapacityReport   `json:"capacityReport,omitempty"`
}

// CapacityReport provides actionable capacity planning insights from a simulation run.
type CapacityReport struct {
	CurrentRPS       int                   `json:"currentRPS"`
	SaturationPoint  int                   `json:"saturationPoint"`
	Headroom         string                `json:"headroom"`
	HealthScore      int                   `json:"healthScore"`       // 0-100
	Bottleneck       string                `json:"bottleneck"`        // node ID of bottleneck
	BottleneckReason string                `json:"bottleneckReason"`
	NodeCapacities   []NodeCapacityDetail  `json:"nodeCapacities"`
	Warnings         []string              `json:"warnings,omitempty"`
	Recommendations  []CapacityRecommendation `json:"recommendations,omitempty"`
}

type NodeCapacityDetail struct {
	NodeID        string  `json:"nodeId"`
	NodeType      string  `json:"nodeType"`
	InstanceType  string  `json:"instanceType,omitempty"`
	MaxRPS        float64 `json:"maxRPS"`
	CurrentUtil   float64 `json:"currentUtilization"` // 0-1
	Replicas      int     `json:"replicas"`
	LatencyAtLoad float64 `json:"latencyAtLoadMs"`
}

type CapacityRecommendation struct {
	NodeID   string `json:"nodeId"`
	Severity string `json:"severity"` // critical, high, medium, low
	Category string `json:"category"` // performance, reliability, cost
	Title    string `json:"title"`
	Action   string `json:"action"`
	Impact   string `json:"impact"`
}

type RunMetadata struct {
	Deterministic         bool    `json:"deterministic"`
	SeedUsed              int64   `json:"seedUsed"`
	VirtualDurationMs     float64 `json:"virtualDurationMs"`
	EventTaxonomyVersion  string  `json:"eventTaxonomyVersion"`
	EventsTruncated       bool    `json:"eventsTruncated"`
	ResolvedRequestCounts int     `json:"resolvedRequestCounts"`
}

type ProgressSnapshot struct {
	Progress      float64                     `json:"progress"`
	Metrics       Metrics                     `json:"metrics"`
	CurrentEvent  string                      `json:"currentEvent"`
	CurrentTimeMs float64                     `json:"currentTimeMs"`
	NodeStats     map[string]NodeRuntimeStats `json:"nodeStats,omitempty"`
}

type NodeRuntimeStats struct {
	TouchCount     int `json:"touchCount"`
	FailureCount   int `json:"failureCount"`
	CompletedCount int `json:"completedCount"`
	QueuedCount    int `json:"queuedCount"`
}

type StreamMessage struct {
	Type     string            `json:"type"`
	Snapshot *ProgressSnapshot `json:"snapshot,omitempty"`
	Result   *RunResponse      `json:"result,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type StartRunResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
}

type RunStateResponse struct {
	RunID       string            `json:"runId"`
	Status      string            `json:"status"`
	StartedAt   string            `json:"startedAt"`
	CompletedAt string            `json:"completedAt,omitempty"`
	Snapshot    *ProgressSnapshot `json:"snapshot,omitempty"`
	Result      *RunResponse      `json:"result,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// BlastRadiusAnalysis quantifies the impact of failures on the overall system.
// Industry standard: Gremlin blast radius, Netflix failure impact scoring.
type BlastRadiusAnalysis struct {
	ImpactedNodes       int                `json:"impactedNodes"`       // nodes affected by failure propagation
	TotalNodes          int                `json:"totalNodes"`          // total nodes in topology
	ImpactPercentage    float64            `json:"impactPercentage"`    // % of system impacted (0-100)
	UserImpactEstimate  float64            `json:"userImpactEstimate"`  // estimated % of end-user requests affected
	PerNodeImpact       map[string]float64 `json:"perNodeImpact"`       // node_id → impact score (0-1)
	MostVulnerableNode  string             `json:"mostVulnerableNode"`  // node whose failure causes largest blast radius
	MostVulnerableScore float64            `json:"mostVulnerableScore"` // impact score of most vulnerable node
}

// CascadeFailure represents a single link in a cascading failure chain.
// Industry standard: Netflix cascade detection, AWS failure propagation analysis.
type CascadeFailure struct {
	SourceNode    string  `json:"sourceNode"`    // where the failure originated
	AffectedNode  string  `json:"affectedNode"`  // downstream node that failed as a result
	PropagationMs float64 `json:"propagationMs"` // time for failure to propagate
	Mechanism     string  `json:"mechanism"`     // "backpressure", "timeout", "circuit_break", "pool_exhaustion"
	Depth         int     `json:"depth"`         // cascade depth (1 = direct, 2+ = transitive)
}

// ErrorBudgetStatus tracks SLO consumption across multiple windows.
// Industry standard: Google SRE error budget methodology.
type ErrorBudgetStatus struct {
	SLOTarget         float64 `json:"sloTarget"`         // e.g., 0.999 for 99.9%
	BudgetTotal       float64 `json:"budgetTotal"`       // total error budget (1 - SLO target)
	BudgetConsumed    float64 `json:"budgetConsumed"`    // fraction of budget used (0-1)
	BudgetRemaining   float64 `json:"budgetRemaining"`   // fraction remaining (0-1)
	BurnRate1h        float64 `json:"burnRate1h"`        // extrapolated burn rate over 1h equivalent
	BurnRate6h        float64 `json:"burnRate6h"`        // extrapolated burn rate over 6h equivalent
	BurnRate24h       float64 `json:"burnRate24h"`       // extrapolated burn rate over 24h equivalent
	TimeToExhaustion  float64 `json:"timeToExhaustion"`  // minutes until budget exhausted at current rate
	AlertSeverity     string  `json:"alertSeverity"`     // "none", "warning", "critical", "page"
	WindowDurationSec int     `json:"windowDurationSec"` // simulation window in seconds
}

// CriticalPathNode represents a node on the latency-critical path.
// Industry standard: distributed tracing critical path analysis.
type CriticalPathNode struct {
	NodeID            string  `json:"nodeId"`
	LatencyContrib    float64 `json:"latencyContrib"`    // ms contributed to end-to-end latency
	ContribPercentage float64 `json:"contribPercentage"` // % of total e2e latency
	Bottleneck        bool    `json:"bottleneck"`        // true if this node contributes > 40% of latency
	Saturation        float64 `json:"saturation"`        // resource saturation at this node (0-1)
}
