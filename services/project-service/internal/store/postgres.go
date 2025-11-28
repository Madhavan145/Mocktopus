package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/project-service/internal/model"
)

// PostgresStore persists projects in PostgreSQL
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) List(orgID, cursor string, limit int) model.ProjectListResponse {
	// SECURITY: Never list projects without an org scope — prevents cross-tenant data leakage
	if orgID == "" {
		return model.ProjectListResponse{Projects: []model.Project{}}
	}

	if limit <= 0 {
		limit = 20
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id, name, description, COALESCE(org_id::text, '') as org_id, COALESCE(created_by, '') as owner_id, created_at, updated_at
		FROM projects WHERE status = 'active'`
	args := []any{}
	argIdx := 1

	// Filter by organization — always required
	query += fmt.Sprintf(` AND org_id = $%d`, argIdx)
	args = append(args, orgID)
	argIdx++

	if cursor != "" {
		query += fmt.Sprintf(` AND id > $%d`, argIdx)
		args = append(args, cursor)
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d`, argIdx)
	args = append(args, limit+1)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return model.ProjectListResponse{Projects: []model.Project{}}
	}
	defer rows.Close()

	projects := make([]model.Project, 0, limit)
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OrgID, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		projects = append(projects, p)
	}

	var nextCursor *string
	if len(projects) > limit {
		projects = projects[:limit]
		last := projects[len(projects)-1].ID
		nextCursor = &last
	}

	return model.ProjectListResponse{Projects: projects, NextCursor: nextCursor}
}

func (s *PostgresStore) Get(id, orgID string) (model.Project, bool) {
	// SECURITY: Require org scope for project access
	if orgID == "" {
		return model.Project{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var p model.Project
	query := `SELECT id, name, description, COALESCE(org_id::text, '') as org_id, COALESCE(created_by, '') as owner_id, created_at, updated_at
		 FROM projects WHERE id = $1 AND status = 'active' AND org_id = $2`
	args := []any{id, orgID}

	err := s.pool.QueryRow(ctx, query, args...).
		Scan(&p.ID, &p.Name, &p.Description, &p.OrgID, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return model.Project{}, false
	}
	return p, true
}

func (s *PostgresStore) Upsert(id string, req model.UpsertProjectRequest) model.Project {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	var p model.Project

	if id == "" {
		id = generateID()
	}

	// SECURITY: Require org_id — never create/update a project without tenant scoping
	if req.OrgID == "" {
		return model.Project{ID: id, Name: req.Name, Description: req.Description, OrgID: req.OrgID, CreatedAt: now, UpdatedAt: now}
	}

	// Try UPDATE first — scoped by org_id to prevent cross-tenant writes
	tag, err := s.pool.Exec(ctx,
		`UPDATE projects SET name = $1, description = $2, updated_at = $3
		 WHERE id = $4 AND org_id = $5 AND status = 'active'`,
		req.Name, req.Description, now, id, req.OrgID)
	if err == nil && tag.RowsAffected() > 0 {
		// Successfully updated — fetch the full record
		_ = s.pool.QueryRow(ctx,
			`SELECT id, name, description, COALESCE(org_id::text, ''), COALESCE(created_by, ''), created_at, updated_at
			 FROM projects WHERE id = $1 AND org_id = $2`,
			id, req.OrgID).
			Scan(&p.ID, &p.Name, &p.Description, &p.OrgID, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt)
		return p
	}

	// INSERT — new project for this org
	err = s.pool.QueryRow(ctx,
		`INSERT INTO projects (id, name, description, org_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4::uuid, NULLIF($5, ''), $6, $6)
		 ON CONFLICT (id) DO NOTHING
		 RETURNING id, name, description, COALESCE(org_id::text, ''), COALESCE(created_by, ''), created_at, updated_at`,
		id, req.Name, req.Description, req.OrgID, nilIfEmpty(req.OwnerID), now).
		Scan(&p.ID, &p.Name, &p.Description, &p.OrgID, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		// Conflict on id with different org — reject silently (cross-tenant attempt)
		return model.Project{ID: id, Name: req.Name, Description: req.Description, OrgID: req.OrgID, CreatedAt: now, UpdatedAt: now}
	}

	return p
}

func (s *PostgresStore) Delete(id, orgID string) {
	// SECURITY: Require org scope
	if orgID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `UPDATE projects SET status = 'deleted', updated_at = NOW() WHERE id = $1 AND org_id = $2`
	args := []any{id, orgID}
	_, _ = s.pool.Exec(ctx, query, args...)
}

func (s *PostgresStore) ListTeam(projectID string) []model.TeamMember {
	// Team feature will use organizations.members in future
	return []model.TeamMember{
		{Email: "owner@moktopus.local", Role: "owner"},
	}
}

func (s *PostgresStore) AddTeamMember(projectID string, member model.TeamMember) model.TeamMember {
	return member
}

func (s *PostgresStore) CountByOrg(orgID string) int {
	if orgID == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM projects WHERE org_id = $1 AND status = 'active'`,
		orgID).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (s *PostgresStore) GetOrgMaxProjects(orgID string) int {
	if orgID == "" {
		return 5 // default trial limit
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var maxProjects int
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(max_projects, 5) FROM organizations WHERE id = $1`,
		orgID).Scan(&maxProjects)
	if err != nil {
		return 5 // default trial limit
	}
	return maxProjects
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
