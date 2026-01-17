package engine

import (
	"fmt"
	"strings"

	"moktopus/simulation-service/internal/model"
)

// GenerateCapacityReport analyzes simulation results and produces actionable capacity insights.
func GenerateCapacityReport(
	topology model.Topology,
	traffic model.TrafficConfig,
	metrics model.Metrics,
	nodeStats map[string]model.NodeRuntimeStats,
) *model.CapacityReport {
	report := &model.CapacityReport{
		CurrentRPS:     traffic.RequestsPerSecond,
		NodeCapacities: make([]model.NodeCapacityDetail, 0, len(topology.Nodes)),
		Warnings:       []string{},
		Recommendations: []model.CapacityRecommendation{},
	}

	var bottleneckNode string
	var bottleneckUtil float64
	var minSaturationRPS float64 = 1e9

	// Analyze each node's capacity
	for _, node := range topology.Nodes {
		profile := ResolveProfile(node)
		instanceType := configStr(node.Config, "instanceType")
		if instanceType == "" {
			instanceType = configStr(node.Config, "instanceClass")
		}
		if instanceType == "" {
			instanceType = configStr(node.Config, "nodeType")
		}

		replicas := 1
		if v, ok := getNumber(node.Config, "replicas"); ok && v > 0 {
			replicas = int(v)
		}
		if v, ok := getNumber(node.Config, "desiredCount"); ok && v > 0 {
			replicas = int(v)
		}

		// Estimate utilization: touches / (maxRPS * duration * replicas)
		stats := nodeStats[node.ID]
		durationSec := float64(traffic.DurationSeconds)
		if durationSec <= 0 {
			durationSec = 60
		}

		effectiveRPS := float64(stats.TouchCount) / durationSec
		capacity := profile.MaxRPS * float64(replicas)
		utilization := 0.0
		if capacity > 0 {
			utilization = effectiveRPS / capacity
		}

		// Latency at current load
		latencyAtLoad := LoadDependentLatency(profile.BaseLatency, int(effectiveRPS), profile.MaxRPS, replicas)

		detail := model.NodeCapacityDetail{
			NodeID:        node.ID,
			NodeType:      node.Type,
			InstanceType:  instanceType,
			MaxRPS:        capacity,
			CurrentUtil:   utilization,
			Replicas:      replicas,
			LatencyAtLoad: latencyAtLoad,
		}
		report.NodeCapacities = append(report.NodeCapacities, detail)

		// Track bottleneck (highest utilization node)
		if utilization > bottleneckUtil {
			bottleneckUtil = utilization
			bottleneckNode = node.ID
		}

		// Saturation point: at what RPS does this node hit 85% utilization
		nodeMaxSafeRPS := profile.MaxRPS * float64(replicas) * 0.85
		if nodeMaxSafeRPS < minSaturationRPS {
			minSaturationRPS = nodeMaxSafeRPS
		}

		// Generate per-node recommendations
		if utilization > 0.85 {
			report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
				NodeID:   node.ID,
				Severity: "critical",
				Category: "performance",
				Title:    fmt.Sprintf("%s is at %.0f%% capacity", node.ID, utilization*100),
				Action:   fmt.Sprintf("Scale up %s or add replicas (currently %d)", node.ID, replicas),
				Impact:   "Prevents latency degradation and potential failures under current load",
			})
		} else if utilization > 0.7 {
			report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
				NodeID:   node.ID,
				Severity: "high",
				Category: "performance",
				Title:    fmt.Sprintf("%s is at %.0f%% capacity", node.ID, utilization*100),
				Action:   "Consider adding a replica or upgrading instance type for headroom",
				Impact:   "Provides 30% headroom for traffic spikes",
			})
		} else if utilization < 0.1 && utilization > 0 {
			report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
				NodeID:   node.ID,
				Severity: "low",
				Category: "cost",
				Title:    fmt.Sprintf("%s is over-provisioned (%.0f%% utilization)", node.ID, utilization*100),
				Action:   "Consider downsizing instance type to reduce cost",
				Impact:   "Could reduce cost by 50%+ without affecting performance",
			})
		}
	}

	// Set bottleneck info
	report.Bottleneck = bottleneckNode
	if bottleneckUtil > 0.85 {
		report.BottleneckReason = fmt.Sprintf("Node at %.0f%% capacity — will saturate soon", bottleneckUtil*100)
	} else if bottleneckUtil > 0.6 {
		report.BottleneckReason = fmt.Sprintf("Highest utilization node at %.0f%%", bottleneckUtil*100)
	} else {
		report.BottleneckReason = "All nodes have comfortable headroom"
	}

	// Saturation point
	if minSaturationRPS < 1e9 {
		report.SaturationPoint = int(minSaturationRPS)
	}

	// Headroom calculation
	if report.SaturationPoint > 0 && traffic.RequestsPerSecond > 0 {
		headroomPct := (float64(report.SaturationPoint) - float64(traffic.RequestsPerSecond)) / float64(traffic.RequestsPerSecond) * 100
		if headroomPct > 0 {
			report.Headroom = fmt.Sprintf("%.0f%%", headroomPct)
		} else {
			report.Headroom = "0% (at or over capacity)"
		}
	}

	// Health score (0-100)
	report.HealthScore = computeHealthScore(topology, metrics, nodeStats, report)

	// Add architectural recommendations
	addArchitecturalRecommendations(topology, metrics, report)

	return report
}

