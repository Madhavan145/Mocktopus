package store

import "moktopus/metrics-service/internal/model"

// Store provides persistence for derived metrics.
type Store interface {
	SaveSnapshot(runID string, snapshot model.MetricSnapshot) error
	SaveNodeMetrics(runID string, metrics []model.NodeMetric) error
	GetSnapshots(runID string) ([]model.MetricSnapshot, error)
	GetNodeMetrics(runID string) ([]model.NodeMetric, error)
}
