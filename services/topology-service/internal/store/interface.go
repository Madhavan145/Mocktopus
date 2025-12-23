package store

import "moktopus/topology-service/internal/model"

// Store defines persistence operations for topologies.
type Store interface {
	Save(topology model.SavedTopology) error
	Get(projectID string) (model.SavedTopology, bool, error)
}
