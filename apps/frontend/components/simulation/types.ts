export type TopologyGraphPayload = {
  nodes: Array<{
    id: string;
    type: string;
    config: Record<string, unknown>;
  }>;
  edges: Array<{
    id: string;
    source: string;
    target: string;
  }>;
};

export type SimulationMetrics = {
  requestsTotal: number;
  completed: number;
  failed: number;
  rps: number;
  avgLatencyMs: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  p999LatencyMs: number;
  errorRate: number;
  queueDepthMax: number;
  queueWaitMs: number;
  throughput: number;
  cpuEstimate: number;
  // Reliability engineering
  goodput: number;
  availability: number;
  availabilityNines: number;
  mttrMs: number;
  mttfMs: number;
  mtbfMs: number;
  apdexScore: number;
  apdexT: number;
  sloBurnRate: number;
  // Distribution
  latencyStdDev: number;
  latencyJitter: number;
  latencyHistogram: number[];
  // Saturation
  memoryEstimate: number;
  concurrencyPeak: number;
  costPerRequest: number;
  throughputPerDollar: number;
  // Enhanced
  circuitBreakerTrips?: number;
  autoscaleEvents?: number;
  connectionPoolExhausted?: number;
  backpressureShed?: number;
  brownoutActivations?: number;
  peakLoadFactor?: number;
  // Industry-standard resilience metrics
  blastRadius?: BlastRadiusAnalysis;
  cascadeChain?: CascadeFailure[];
  errorBudget?: ErrorBudgetStatus;
  steadyStateDeviation?: number;
  criticalPath?: CriticalPathNode[];
  resilienceScore?: number;
};

export type BlastRadiusAnalysis = {
  impactedNodes: number;
  totalNodes: number;
  impactPercentage: number;
  userImpactEstimate: number;
  perNodeImpact: Record<string, number>;
  mostVulnerableNode: string;
  mostVulnerableScore: number;
};

export type CascadeFailure = {
  sourceNode: string;
  affectedNode: string;
  propagationMs: number;
  mechanism: string;
  depth: number;
};

export type ErrorBudgetStatus = {
  sloTarget: number;
  budgetTotal: number;
  budgetConsumed: number;
  budgetRemaining: number;
  burnRate1h: number;
  burnRate6h: number;
  burnRate24h: number;
  timeToExhaustion: number;
  alertSeverity: string;
  windowDurationSec: number;
};

export type CriticalPathNode = {
  nodeId: string;
  latencyContrib: number;
  contribPercentage: number;
  bottleneck: boolean;
  saturation: number;
};

export type SimulationEvent = {
  timeMs: number;
  type: string;
  requestId: number;
  nodeId?: string;
  detail?: string;
};

export type SimulationRunResult = {
  metrics: SimulationMetrics;
  events: SimulationEvent[];
  metadata?: {
    deterministic: boolean;
    seedUsed: number;
    virtualDurationMs: number;
    eventTaxonomyVersion: string;
    eventsTruncated: boolean;
    resolvedRequestCounts: number;
  };
};

export type SimulationRandomnessConfig = {
  enabled: boolean;
  seed: number;
  arrivalJitter: number;
  latencyJitter: number;
  failureVariance: number;
};

export type MetricsDerivedPoint = {
  timeIndex: number;
  rps: number;
  latencyMs: number;
  errorRate: number;
};

export type MetricsDerivedResponse = {
  projectId: string;
  summary: SimulationMetrics;
  bottleneckScore: number;
  capacityHint: string;
  series: MetricsDerivedPoint[];
  sloCompliance: number;
  scalabilityGrade: string;
  reliabilityTier: string;
  costEfficiency: string;
  latencyGrade: string;
};

export type Recommendation = {
  id: string;
  problem: string;
  reason: string;
  fix: string;
  expectedLatencyDeltaPct: number;
  expectedCostDeltaPct: number;
  riskScore: number;
};

export type RecommendationResponse = {
  projectId: string;
  confidence: number;
  recommendations: Recommendation[];
};

export type ReplayEvent = {
  timeMs: number;
  type: string;
  requestId: number;
  nodeId?: string;
  detail?: string;
};

export type ReplayNodeState = {
  touches: number;
  failures: number;
  completed: number;
  queued: number;
};

export type ReplaySnapshot = {
  eventIndex: number;
  timestamp: number;
  nodeState: Record<string, ReplayNodeState>;
};

export type ReplayReconstructResponse = {
  runId: string;
  snapshotStride: number;
  snapshots: ReplaySnapshot[];
  eventsProcessed: number;
};

export type LiveMetricsSnapshot = {
  progress: number;
  metrics: SimulationMetrics;
  currentEvent: string;
  currentTimeMs: number;
  nodeStats?: Record<string, NodeRuntimeStats>;
};

export type NodeRuntimeStats = {
  touchCount: number;
  failureCount: number;
  completedCount: number;
  queuedCount: number;
};

export type CostLineItem = {
  resourceType: string;
  count: number;
  unitMonthly: number;
  subtotal: number;
};

export type CostEstimateResponse = {
  projectId: string;
  currency: string;
  total: number;
  items: CostLineItem[];
};

export type SimulationRunRecord = {
  runId?: string;
  run: SimulationRunResult;
  derived: MetricsDerivedResponse | null;
  recommendation: RecommendationResponse | null;
  cost: CostEstimateResponse | null;
  capturedAt: string;
};

export type StartRunResponse = {
  runId: string;
  status: string;
};

export type RunStateResponse = {
  runId: string;
  status: 'running' | 'completed' | 'failed';
  startedAt: string;
  completedAt?: string;
  snapshot?: LiveMetricsSnapshot;
  result?: SimulationRunResult;
  error?: string;
};

export type TopologyResponse = {
  projectId: string;
  graph: TopologyGraphPayload;
  ctm: TopologyGraphPayload;
  updatedAt: string;
};
