package engine

import (
	"reflect"
	"testing"

	"moktopus/simulation-service/internal/model"
)

func TestRunWithSeedIsDeterministic(t *testing.T) {
	eng := New()
	req := sampleRunRequest(77)

	first, err := eng.Run(req)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	second, err := eng.Run(req)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	// Compare core deterministic metrics (avoids nil vs empty slice issues in nested fields)
	if first.Metrics.RequestsTotal != second.Metrics.RequestsTotal ||
		first.Metrics.Completed != second.Metrics.Completed ||
		first.Metrics.Failed != second.Metrics.Failed ||
		first.Metrics.AvgLatencyMs != second.Metrics.AvgLatencyMs ||
		first.Metrics.P50LatencyMs != second.Metrics.P50LatencyMs ||
		first.Metrics.P95LatencyMs != second.Metrics.P95LatencyMs ||
		first.Metrics.P99LatencyMs != second.Metrics.P99LatencyMs ||
		first.Metrics.ErrorRate != second.Metrics.ErrorRate {
		t.Fatalf("expected deterministic metrics for same seeded request")
	}
	if !reflect.DeepEqual(first.Events, second.Events) {
		t.Fatalf("expected deterministic events for same seeded request")
	}
	if first.Metadata != second.Metadata {
		t.Fatalf("expected deterministic metadata for same seeded request")
	}
	if !first.Metadata.Deterministic {
		t.Fatalf("expected deterministic metadata=true")
	}
	if first.Metadata.SeedUsed != 77 {
		t.Fatalf("expected seedUsed=77, got %d", first.Metadata.SeedUsed)
	}
	if first.Metadata.EventTaxonomyVersion != "v1" {
		t.Fatalf("expected taxonomy version v1, got %q", first.Metadata.EventTaxonomyVersion)
	}
}

func TestRunWithZeroSeedMarksNondeterministic(t *testing.T) {
	eng := New()
	req := sampleRunRequest(0)
	req.Traffic.Randomness.Enabled = true

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metadata.Deterministic {
		t.Fatalf("expected deterministic=false when seed is auto-generated")
	}
	if result.Metadata.SeedUsed == 0 {
		t.Fatalf("expected non-zero seedUsed when seed is auto-generated")
	}
}

func TestEventTimelineAndTaxonomy(t *testing.T) {
	eng := New()
	req := sampleRunRequest(11)

	result, err := eng.Run(req)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(result.Events) == 0 {
		t.Fatalf("expected at least one event")
	}

	allowed := map[string]bool{
		"REQUEST_RECEIVED": true,
		"REQUEST_ROUTED":   true,
		"PROCESS_START":    true,
		"PROCESS_COMPLETE": true,
		"QUEUED":           true,
		"FAILED":           true,
		"REQUEST_FAILED":   true,
		"RETRY":            true,
		"REQUEST_COMPLETE": true,
	}

	lastTime := -1.0
	for _, ev := range result.Events {
		if ev.TimeMs < lastTime {
			t.Fatalf("event timeline not monotonic: %v < %v", ev.TimeMs, lastTime)
		}
		lastTime = ev.TimeMs
		if !allowed[ev.Type] {
			t.Fatalf("unexpected event type %q", ev.Type)
		}
	}

	resolved := result.Metrics.Completed + result.Metrics.Failed
	if result.Metadata.ResolvedRequestCounts != resolved {
		t.Fatalf("resolved request count mismatch: metadata=%d metrics=%d", result.Metadata.ResolvedRequestCounts, resolved)
	}
}

func sampleRunRequest(seed int64) model.RunRequest {
	return model.RunRequest{
		Topology: model.Topology{
			Nodes: []model.TopologyNode{
				{ID: "gateway", Type: "api-gateway", Config: map[string]any{"processingTimeMs": 5.0, "failureRate": 0.0}},
				{ID: "worker", Type: "service", Config: map[string]any{"processingTimeMs": 18.0, "failureRate": 0.06}},
				{ID: "queue", Type: "queue", Config: map[string]any{"processingRate": 120.0, "maxDepth": 500.0}},
				{ID: "db", Type: "database", Config: map[string]any{"latencyMs": 11.0, "failureRate": 0.02}},
			},
			Edges: []model.TopologyEdge{
				{ID: "e1", Source: "gateway", Target: "worker"},
				{ID: "e2", Source: "worker", Target: "queue"},
				{ID: "e3", Source: "queue", Target: "db"},
			},
		},
		Traffic: model.TrafficConfig{
			DurationSeconds:      4,
			RequestsPerSecond:    12,
			MaxRetries:           1,
			BaseNetworkLatencyMs: 7,
			RetryPenaltyMs:       20,
			Randomness: model.RandomnessConfig{
				Enabled:         true,
				Seed:            seed,
				ArrivalJitter:   0.2,
				LatencyJitter:   0.2,
				FailureVariance: 0.25,
			},
		},
	}
}
