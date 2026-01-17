package engine

import (
	"math"
	"strings"

	"moktopus/simulation-service/internal/model"
)

// InstanceProfile represents the performance characteristics of a cloud instance.
type InstanceProfile struct {
	VCPUs       float64 // number of virtual CPUs
	MemoryGB    float64 // RAM in GB
	NetworkGbps float64 // network bandwidth in Gbps
	MaxRPS      float64 // max requests/second this instance can handle
	BaseLatency float64 // base processing latency in ms at low load
	MaxConns    int     // max connections (for DB/cache nodes)
	IOPS        int     // storage IOPS (for DB nodes)
}

// knownInstances maps common instance types to performance profiles.
// These are derived from real AWS/GCP/Azure benchmarks.
var knownInstances = map[string]InstanceProfile{
	// AWS Compute
	"t3.micro":    {VCPUs: 2, MemoryGB: 1, NetworkGbps: 5, MaxRPS: 400, BaseLatency: 25},
	"t3.small":    {VCPUs: 2, MemoryGB: 2, NetworkGbps: 5, MaxRPS: 600, BaseLatency: 20},
	"t3.medium":   {VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 1000, BaseLatency: 15},
	"t3.large":    {VCPUs: 2, MemoryGB: 8, NetworkGbps: 5, MaxRPS: 1500, BaseLatency: 12},
	"t3.xlarge":   {VCPUs: 4, MemoryGB: 16, NetworkGbps: 5, MaxRPS: 3000, BaseLatency: 10},
	"m6i.large":   {VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, MaxRPS: 2000, BaseLatency: 10},
	"m6i.xlarge":  {VCPUs: 4, MemoryGB: 16, NetworkGbps: 12.5, MaxRPS: 4000, BaseLatency: 8},
	"m6i.2xlarge": {VCPUs: 8, MemoryGB: 32, NetworkGbps: 12.5, MaxRPS: 8000, BaseLatency: 6},
	"m7g.medium":  {VCPUs: 1, MemoryGB: 4, NetworkGbps: 12.5, MaxRPS: 1200, BaseLatency: 12},
	"m7g.large":   {VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, MaxRPS: 2400, BaseLatency: 9},
	"c6i.large":   {VCPUs: 2, MemoryGB: 4, NetworkGbps: 12.5, MaxRPS: 2500, BaseLatency: 8},
	"c6i.xlarge":  {VCPUs: 4, MemoryGB: 8, NetworkGbps: 12.5, MaxRPS: 5000, BaseLatency: 6},
	"c6i.2xlarge": {VCPUs: 8, MemoryGB: 16, NetworkGbps: 12.5, MaxRPS: 10000, BaseLatency: 5},
	"r6i.large":   {VCPUs: 2, MemoryGB: 16, NetworkGbps: 12.5, MaxRPS: 1800, BaseLatency: 10},
	"r6i.xlarge":  {VCPUs: 4, MemoryGB: 32, NetworkGbps: 12.5, MaxRPS: 3500, BaseLatency: 8},
	"t4g.micro":   {VCPUs: 2, MemoryGB: 1, NetworkGbps: 5, MaxRPS: 500, BaseLatency: 22},
	"t4g.small":   {VCPUs: 2, MemoryGB: 2, NetworkGbps: 5, MaxRPS: 750, BaseLatency: 18},
	"t4g.medium":  {VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 1200, BaseLatency: 13},
	"t4g.large":   {VCPUs: 2, MemoryGB: 8, NetworkGbps: 5, MaxRPS: 1800, BaseLatency: 10},

	// AWS Database (RDS)
	"db.t4g.micro":   {VCPUs: 2, MemoryGB: 1, NetworkGbps: 5, MaxRPS: 200, BaseLatency: 15, MaxConns: 40, IOPS: 3000},
	"db.t4g.small":   {VCPUs: 2, MemoryGB: 2, NetworkGbps: 5, MaxRPS: 350, BaseLatency: 12, MaxConns: 80, IOPS: 3000},
	"db.t4g.medium":  {VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 600, BaseLatency: 10, MaxConns: 150, IOPS: 3000},
	"db.t4g.large":   {VCPUs: 2, MemoryGB: 8, NetworkGbps: 5, MaxRPS: 1000, BaseLatency: 8, MaxConns: 300, IOPS: 3000},
	"db.r6g.large":   {VCPUs: 2, MemoryGB: 16, NetworkGbps: 10, MaxRPS: 1500, BaseLatency: 6, MaxConns: 500, IOPS: 15000},
	"db.r6g.xlarge":  {VCPUs: 4, MemoryGB: 32, NetworkGbps: 10, MaxRPS: 3000, BaseLatency: 5, MaxConns: 1000, IOPS: 15000},
	"db.r6g.2xlarge": {VCPUs: 8, MemoryGB: 64, NetworkGbps: 10, MaxRPS: 6000, BaseLatency: 4, MaxConns: 2000, IOPS: 15000},
	"db.m6i.large":   {VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, MaxRPS: 1200, BaseLatency: 7, MaxConns: 400, IOPS: 12000},
	"db.m6i.xlarge":  {VCPUs: 4, MemoryGB: 16, NetworkGbps: 12.5, MaxRPS: 2500, BaseLatency: 5, MaxConns: 800, IOPS: 12000},

	// AWS Cache (ElastiCache)
	"cache.t4g.micro":   {VCPUs: 2, MemoryGB: 0.5, NetworkGbps: 5, MaxRPS: 20000, BaseLatency: 1, MaxConns: 65000},
	"cache.t4g.small":   {VCPUs: 2, MemoryGB: 1.37, NetworkGbps: 5, MaxRPS: 40000, BaseLatency: 0.8, MaxConns: 65000},
	"cache.t4g.medium":  {VCPUs: 2, MemoryGB: 3.09, NetworkGbps: 5, MaxRPS: 60000, BaseLatency: 0.5, MaxConns: 65000},
	"cache.r6g.large":   {VCPUs: 2, MemoryGB: 13.07, NetworkGbps: 10, MaxRPS: 100000, BaseLatency: 0.3, MaxConns: 65000},
	"cache.r6g.xlarge":  {VCPUs: 4, MemoryGB: 26.32, NetworkGbps: 10, MaxRPS: 200000, BaseLatency: 0.2, MaxConns: 65000},
	"cache.r7g.large":   {VCPUs: 2, MemoryGB: 13.07, NetworkGbps: 12.5, MaxRPS: 120000, BaseLatency: 0.25, MaxConns: 65000},

	// GCP Compute
	"e2-micro":       {VCPUs: 0.25, MemoryGB: 1, NetworkGbps: 2, MaxRPS: 200, BaseLatency: 30},
	"e2-small":       {VCPUs: 0.5, MemoryGB: 2, NetworkGbps: 2, MaxRPS: 400, BaseLatency: 25},
	"e2-medium":      {VCPUs: 1, MemoryGB: 4, NetworkGbps: 2, MaxRPS: 800, BaseLatency: 18},
	"e2-standard-2":  {VCPUs: 2, MemoryGB: 8, NetworkGbps: 10, MaxRPS: 1600, BaseLatency: 12},
	"e2-standard-4":  {VCPUs: 4, MemoryGB: 16, NetworkGbps: 10, MaxRPS: 3200, BaseLatency: 8},
	"n2-standard-2":  {VCPUs: 2, MemoryGB: 8, NetworkGbps: 10, MaxRPS: 2000, BaseLatency: 10},
	"n2-standard-4":  {VCPUs: 4, MemoryGB: 16, NetworkGbps: 16, MaxRPS: 4000, BaseLatency: 7},
	"n2-standard-8":  {VCPUs: 8, MemoryGB: 32, NetworkGbps: 16, MaxRPS: 8000, BaseLatency: 5},
	"c2-standard-4":  {VCPUs: 4, MemoryGB: 16, NetworkGbps: 10, MaxRPS: 5000, BaseLatency: 6},
	"c2-standard-8":  {VCPUs: 8, MemoryGB: 32, NetworkGbps: 16, MaxRPS: 10000, BaseLatency: 4},

	// Azure Compute
	"Standard_B1s":    {VCPUs: 1, MemoryGB: 1, NetworkGbps: 1, MaxRPS: 300, BaseLatency: 30},
	"Standard_B2s":    {VCPUs: 2, MemoryGB: 4, NetworkGbps: 2, MaxRPS: 800, BaseLatency: 18},
	"Standard_D2s_v5": {VCPUs: 2, MemoryGB: 8, NetworkGbps: 12.5, MaxRPS: 1800, BaseLatency: 12},
	"Standard_D4s_v5": {VCPUs: 4, MemoryGB: 16, NetworkGbps: 12.5, MaxRPS: 3600, BaseLatency: 8},
	"Standard_D8s_v5": {VCPUs: 8, MemoryGB: 32, NetworkGbps: 12.5, MaxRPS: 7200, BaseLatency: 5},
	"Standard_F2s_v2": {VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 2500, BaseLatency: 8},
	"Standard_F4s_v2": {VCPUs: 4, MemoryGB: 8, NetworkGbps: 7, MaxRPS: 5000, BaseLatency: 6},

	// Special / serverless
	"FARGATE-0.25":  {VCPUs: 0.25, MemoryGB: 0.5, MaxRPS: 200, BaseLatency: 30},
	"FARGATE-0.5":   {VCPUs: 0.5, MemoryGB: 1, MaxRPS: 500, BaseLatency: 20},
	"FARGATE-1":     {VCPUs: 1, MemoryGB: 2, MaxRPS: 1000, BaseLatency: 15},
	"FARGATE-2":     {VCPUs: 2, MemoryGB: 4, MaxRPS: 2000, BaseLatency: 10},
	"FARGATE-4":     {VCPUs: 4, MemoryGB: 8, MaxRPS: 4000, BaseLatency: 7},
	"LAMBDA-128":    {VCPUs: 0.08, MemoryGB: 0.128, MaxRPS: 100, BaseLatency: 50},
	"LAMBDA-256":    {VCPUs: 0.17, MemoryGB: 0.256, MaxRPS: 200, BaseLatency: 35},
	"LAMBDA-512":    {VCPUs: 0.33, MemoryGB: 0.512, MaxRPS: 400, BaseLatency: 25},
	"LAMBDA-1024":   {VCPUs: 0.67, MemoryGB: 1.0, MaxRPS: 800, BaseLatency: 18},
	"LAMBDA-2048":   {VCPUs: 1.33, MemoryGB: 2.0, MaxRPS: 1500, BaseLatency: 12},
}

