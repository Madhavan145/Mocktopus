package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/replay-service/internal/model"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SaveSnapshots(runID string, snapshots []model.Snapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch := "INSERT INTO replay_snapshots (run_id, timestamp_ms, node_states) VALUES "
	args := make([]any, 0, len(snapshots)*3)
	for i, snap := range snapshots {
		nodeJSON, _ := json.Marshal(snap.NodeState)
		offset := i * 3
		batch += "($" + itoa(offset+1) + ",$" + itoa(offset+2) + ",$" + itoa(offset+3) + "),"
		args = append(args, runID, int64(snap.Timestamp), nodeJSON)
	}
	batch = batch[:len(batch)-1]

	_, err := s.pool.Exec(ctx, batch, args...)
	return err
}

func (s *PostgresStore) GetSnapshots(runID string) ([]model.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT timestamp_ms, node_states
		FROM replay_snapshots
		WHERE run_id = $1
		ORDER BY timestamp_ms
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []model.Snapshot
	for rows.Next() {
		var snap model.Snapshot
		var nodeJSON []byte
		var tsMs int64
		if err := rows.Scan(&tsMs, &nodeJSON); err != nil {
			return nil, err
		}
		snap.Timestamp = float64(tsMs)
		_ = json.Unmarshal(nodeJSON, &snap.NodeState)
		snapshots = append(snapshots, snap)
	}
	return snapshots, rows.Err()
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
