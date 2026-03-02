package store

import "moktopus/cost-service/internal/model"

// Store provides persistence for cost estimates.
type Store interface {
	SaveEstimate(est *CostRecord) error
	GetEstimates(projectID string, limit int) ([]CostRecord, error)
}

// CostRecord is the persisted representation of a cost estimate.
type CostRecord struct {
	ID             string
	ProjectID      string
	RunID          string
	Provider       string
	Region         string
	MonthlyTotal   float64
	AnnualEstimate float64
	Breakdown      model.CostBreakdown
	LineItems      []model.CostLineItem
	Comparison     []model.ProviderCost
	CreatedAt      string
}