// ResolveProfile determines the performance profile for a topology node.
// Priority: explicit config overrides > instance type lookup > type-based defaults.
func ResolveProfile(node model.TopologyNode) InstanceProfile {
	nodeType := strings.ToLower(strings.TrimSpace(node.Type))

	// Check for explicit instanceType in config
	instanceType := configStr(node.Config, "instanceType")
	if instanceType == "" {
		instanceType = configStr(node.Config, "instanceClass")
	}
	if instanceType == "" {
		instanceType = configStr(node.Config, "nodeType")
	}

	// Look up known instance
	if instanceType != "" {
		if profile, ok := knownInstances[instanceType]; ok {
			return profile
		}
	}

	// Type-based defaults when no instance type specified
	switch nodeType {
	case "service", "worker":
		return InstanceProfile{VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 1000, BaseLatency: 20}
	case "database":
		return InstanceProfile{VCPUs: 2, MemoryGB: 8, NetworkGbps: 5, MaxRPS: 500, BaseLatency: 10, MaxConns: 150, IOPS: 3000}
	case "cache":
		return InstanceProfile{VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 50000, BaseLatency: 1, MaxConns: 65000}
	case "queue":
		return InstanceProfile{VCPUs: 1, MemoryGB: 2, NetworkGbps: 5, MaxRPS: 10000, BaseLatency: 3}
	case "load-balancer", "api-gateway":
		return InstanceProfile{VCPUs: 4, MemoryGB: 8, NetworkGbps: 25, MaxRPS: 50000, BaseLatency: 2}
	case "cdn":
		return InstanceProfile{VCPUs: 0, MemoryGB: 0, NetworkGbps: 100, MaxRPS: 1000000, BaseLatency: 1}
	case "storage":
		return InstanceProfile{VCPUs: 0, MemoryGB: 0, NetworkGbps: 10, MaxRPS: 5000, BaseLatency: 10}
	case "serverless", "lambda", "function":
		return InstanceProfile{VCPUs: 0.5, MemoryGB: 0.512, MaxRPS: 500, BaseLatency: 25}
	case "stream":
		return InstanceProfile{VCPUs: 2, MemoryGB: 4, NetworkGbps: 10, MaxRPS: 20000, BaseLatency: 5}
	case "notification":
		return InstanceProfile{VCPUs: 0, MemoryGB: 0, NetworkGbps: 10, MaxRPS: 100000, BaseLatency: 2}
	case "monitoring":
		return InstanceProfile{VCPUs: 2, MemoryGB: 8, NetworkGbps: 10, MaxRPS: 50000, BaseLatency: 5}
	default:
		return InstanceProfile{VCPUs: 2, MemoryGB: 4, NetworkGbps: 5, MaxRPS: 1000, BaseLatency: 20}
	}
}

