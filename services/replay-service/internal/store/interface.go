package store

import "moktopus/replay-service/internal/model"

// Store provides persistence for replay snapshots.
type Store interface {
	SaveSnapshots(runID string, snapshots []model.Snapshot) error
	GetSnapshots(runID string) ([]model.Snapshot, error)
}
