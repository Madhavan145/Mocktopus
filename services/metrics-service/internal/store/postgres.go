package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/metrics-service/internal/model"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SaveSnapshot(runID string, snap model.MetricSnapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO metric_snapshots (run_id, timestamp_ms, rps, latency_p50, latency_p95, latency_p99,
		                              error_rate, throughput, cpu_estimate, queue_depth)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, runID, snap.TimestampMs, snap.RPS, snap.LatencyP50, snap.LatencyP95, snap.LatencyP99,
		snap.ErrorRate, snap.Throughput, snap.CPUEstimate, snap.QueueDepth)
	return err
}

func (s *PostgresStore) SaveNodeMetrics(runID string, metrics []model.NodeMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `INSERT INTO node_metrics (run_id, node_id, node_label, node_type, timestamp_ms,
	           requests_in, requests_out, requests_failed, requests_queued,
	           latency_avg, latency_p95, cpu_usage, memory_usage, queue_depth, error_rate, saturation) VALUES `
	args := make([]any, 0, len(metrics)*16)

	for i, m := range metrics {
		offset := i * 16
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d),",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7, offset+8,
			offset+9, offset+10, offset+11, offset+12, offset+13, offset+14, offset+15, offset+16)
		args = append(args, runID, m.NodeID, m.NodeLabel, m.NodeType, m.TimestampMs,
			m.RequestsIn, m.RequestsOut, m.RequestsFailed, m.RequestsQueued,
			m.LatencyAvg, m.LatencyP95, m.CPUUsage, m.MemoryUsage, m.QueueDepth, m.ErrorRate, m.Saturation)
	}
	query = query[:len(query)-1]

	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *PostgresStore) GetSnapshots(runID string) ([]model.MetricSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT timestamp_ms, rps, latency_p50, latency_p95, latency_p99,
		       error_rate, throughput, cpu_estimate, queue_depth
		FROM metric_snapshots WHERE run_id = $1 ORDER BY timestamp_ms
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []model.MetricSnapshot
	for rows.Next() {
		var s model.MetricSnapshot
		s.RunID = runID
		if err := rows.Scan(&s.TimestampMs, &s.RPS, &s.LatencyP50, &s.LatencyP95, &s.LatencyP99,
			&s.ErrorRate, &s.Throughput, &s.CPUEstimate, &s.QueueDepth); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

func (s *PostgresStore) GetNodeMetrics(runID string) ([]model.NodeMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT node_id, node_label, node_type, timestamp_ms,
		       requests_in, requests_out, requests_failed, requests_queued,
		       latency_avg, latency_p95, cpu_usage, memory_usage, queue_depth, error_rate, saturation
		FROM node_metrics WHERE run_id = $1 ORDER BY timestamp_ms
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []model.NodeMetric
	for rows.Next() {
		var m model.NodeMetric
		m.RunID = runID
		if err := rows.Scan(&m.NodeID, &m.NodeLabel, &m.NodeType, &m.TimestampMs,
			&m.RequestsIn, &m.RequestsOut, &m.RequestsFailed, &m.RequestsQueued,
			&m.LatencyAvg, &m.LatencyP95, &m.CPUUsage, &m.MemoryUsage, &m.QueueDepth, &m.ErrorRate, &m.Saturation); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}
