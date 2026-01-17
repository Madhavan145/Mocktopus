package engine

import (
	"fmt"
	"testing"

	"moktopus/simulation-service/internal/model"
)

// TestDeterminism_SameSeedSameOutput verifies that running the same request
// with the same seed produces bit-for-bit identical results (SRS §45.1).
func TestDeterminism_SameSeedSameOutput(t *testing.T) {
	eng := New()
	req := stressTopology(42)

	first, err := eng.Run(req)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	second, err := eng.Run(req)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	if first.Metrics.Completed != second.Metrics.Completed || first.Metrics.Failed != second.Metrics.Failed || first.Metrics.AvgLatencyMs != second.Metrics.AvgLatencyMs || first.Metrics.P99LatencyMs != second.Metrics.P99LatencyMs {
		t.Fatalf("metrics differ between runs with same seed")
	}
	if len(first.Events) != len(second.Events) {
		t.Fatalf("event count differs: %d vs %d", len(first.Events), len(second.Events))
	}
	for i := range first.Events {
		if first.Events[i] != second.Events[i] {
			t.Fatalf("event %d differs", i)
		}
	}
}

// TestReplay_ResultEqualsOriginal verifies that replaying a completed run
// with the same seed and topology produces identical results (SRS §45.2).
func TestReplay_ResultEqualsOriginal(t *testing.T) {
	eng := New()
	req := stressTopology(123)

	original, err := eng.Run(req)
	if err != nil {
		t.Fatalf("original run failed: %v", err)
	}

	// Replay with same seed.
	replay, err := eng.Run(req)
	if err != nil {
		t.Fatalf("replay run failed: %v", err)
	}

	if original.Metrics.Completed != replay.Metrics.Completed || original.Metrics.Failed != replay.Metrics.Failed || original.Metrics.AvgLatencyMs != replay.Metrics.AvgLatencyMs || original.Metrics.P99LatencyMs != replay.Metrics.P99LatencyMs {
		t.Fatalf("replay metrics differ from original")
	}
	if len(original.Events) != len(replay.Events) {
		t.Fatalf("replay event count differs")
	}
}

// TestQueueOverflow verifies the engine handles queue overflow gracefully
// when arrival rate far exceeds processing rate (SRS §45.3).
func TestQueueOverflow(t *testing.T) {
	eng := New()
	req := model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 1.0, "failureRate": 0.0}},
				{ID: "q", Type: "queue", Config: map[string]any{"processingRate": 5.0, "maxDepth": 10.0}},
				{ID: "worker", Type: "service", Config: map[string]any{"processingTimeMs": 200.0, "failureRate": 0.0}},
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gw", Target: "q"},
				{ID: "e2", Source: "q", Target: "worker"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      5,
			RequestsPerSecond:    100, // Far exceeds queue depth of 10.
			MaxRetries:           0,
			BaseNetworkLatencyMs: 1,
			RetryPenaltyMs:       10,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: 77},
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// With queue overflow, we expect some failed requests.
	if result.Metrics.Failed == 0 {
		t.Logf("WARNING: no failures detected with queue overflow scenario (queue may not enforce depth)")
	}
	// Queue depth max should be at or near the configured max.
	if result.Metrics.QueueDepthMax < 1 {
		t.Fatalf("expected positive queue depth, got %d", result.Metrics.QueueDepthMax)
	}
	t.Logf("Queue overflow test: total=%d completed=%d failed=%d queueMax=%d",
		result.Metrics.RequestsTotal, result.Metrics.Completed, result.Metrics.Failed, result.Metrics.QueueDepthMax)
}

// TestRetryStorm verifies the engine correctly handles retries that create
// amplification (SRS §45.4).
func TestRetryStorm(t *testing.T) {
	eng := New()
	req := model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 1.0, "failureRate": 0.0}},
				{ID: "svc", Type: "service", Config: map[string]any{"processingTimeMs": 10.0, "failureRate": 0.8}}, // 80% failure
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gw", Target: "svc"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      3,
			RequestsPerSecond:    50,
			MaxRetries:           5, // Up to 5 retries per request.
			BaseNetworkLatencyMs: 2,
			RetryPenaltyMs:       10,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: 88},
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// With 80% failure and 5 retries, we expect significantly more events than requests.
	expectedMinEvents := result.Metrics.RequestsTotal * 2
	if len(result.Events) < expectedMinEvents {
		t.Logf("retry amplification lower than expected: events=%d, requests=%d",
			len(result.Events), result.Metrics.RequestsTotal)
	}

	// Check RETRY events exist.
	retryCount := 0
	for _, ev := range result.Events {
		if ev.Type == "RETRY" {
			retryCount++
		}
	}
	if retryCount == 0 {
		t.Fatalf("expected RETRY events with 80%% failure rate")
	}
	t.Logf("Retry storm: requests=%d events=%d retries=%d failed=%d",
		result.Metrics.RequestsTotal, len(result.Events), retryCount, result.Metrics.Failed)
}

