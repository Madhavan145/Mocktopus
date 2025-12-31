package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"moktopus/admin-service/internal/model"
)

// Store provides admin persistence operations.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// --- Organizations ---

func (s *Store) CreateOrg(ctx context.Context, org *model.Organization) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, plan, max_seats, max_projects, max_simulations_per_day, billing_email, settings)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		org.ID, org.Name, org.Slug, org.Plan, org.MaxSeats, org.MaxProjects, org.MaxSimulationsPerDay, org.BillingEmail, org.Settings)
	return err
}

func (s *Store) GetOrg(ctx context.Context, id string) (*model.Organization, error) {
	var org model.Organization
	var suspendedAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan, max_seats, max_projects, max_simulations_per_day, billing_email, suspended_at, created_at, updated_at
		 FROM organizations WHERE id = $1`, id).
		Scan(&org.ID, &org.Name, &org.Slug, &org.Plan, &org.MaxSeats, &org.MaxProjects, &org.MaxSimulationsPerDay, &org.BillingEmail, &suspendedAt, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}
	org.SuspendedAt = suspendedAt
	return &org, nil
}

func (s *Store) ListOrgs(ctx context.Context, limit int) ([]model.Organization, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, slug, plan, max_seats, max_projects, max_simulations_per_day, billing_email, suspended_at, created_at, updated_at
		 FROM organizations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []model.Organization
	for rows.Next() {
		var org model.Organization
		var suspendedAt *time.Time
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.Plan, &org.MaxSeats, &org.MaxProjects, &org.MaxSimulationsPerDay, &org.BillingEmail, &suspendedAt, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		org.SuspendedAt = suspendedAt
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (s *Store) UpdateOrgPlan(ctx context.Context, orgID, plan string, seats, projects, sims int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE organizations SET plan=$2, max_seats=$3, max_projects=$4, max_simulations_per_day=$5, updated_at=NOW()
		 WHERE id=$1`, orgID, plan, seats, projects, sims)
	return err
}

func (s *Store) SuspendOrg(ctx context.Context, orgID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE organizations SET suspended_at=NOW(), updated_at=NOW() WHERE id=$1`, orgID)
	return err
}

func (s *Store) UnsuspendOrg(ctx context.Context, orgID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE organizations SET suspended_at=NULL, updated_at=NOW() WHERE id=$1`, orgID)
	return err
}

// --- Memberships ---

func (s *Store) AddMember(ctx context.Context, orgID, userID, role, invitedBy string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO org_memberships (org_id, user_id, role, invited_by) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (org_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		orgID, userID, role, invitedBy)
	return err
}

func (s *Store) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM org_memberships WHERE org_id=$1 AND user_id=$2`, orgID, userID)
	return err
}

func (s *Store) ListMembers(ctx context.Context, orgID string) ([]model.OrgMembership, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT m.id, m.org_id, m.user_id, m.role, u.email, u.full_name, m.joined_at
		 FROM org_memberships m JOIN users u ON u.id = m.user_id
		 WHERE m.org_id = $1 ORDER BY m.joined_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.OrgMembership
	for rows.Next() {
		var m model.OrgMembership
		if err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.Email, &m.FullName, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// --- Users (platform admin view) ---

func (s *Store) ListUsers(ctx context.Context, limit int) ([]model.UserSummary, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, full_name, role, is_platform_admin, email_verified, created_at
		 FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.UserSummary
	for rows.Next() {
		var u model.UserSummary
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsPlatformAdmin, &u.EmailVerified, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) SetPlatformAdmin(ctx context.Context, userID string, isAdmin bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET is_platform_admin=$2, updated_at=NOW() WHERE id=$1`, userID, isAdmin)
	return err
}

// --- API Keys ---

func (s *Store) CreateAPIKey(ctx context.Context, key *model.APIKey, keyHash string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO api_keys (id, org_id, name, key_hash, key_prefix, scopes, rate_limit_per_minute, expires_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		key.ID, key.OrgID, key.Name, keyHash, key.KeyPrefix, key.Scopes, key.RateLimitPerMin, key.ExpiresAt, key.CreatedBy)
	return err
}

func (s *Store) ListAPIKeys(ctx context.Context, orgID string) ([]model.APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, name, key_prefix, scopes, rate_limit_per_minute, last_used_at, expires_at, revoked_at, created_by, created_at
		 FROM api_keys WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.APIKey
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(&k.ID, &k.OrgID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.RateLimitPerMin, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt, &k.CreatedBy, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, keyID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE api_keys SET revoked_at=NOW() WHERE id=$1`, keyID)
	return err
}

func (s *Store) ValidateAPIKey(ctx context.Context, keyHash string) (*model.APIKey, error) {
	var k model.APIKey
	err := s.pool.QueryRow(ctx,
		`SELECT id, org_id, name, key_prefix, scopes, rate_limit_per_minute, expires_at, revoked_at, created_by, created_at
		 FROM api_keys WHERE key_hash=$1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())`, keyHash).
		Scan(&k.ID, &k.OrgID, &k.Name, &k.KeyPrefix, &k.Scopes, &k.RateLimitPerMin, &k.ExpiresAt, &k.RevokedAt, &k.CreatedBy, &k.CreatedAt)
	if err != nil {
		return nil, err
	}
	// Update last_used_at asynchronously (fire-and-forget)
	go func() {
		_, _ = s.pool.Exec(context.Background(), `UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, k.ID)
	}()
	return &k, nil
}

// --- Audit Log ---

func (s *Store) WriteAudit(ctx context.Context, entry *model.AuditEntry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (org_id, user_id, action, resource_type, resource_id, metadata, ip_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.OrgID, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID, entry.Metadata, entry.IPAddress)
	return err
}

func (s *Store) ListAuditLog(ctx context.Context, orgID string, limit int) ([]model.AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, org_id, user_id, action, resource_type, resource_id, metadata, ip_address, created_at
		 FROM audit_log WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Metadata, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// --- Usage Stats ---

func (s *Store) GetUsageStats(ctx context.Context, orgID string) (*model.UsageStats, error) {
	stats := &model.UsageStats{OrgID: orgID}

	// Get org limits
	var org model.Organization
	err := s.pool.QueryRow(ctx, `SELECT max_seats, max_projects, max_simulations_per_day FROM organizations WHERE id=$1`, orgID).
		Scan(&org.MaxSeats, &org.MaxProjects, &org.MaxSimulationsPerDay)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	stats.SeatsLimit = org.MaxSeats
	stats.ProjectsLimit = org.MaxProjects
	stats.SimulationsLimit = org.MaxSimulationsPerDay

	// Count active seats
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM org_memberships WHERE org_id=$1`, orgID).Scan(&stats.ActiveSeats)

	// Count active projects
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE org_id=$1 AND status='active'`, orgID).Scan(&stats.ActiveProjects)

	// Count simulations today
	_ = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM usage_records WHERE org_id=$1 AND resource_type='simulation_run' AND recorded_at >= CURRENT_DATE`,
		orgID).Scan(&stats.SimulationsToday)

	// Count LLM calls this month
	_ = s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM usage_records WHERE org_id=$1 AND resource_type='llm_call' AND recorded_at >= date_trunc('month', CURRENT_DATE)`,
		orgID).Scan(&stats.LLMCallsThisMonth)

	return stats, nil
}

func (s *Store) RecordUsage(ctx context.Context, orgID, userID, resourceType string, metadata map[string]any) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO usage_records (org_id, user_id, resource_type, metadata) VALUES ($1, $2, $3, $4)`,
		orgID, userID, resourceType, metadata)
	return err
}
