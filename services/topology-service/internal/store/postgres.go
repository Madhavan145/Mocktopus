package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/topology-service/internal/model"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Save(topology model.SavedTopology) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	graphJSON, err := json.Marshal(topology.Graph)
	if err != nil {
		return err
	}
	ctmJSON, err := json.Marshal(topology.CTM)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO architectures (project_id, version, graph_json, ctm_json, created_at)
		VALUES ($1, 1, $2, $3, $4)
		ON CONFLICT (project_id, version) DO UPDATE SET
			graph_json = EXCLUDED.graph_json,
			ctm_json   = EXCLUDED.ctm_json,
			created_at = EXCLUDED.created_at
	`, topology.ProjectID, graphJSON, ctmJSON, topology.UpdatedAt)
	return err
}

func (s *PostgresStore) Get(projectID string) (model.SavedTopology, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var graphJSON, ctmJSON []byte
	var createdAt time.Time

	err := s.pool.QueryRow(ctx, `
		SELECT graph_json, ctm_json, created_at
		FROM architectures
		WHERE project_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, projectID).Scan(&graphJSON, &ctmJSON, &createdAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return model.SavedTopology{}, false, nil
		}
		return model.SavedTopology{}, false, err
	}

	var graph model.Graph
	if err := json.Unmarshal(graphJSON, &graph); err != nil {
		return model.SavedTopology{}, false, err
	}
	var ctm model.Graph
	if err := json.Unmarshal(ctmJSON, &ctm); err != nil {
		return model.SavedTopology{}, false, err
	}

	return model.SavedTopology{
		ProjectID: projectID,
		Graph:     graph,
		CTM:       ctm,
		UpdatedAt: createdAt,
	}, true, nil
}
