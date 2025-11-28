package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"moktopus/project-service/internal/model"
)

type InMemoryStore struct {
	mu         sync.RWMutex
	projects   map[string]model.Project
	teamByProj map[string][]model.TeamMember
}

func NewInMemoryStore() *InMemoryStore {
	demoID := "demo"
	now := time.Now().UTC()
	return &InMemoryStore{
		projects: map[string]model.Project{
			demoID: {
				ID:          demoID,
				Name:        "Demo Infrastructure",
				Description: "Starter project for simulation and replay workflows",
				OwnerID:     "local-owner",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		teamByProj: map[string][]model.TeamMember{
			demoID: {
				{Email: "owner@moktopus.local", Role: "owner"},
			},
		},
	}
}

func (s *InMemoryStore) List(orgID, cursor string, limit int) model.ProjectListResponse {
	if limit <= 0 {
		limit = 20
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.projects))
	for id, p := range s.projects {
		if orgID != "" && p.OrgID != orgID {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start := 0
	if cursor != "" {
		for i, id := range ids {
			if id == cursor {
				start = i + 1
				break
			}
		}
	}

	projects := make([]model.Project, 0, limit)
	for i := start; i < len(ids) && len(projects) < limit; i++ {
		projects = append(projects, s.projects[ids[i]])
	}

	var nextCursor *string
	if start+len(projects) < len(ids) {
		value := ids[start+len(projects)-1]
		nextCursor = &value
	}

	return model.ProjectListResponse{Projects: projects, NextCursor: nextCursor}
}

func (s *InMemoryStore) Get(projectID, orgID string) (model.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[projectID]
	if !ok {
		return model.Project{}, false
	}
	if orgID != "" && project.OrgID != orgID {
		return model.Project{}, false
	}
	return project, ok
}

func (s *InMemoryStore) Upsert(projectID string, request model.UpsertProjectRequest) model.Project {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(projectID) == "" {
		projectID = uuid.NewString()
	}

	existing, ok := s.projects[projectID]
	if !ok {
		existing = model.Project{
			ID:        projectID,
			CreatedAt: now,
		}
	}

	if strings.TrimSpace(request.Name) == "" {
		request.Name = "Untitled Project"
	}
	if strings.TrimSpace(request.OwnerID) == "" {
		request.OwnerID = existing.OwnerID
	}
	if strings.TrimSpace(request.OwnerID) == "" {
		request.OwnerID = "local-owner"
	}

	existing.Name = strings.TrimSpace(request.Name)
	existing.Description = strings.TrimSpace(request.Description)
	existing.OwnerID = request.OwnerID
	existing.OrgID = request.OrgID
	existing.UpdatedAt = now

	s.projects[projectID] = existing
	return existing
}

func (s *InMemoryStore) Delete(projectID, orgID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if orgID != "" {
		if p, ok := s.projects[projectID]; ok && p.OrgID != orgID {
			return
		}
	}
	delete(s.projects, projectID)
	delete(s.teamByProj, projectID)
}

func (s *InMemoryStore) ListTeam(projectID string) []model.TeamMember {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := s.teamByProj[projectID]
	cloned := make([]model.TeamMember, len(members))
	copy(cloned, members)
	return cloned
}

func (s *InMemoryStore) AddTeamMember(projectID string, member model.TeamMember) model.TeamMember {
	s.mu.Lock()
	defer s.mu.Unlock()

	member.Email = strings.TrimSpace(strings.ToLower(member.Email))
	member.Role = normalizeRole(member.Role)
	if member.Email == "" {
		member.Email = "unknown@moktopus.local"
	}

	current := s.teamByProj[projectID]
	for _, existing := range current {
		if existing.Email == member.Email {
			return existing
		}
	}

	s.teamByProj[projectID] = append(current, member)
	return member
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner", "engineer", "viewer":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return "viewer"
	}
}

func (s *InMemoryStore) CountByOrg(orgID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, p := range s.projects {
		if p.OrgID == orgID {
			count++
		}
	}
	return count
}

func (s *InMemoryStore) GetOrgMaxProjects(_ string) int {
	return 5 // default trial limit for in-memory store
}
