package http

import (
	"testing"

	"moktopus/replay-service/internal/model"
)

func TestBuildReplaySnapshotStride(t *testing.T) {
	req := model.ReconstructRequest{
		RunID:          "run-1",
		SnapshotStride: 2,
		Events: []model.ReplayEvent{
			{TimeMs: 1, Type: "QUEUED", NodeID: "api"},
			{TimeMs: 2, Type: "PROCESS_COMPLETE", NodeID: "api"},
			{TimeMs: 3, Type: "FAILED", NodeID: "api"},
			{TimeMs: 4, Type: "QUEUED", NodeID: "api"},
			{TimeMs: 5, Type: "REQUEST_COMPLETE", NodeID: "api"},
		},
	}

	got := buildReplay(req)
	if got.EventsProcessed != 5 {
		t.Fatalf("expected 5 events processed, got %d", got.EventsProcessed)
	}
	if got.SnapshotStride != 2 {
		t.Fatalf("expected stride=2, got %d", got.SnapshotStride)
	}
	if len(got.Snapshots) != 3 {
		t.Fatalf("expected 3 snapshots for stride=2 over 5 events, got %d", len(got.Snapshots))
	}

	if got.Snapshots[0].EventIndex != 1 || got.Snapshots[1].EventIndex != 3 || got.Snapshots[2].EventIndex != 4 {
		t.Fatalf("unexpected snapshot event indices: %+v", []int{got.Snapshots[0].EventIndex, got.Snapshots[1].EventIndex, got.Snapshots[2].EventIndex})
	}
}

func TestBuildReplayAggregatesNodeState(t *testing.T) {
	req := model.ReconstructRequest{
		RunID:          "run-2",
		SnapshotStride: 3,
		Events: []model.ReplayEvent{
			{TimeMs: 1, Type: "QUEUED", NodeID: "node-a"},
			{TimeMs: 2, Type: "PROCESS_COMPLETE", NodeID: "node-a"},
			{TimeMs: 3, Type: "FAILED", NodeID: "node-a"},
			{TimeMs: 4, Type: "REQUEST_COMPLETE", NodeID: "node-b"},
		},
	}

	got := buildReplay(req)
	if len(got.Snapshots) == 0 {
		t.Fatalf("expected snapshots")
	}

	last := got.Snapshots[len(got.Snapshots)-1]
	a := last.NodeState["node-a"]
	if a.Touches != 3 || a.Queued != 1 || a.Completed != 1 || a.Failures != 1 {
		t.Fatalf("unexpected node-a state: %+v", a)
	}
	b := last.NodeState["node-b"]
	if b.Touches != 1 || b.Completed != 1 {
		t.Fatalf("unexpected node-b state: %+v", b)
	}
}