// LoadDependentLatency computes effective latency based on current load.
// As utilization approaches max, latency increases exponentially (queuing theory).
func LoadDependentLatency(baseLatency float64, currentConcurrency int, maxRPS float64, replicas int) float64 {
	if maxRPS <= 0 || replicas <= 0 {
		return baseLatency
	}

	effectiveCapacity := maxRPS * float64(replicas)
	if effectiveCapacity <= 0 {
		return baseLatency
	}

	// Utilization: ratio of current load to capacity
	utilization := float64(currentConcurrency) / effectiveCapacity
	if utilization <= 0 {
		return baseLatency
	}

	// Piecewise latency degradation model:
	// - 0-60% utilization: flat (no degradation)
	// - 60-85% utilization: linear increase (1x → 2x)
	// - 85-95% utilization: quadratic increase (2x → 5x)
	// - 95-100%+: exponential spike (5x → 20x+)
	var multiplier float64
	switch {
	case utilization < 0.6:
		multiplier = 1.0
	case utilization < 0.85:
		// Linear: 1.0 at 60% → 2.0 at 85%
		progress := (utilization - 0.6) / 0.25
		multiplier = 1.0 + progress
	case utilization < 0.95:
		// Quadratic: 2.0 at 85% → 5.0 at 95%
		progress := (utilization - 0.85) / 0.10
		multiplier = 2.0 + 3.0*progress*progress
	default:
		// Exponential spike above 95%
		excess := utilization - 0.95
		multiplier = 5.0 + 15.0*math.Pow(excess*20, 2)
		if multiplier > 50.0 {
			multiplier = 50.0 // cap at 50x to avoid infinite latency
		}
	}

	return baseLatency * multiplier
}

