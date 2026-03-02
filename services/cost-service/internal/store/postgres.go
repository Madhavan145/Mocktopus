package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) SaveEstimate(est *CostRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	breakdownJSON, _ := json.Marshal(est.Breakdown)
	lineItemsJSON, _ := json.Marshal(est.LineItems)
	comparisonJSON, _ := json.Marshal(est.Comparison)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO cost_estimates (project_id, run_id, provider, region, monthly_total, annual_estimate,
		                            breakdown, line_items, comparison)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9)
	`, est.ProjectID, est.RunID, est.Provider, est.Region, est.MonthlyTotal, est.AnnualEstimate,
		breakdownJSON, lineItemsJSON, comparisonJSON)
	return err
}

func (s *PostgresStore) GetEstimates(projectID string, limit int) ([]CostRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, project_id, COALESCE(run_id, ''), provider, region,
		       monthly_total, annual_estimate, breakdown, line_items, comparison, created_at
		FROM cost_estimates
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CostRecord
	for rows.Next() {
		var r CostRecord
		var breakdownJSON, lineItemsJSON, comparisonJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.RunID, &r.Provider, &r.Region,
			&r.MonthlyTotal, &r.AnnualEstimate, &breakdownJSON, &lineItemsJSON, &comparisonJSON, &createdAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(breakdownJSON, &r.Breakdown)
		_ = json.Unmarshal(lineItemsJSON, &r.LineItems)
		_ = json.Unmarshal(comparisonJSON, &r.Comparison)
		r.CreatedAt = createdAt.Format(time.RFC3339)
		records = append(records, r)
	}
	return records, rows.Err()
}
