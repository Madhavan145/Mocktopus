package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/simulation-service/internal/model"
)

// PostgresStore implements Store backed by PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) CreateRun(run *RunRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO simulation_runs (id, project_id, status, seed, traffic_config, topology_snapshot, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, run.ID, run.ProjectID, run.Status, run.Seed, run.TrafficConfig, run.TopologySnapshot)
	return err
}

func (s *PostgresStore) UpdateRun(run *RunRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx, `
		UPDATE simulation_runs SET
			status = $2,
			result = $3,
			error_message = $4,
			duration_ms = $5,
			total_requests = $6,
			total_completed = $7,
			total_failed = $8,
			started_at = CASE WHEN $9 != '' THEN $9::timestamptz ELSE started_at END,
			completed_at = CASE WHEN $10 != '' THEN $10::timestamptz ELSE completed_at END
		WHERE id = $1
	`, run.ID, run.Status, run.Result, run.ErrorMessage, run.DurationMs,
		run.TotalRequests, run.TotalCompleted, run.TotalFailed,
		nullableTime(run.StartedAt), nullableTime(run.CompletedAt))
	return err
}

func (s *PostgresStore) GetRun(id string) (*RunRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var r RunRecord
	var startedAt, completedAt, createdAt *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, status, seed, traffic_config, topology_snapshot,
		       result, error_message, duration_ms, total_requests, total_completed, total_failed,
		       started_at, completed_at, created_at
		FROM simulation_runs WHERE id = $1
	`, id).Scan(&r.ID, &r.ProjectID, &r.Status, &r.Seed, &r.TrafficConfig, &r.TopologySnapshot,
		&r.Result, &r.ErrorMessage, &r.DurationMs, &r.TotalRequests, &r.TotalCompleted, &r.TotalFailed,
		&startedAt, &completedAt, &createdAt)
	if err != nil {
		return nil, err
	}
	if startedAt != nil {
		s := startedAt.Format(time.RFC3339)
		r.StartedAt = &s
	}
	if completedAt != nil {
		s := completedAt.Format(time.RFC3339)
		r.CompletedAt = &s
	}
	if createdAt != nil {
		r.CreatedAt = createdAt.Format(time.RFC3339)
	}
	return &r, nil
}

func (s *PostgresStore) ListRuns(projectID string, limit int) ([]RunRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, status, seed, duration_ms, total_requests, total_completed, total_failed, created_at
		FROM simulation_runs
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []RunRecord
	for rows.Next() {
		var r RunRecord
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.Status, &r.Seed, &r.DurationMs,
			&r.TotalRequests, &r.TotalCompleted, &r.TotalFailed, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = createdAt.Format(time.RFC3339)
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (s *PostgresStore) SaveEvents(runID string, events []model.SimulationEvent) error {
	if len(events) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Batch insert in chunks of 500
	const chunkSize = 500
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize
		if end > len(events) {
			end = len(events)
		}
		chunk := events[i:end]

		query := "INSERT INTO simulation_events (run_id, event_type, node_id, request_id, timestamp_ms, payload) VALUES "
		args := make([]any, 0, len(chunk)*6)
		for j, e := range chunk {
			offset := j * 6
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d),",
				offset+1, offset+2, offset+3, offset+4, offset+5, offset+6)
			payload, _ := json.Marshal(map[string]string{"detail": e.Detail})
			args = append(args, runID, e.Type, e.NodeID, fmt.Sprintf("%d", e.RequestID),
				int64(e.TimeMs), payload)
		}
		query = query[:len(query)-1] // trim trailing comma

		if _, err := s.pool.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) GetEvents(runID string, limit int) ([]model.SimulationEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 1000
	}

	rows, err := s.pool.Query(ctx, `
		SELECT event_type, node_id, request_id, timestamp_ms
		FROM simulation_events
		WHERE run_id = $1
		ORDER BY timestamp_ms
		LIMIT $2
	`, runID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.SimulationEvent
	for rows.Next() {
		var e model.SimulationEvent
		var reqID string
		if err := rows.Scan(&e.Type, &e.NodeID, &reqID, &e.TimeMs); err != nil {
			return nil, err
		}
		fmt.Sscanf(reqID, "%d", &e.RequestID)
		events = append(events, e)
	}
	return events, rows.Err()
}

func nullableTime(t *string) string {
	if t == nil {
		return ""
	}
	return *t
}

func (s *PostgresStore) CountTodayByOrg(orgID string) int {
	if orgID == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM simulation_runs sr
		JOIN projects p ON p.id = sr.project_id
		WHERE p.org_id = $1 AND sr.created_at >= CURRENT_DATE
	`, orgID).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (s *PostgresStore) GetOrgMaxSimulations(orgID string) int {
	if orgID == "" {
		return 50 // default trial limit
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var maxSims int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max_simulations_per_day, 50) FROM organizations WHERE id = $1`,
		orgID).Scan(&maxSims)
	if err != nil {
		return 50
	}
	return maxSims
}