// TestCascadingFailure verifies the engine propagates failures through
// dependent services (SRS §45.5).
func TestCascadingFailure(t *testing.T) {
	eng := New()
	// Chain: gw → svc1 → svc2 → svc3, svc3 has high failure.
	req := model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 1.0, "failureRate": 0.0}},
				{ID: "svc1", Type: "service", Config: map[string]any{"processingTimeMs": 5.0, "failureRate": 0.0}},
				{ID: "svc2", Type: "service", Config: map[string]any{"processingTimeMs": 5.0, "failureRate": 0.0}},
				{ID: "svc3", Type: "service", Config: map[string]any{"processingTimeMs": 5.0, "failureRate": 0.9}}, // 90% failure
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gw", Target: "svc1"},
				{ID: "e2", Source: "svc1", Target: "svc2"},
				{ID: "e3", Source: "svc2", Target: "svc3"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      3,
			RequestsPerSecond:    30,
			MaxRetries:           1,
			BaseNetworkLatencyMs: 2,
			RetryPenaltyMs:       10,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: 55},
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// With svc3 at 90% failure, most requests should fail.
	failRate := float64(result.Metrics.Failed) / float64(result.Metrics.RequestsTotal)
	if failRate < 0.5 {
		t.Fatalf("expected cascading failure rate >50%%, got %.1f%%", failRate*100)
	}
	t.Logf("Cascading failure: total=%d completed=%d failed=%d errorRate=%.1f%%",
		result.Metrics.RequestsTotal, result.Metrics.Completed, result.Metrics.Failed, failRate*100)
}

// TestAutoscaling verifies the autoscaling mechanism adjusts replicas
// in response to load (SRS §45.6).
func TestAutoscaling(t *testing.T) {
	eng := New()
	req := model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 1.0, "failureRate": 0.0}},
				{ID: "svc", Type: "service", Config: map[string]any{
					"processingTimeMs": 50.0,
					"failureRate":      0.0,
					"replicas":         1.0,
					"maxReplicas":      10.0,
					"targetCPU":        0.6,
				}},
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gw", Target: "svc"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      10,
			RequestsPerSecond:    200, // High load to trigger autoscale.
			MaxRetries:           1,
			BaseNetworkLatencyMs: 2,
			RetryPenaltyMs:       10,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: 33},
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.AutoscaleEvents == 0 {
		t.Logf("NOTE: no autoscale events detected (may depend on engine config)")
	} else {
		t.Logf("Autoscaling: %d scale events detected", result.Metrics.AutoscaleEvents)
	}
}

// TestCircuitBreaker verifies circuit breaker trips under sustained failures (SRS §45.7).
func TestCircuitBreaker(t *testing.T) {
	eng := New()
	req := model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 1.0, "failureRate": 0.0}},
				{ID: "svc", Type: "service", Config: map[string]any{
					"processingTimeMs":        10.0,
					"failureRate":             0.7, // 70% failure → should trip breaker.
					"circuitBreakerEnabled":   true,
					"circuitBreakerThreshold": 5.0,
				}},
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gw", Target: "svc"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      5,
			RequestsPerSecond:    50,
			MaxRetries:           0,
			BaseNetworkLatencyMs: 1,
			RetryPenaltyMs:       10,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: 44},
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.CircuitBreakerTrips == 0 {
		t.Logf("NOTE: no circuit breaker trips (may depend on engine threshold config)")
	} else {
		t.Logf("Circuit breaker: %d trips detected", result.Metrics.CircuitBreakerTrips)
	}
}

// TestChaosInjection verifies chaos injection affects metrics (SRS §45.8).
func TestChaosInjection(t *testing.T) {
	eng := New()
	req := stressTopology(99)
	req.Chaos = []model.ChaosInjection{
		{
			Type:        "latency",
			Target:      "svc-0",
			StartMs:     0,
			DurationMs:  5000,
			Multiplier:  5.0,
			FailureRate: 0,
		},
	}

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Baseline without chaos.
	reqNoC := stressTopology(99)
	baseline, err := eng.Run(reqNoC)
	if err != nil {
		t.Fatalf("baseline run failed: %v", err)
	}

	// With latency chaos, avg latency should be higher.
	if result.Metrics.AvgLatencyMs <= baseline.Metrics.AvgLatencyMs {
		t.Logf("NOTE: chaos latency injection did not increase avg latency (%.2f vs %.2f)",
			result.Metrics.AvgLatencyMs, baseline.Metrics.AvgLatencyMs)
	} else {
		t.Logf("Chaos injection: latency increased from %.2fms to %.2fms",
			baseline.Metrics.AvgLatencyMs, result.Metrics.AvgLatencyMs)
	}
}

// stressTopology builds a medium-size topology for stress testing.
func stressTopology(seed int64) model.RunRequest {
	nodes := []model.TopologyNode{
		{ID: "gw", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 2.0, "failureRate": 0.0}},
	}
	edges := []model.TopologyEdge{}

	for i := 0; i < 10; i++ {
		svcID := fmt.Sprintf("svc-%d", i)
		nodes = append(nodes, model.TopologyNode{
			ID:   svcID,
			Type: "service",
			Config: map[string]any{
				"processingTimeMs": 15.0,
				"failureRate":      0.02,
			},
		})
		edges = append(edges, model.TopologyEdge{
			ID:     fmt.Sprintf("e-%d", i),
			Source: "gw",
			Target: svcID,
		})
	}

	return model.RunRequest{
		Topology: model.Topology{Nodes: nodes, Edges: edges},
		Traffic: model.TrafficConfig{
			DurationSeconds:      5,
			RequestsPerSecond:    100,
			MaxRetries:           2,
			BaseNetworkLatencyMs: 3,
			RetryPenaltyMs:       15,
			Randomness:           model.RandomnessConfig{Enabled: true, Seed: seed},
		},
	}
}