func computeHealthScore(topology model.Topology, metrics model.Metrics, nodeStats map[string]model.NodeRuntimeStats, report *model.CapacityReport) int {
	score := 100

	// Deduct for high error rate
	if metrics.ErrorRate > 0.05 {
		score -= 30
	} else if metrics.ErrorRate > 0.01 {
		score -= 15
	} else if metrics.ErrorRate > 0.001 {
		score -= 5
	}

	// Deduct for high latency
	if metrics.P99LatencyMs > 5000 {
		score -= 20
	} else if metrics.P99LatencyMs > 1000 {
		score -= 10
	} else if metrics.P99LatencyMs > 500 {
		score -= 5
	}

	// Deduct for saturated nodes
	for _, nc := range report.NodeCapacities {
		if nc.CurrentUtil > 0.9 {
			score -= 15
		} else if nc.CurrentUtil > 0.8 {
			score -= 8
		}
	}

	// Award points for redundancy
	hasMultipleReplicas := false
	for _, nc := range report.NodeCapacities {
		if nc.Replicas > 1 {
			hasMultipleReplicas = true
			break
		}
	}
	if hasMultipleReplicas {
		score += 5 // cap at 100 handled below
	}

	// Award for monitoring presence
	for _, node := range topology.Nodes {
		if strings.ToLower(node.Type) == "monitoring" {
			score += 5
			break
		}
	}

	// Award for load balancer
	for _, node := range topology.Nodes {
		if strings.ToLower(node.Type) == "load-balancer" {
			score += 5
			break
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func addArchitecturalRecommendations(topology model.Topology, metrics model.Metrics, report *model.CapacityReport) {
	nodeTypes := map[string]bool{}
	incomingEdges := map[string]int{}
	for _, node := range topology.Nodes {
		nodeTypes[strings.ToLower(node.Type)] = true
	}
	for _, edge := range topology.Edges {
		incomingEdges[edge.Target]++
	}

	// Missing monitoring
	if !nodeTypes["monitoring"] && len(topology.Nodes) > 2 {
		report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
			Severity: "medium",
			Category: "reliability",
			Title:    "No monitoring/observability component in topology",
			Action:   "Add CloudWatch, Datadog, or Prometheus for production visibility",
			Impact:   "Critical for incident detection and performance debugging",
		})
	}

	// No load balancer but multiple services
	serviceCount := 0
	for _, node := range topology.Nodes {
		if strings.ToLower(node.Type) == "service" {
			serviceCount++
		}
	}
	if serviceCount > 1 && !nodeTypes["load-balancer"] {
		report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
			Severity: "high",
			Category: "reliability",
			Title:    "Multiple services without a load balancer",
			Action:   "Add a load balancer (ALB/NLB) for traffic distribution",
			Impact:   "Enables failover and even load distribution across services",
		})
	}

	// Single database with no Multi-AZ
	for _, node := range topology.Nodes {
		if strings.ToLower(node.Type) == "database" {
			multiAz := false
			if v, ok := node.Config["multiAz"]; ok {
				if b, ok := v.(bool); ok && b {
					multiAz = true
				}
			}
			if !multiAz {
				report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
					NodeID:   node.ID,
					Severity: "high",
					Category: "reliability",
					Title:    fmt.Sprintf("Database '%s' has no Multi-AZ failover", node.ID),
					Action:   "Enable Multi-AZ deployment for automatic failover",
					Impact:   "Eliminates ~4h/year downtime from single-AZ failures",
				})
			}
		}
	}

	// High error rate
	if metrics.ErrorRate > 0.01 {
		report.Recommendations = append(report.Recommendations, model.CapacityRecommendation{
			Severity: "critical",
			Category: "reliability",
			Title:    fmt.Sprintf("Error rate is %.1f%% — exceeds 1%% threshold", metrics.ErrorRate*100),
			Action:   "Investigate bottleneck node; add circuit breaker or increase capacity",
			Impact:   "Reduces user-facing errors and improves SLO compliance",
		})
	}
}