// ComputeCPUUtilization estimates CPU usage based on active processing.
// Returns value 0.0-1.0 representing CPU saturation.
func ComputeCPUUtilization(activeRequests int, avgProcessingMs float64, vcpus float64, replicas int) float64 {
	if vcpus <= 0 || replicas <= 0 {
		return 0
	}
	// Each request consumes avgProcessingMs / 1000 seconds of CPU time.
	// Total CPU seconds needed per second = activeRequests * avgProcessingMs / 1000.
	// Available CPU seconds = vCPUs * replicas.
	cpuDemand := float64(activeRequests) * avgProcessingMs / 1000.0
	cpuAvailable := vcpus * float64(replicas)
	utilization := cpuDemand / cpuAvailable
	if utilization > 1.0 {
		return 1.0
	}
	return utilization
}

// SaturationRPS computes the RPS at which a node reaches a target utilization.
func SaturationRPS(profile InstanceProfile, replicas int, targetUtilization float64) float64 {
	if targetUtilization <= 0 {
		targetUtilization = 0.8
	}
	return profile.MaxRPS * float64(replicas) * targetUtilization
}

// InterNodeLatency calculates the network latency between two nodes based on their regions.
func InterNodeLatency(source, target model.TopologyNode, baseNetworkLatency float64) float64 {
	srcRegion := configStr(source.Config, "region")
	tgtRegion := configStr(target.Config, "region")
	srcAZ := configStr(source.Config, "availabilityZone")
	tgtAZ := configStr(target.Config, "availabilityZone")

	// Same node or no region info: use base network latency
	if srcRegion == "" || tgtRegion == "" {
		return baseNetworkLatency
	}

	// Same region
	if srcRegion == tgtRegion {
		// Same AZ: very low latency
		if srcAZ != "" && tgtAZ != "" && srcAZ == tgtAZ {
			return 0.2 // sub-ms within same AZ
		}
		// Different AZ in same region: ~1-2ms
		return 1.5
	}

	// Cross-region: use lookup table for known pairs, else estimate from region names
	latency := crossRegionLatency(srcRegion, tgtRegion)
	return latency
}

// crossRegionLatency returns estimated latency between two regions.
func crossRegionLatency(from, to string) float64 {
	key := from + "|" + to
	reverseKey := to + "|" + from

	latencies := map[string]float64{
		"us-east-1|us-west-2":    62,
		"us-east-1|eu-west-1":    85,
		"us-east-1|ap-southeast-1": 230,
		"us-west-2|eu-west-1":    140,
		"us-west-2|ap-northeast-1": 120,
		"eu-west-1|ap-southeast-1": 180,
		"eu-west-1|eu-central-1":   25,
		"us-east-1|us-east-2":     12,
		"us-east-1|ca-central-1":   20,
		"eu-west-1|eu-west-2":     10,
		"ap-southeast-1|ap-northeast-1": 75,
	}

	if l, ok := latencies[key]; ok {
		return l
	}
	if l, ok := latencies[reverseKey]; ok {
		return l
	}

	// Heuristic: estimate based on region prefixes
	fromPrefix := regionPrefix(from)
	toPrefix := regionPrefix(to)
	if fromPrefix == toPrefix {
		return 30 // same continent
	}
	return 120 // cross-continent default
}

func regionPrefix(region string) string {
	parts := strings.Split(region, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return region
}

func configStr(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
