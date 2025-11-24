package db

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	Role            string
	OrgID           string
	IsPlatformAdmin bool
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

func (r Repository) CreateUser(ctx context.Context, email, passwordHash, role string) (User, error) {
	id := uuid.NewString()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role) VALUES ($1, $2, $3, $4)`,
		id, email, passwordHash, role,
	)
	if err != nil {
		return User{}, err
	}

	// Auto-create a personal organization for the user and add membership
	orgID := uuid.NewString()
	baseSlug := strings.ReplaceAll(strings.Split(email, "@")[0], ".", "-")
	orgSlug := baseSlug + "-" + id[:8]
	displayName := strings.Split(email, "@")[0]

	_, err = r.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, plan, max_seats, max_projects, max_simulations_per_day)
		 VALUES ($1, $2, $3, 'trial', 3, 5, 50)`,
		orgID, displayName+"'s Org", orgSlug,
	)
	if err != nil {
		log.Printf("[auth] org insert attempt 1 failed for user %s: %v (slug=%s)", id, err, orgSlug)
		// Slug conflict — try with full UUID suffix
		orgSlug = baseSlug + "-" + id
		_, err = r.pool.Exec(ctx,
			`INSERT INTO organizations (id, name, slug, plan, max_seats, max_projects, max_simulations_per_day)
			 VALUES ($1, $2, $3, 'trial', 3, 5, 50)`,
			orgID, displayName+"'s Org", orgSlug,
		)
		if err != nil {
			log.Printf("[auth] org insert attempt 2 failed for user %s: %v (slug=%s)", id, err, orgSlug)
			// Org creation still failed — user exists without org
			return User{ID: id, Email: email, PasswordHash: passwordHash, Role: role}, nil
		}
	}

	_, _ = r.pool.Exec(ctx,
		`INSERT INTO org_memberships (id, org_id, user_id, role) VALUES ($1, $2, $3, 'owner')`,
		uuid.NewString(), orgID, id,
	)

	return User{ID: id, Email: email, PasswordHash: passwordHash, Role: role, OrgID: orgID}, nil
}

func (r Repository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role,
		        COALESCE(u.is_platform_admin, FALSE),
		        COALESCE((SELECT om.org_id::text FROM org_memberships om WHERE om.user_id = u.id ORDER BY om.joined_at ASC LIMIT 1), '')
		 FROM users u WHERE u.email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsPlatformAdmin, &u.OrgID)
	return u, err
}

func (r Repository) GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role,
		        COALESCE(u.is_platform_admin, FALSE),
		        COALESCE((SELECT om.org_id::text FROM org_memberships om WHERE om.user_id = u.id ORDER BY om.joined_at ASC LIMIT 1), '')
		 FROM users u WHERE u.id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsPlatformAdmin, &u.OrgID)
	return u, err
}

func (r Repository) CreateSession(ctx context.Context, userID, refreshHash string, expiresAt time.Time) (string, error) {
	sessionID := uuid.NewString()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		sessionID, userID, refreshHash, expiresAt,
	)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (r Repository) ValidateSession(ctx context.Context, userID, refreshHash string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM sessions
			WHERE user_id = $1 AND refresh_token_hash = $2 AND expires_at > NOW()
		)`,
		userID, refreshHash,
	).Scan(&exists)
	return exists, err
}

func (r Repository) CreatePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	if err := r.ensurePasswordResetTable(ctx); err != nil {
		return err
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.NewString(), userID, tokenHash, expiresAt,
	)
	return err
}

func (r Repository) GetUserIDByPasswordResetToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id
		 FROM password_reset_tokens
		 WHERE token_hash = $1
		   AND used_at IS NULL
		   AND expires_at > NOW()
		 LIMIT 1`,
		tokenHash,
	).Scan(&userID)
	return userID, err
}

func (r Repository) MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

func (r Repository) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2 WHERE id = $1`,
		userID, passwordHash,
	)
	return err
}

func (r Repository) ensurePasswordResetTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
	`)
	return err
}

// ─── Organization Management ────────────────────────────────────────────────

type Org struct {
	ID                   string
	Name                 string
	Slug                 string
	Plan                 string
	MaxSeats             int
	MaxProjects          int
	MaxSimulationsPerDay int
	SuspendedAt          *time.Time
	CreatedAt            time.Time
}

type OrgMember struct {
	UserID   string
	Email    string
	Role     string
	JoinedAt time.Time
}

type OrgInvite struct {
	ID        string
	OrgID     string
	Email     string
	Role      string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (r Repository) GetOrg(ctx context.Context, orgID string) (Org, error) {
	var o Org
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan, max_seats, max_projects, max_simulations_per_day, suspended_at, created_at
		 FROM organizations WHERE id = $1`, orgID,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.MaxSeats, &o.MaxProjects, &o.MaxSimulationsPerDay, &o.SuspendedAt, &o.CreatedAt)
	return o, err
}

func (r Repository) UpdateOrg(ctx context.Context, orgID, name string) error {
	_, err := r.pool.Exec(ctx, `UPDATE organizations SET name = $2 WHERE id = $1`, orgID, name)
	return err
}

func (r Repository) ListOrgMembers(ctx context.Context, orgID string) ([]OrgMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.id, u.email, om.role, om.joined_at
		 FROM org_memberships om
		 JOIN users u ON u.id = om.user_id
		 WHERE om.org_id = $1
		 ORDER BY om.joined_at ASC`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []OrgMember
	for rows.Next() {
		var m OrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func (r Repository) GetMemberRole(ctx context.Context, orgID, userID string) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM org_memberships WHERE org_id = $1 AND user_id = $2`, orgID, userID,
	).Scan(&role)
	return role, err
}

func (r Repository) UpdateMemberRole(ctx context.Context, orgID, userID, newRole string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE org_memberships SET role = $3 WHERE org_id = $1 AND user_id = $2`,
		orgID, userID, newRole,
	)
	return err
}

func (r Repository) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM org_memberships WHERE org_id = $1 AND user_id = $2`,
		orgID, userID,
	)
	return err
}

func (r Repository) CreateOrgInvite(ctx context.Context, orgID, email, role, token string, expiresAt time.Time) (OrgInvite, error) {
	id := uuid.NewString()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO org_invites (id, org_id, email, role, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, orgID, email, role, token, expiresAt,
	)
	if err != nil {
		return OrgInvite{}, err
	}
	return OrgInvite{ID: id, OrgID: orgID, Email: email, Role: role, Token: token, ExpiresAt: expiresAt}, nil
}

func (r Repository) ListOrgInvites(ctx context.Context, orgID string) ([]OrgInvite, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, email, role, expires_at, created_at
		 FROM org_invites
		 WHERE org_id = $1 AND accepted_at IS NULL AND expires_at > NOW()
		 ORDER BY created_at DESC`, orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []OrgInvite
	for rows.Next() {
		var inv OrgInvite
		if err := rows.Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		invites = append(invites, inv)
	}
	return invites, nil
}

func (r Repository) AcceptInvite(ctx context.Context, inviteID, userID string) error {
	// Get invite details
	var orgID, role string
	err := r.pool.QueryRow(ctx,
		`SELECT org_id, role FROM org_invites WHERE id = $1 AND accepted_at IS NULL AND expires_at > NOW()`,
		inviteID,
	).Scan(&orgID, &role)
	if err != nil {
		return err
	}

	// Add user to org
	_, err = r.pool.Exec(ctx,
		`INSERT INTO org_memberships (id, org_id, user_id, role)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (org_id, user_id) DO UPDATE SET role = $4`,
		uuid.NewString(), orgID, userID, role,
	)
	if err != nil {
		return err
	}

	// Mark invite as accepted
	_, err = r.pool.Exec(ctx,
		`UPDATE org_invites SET accepted_at = NOW() WHERE id = $1`, inviteID,
	)
	return err
}

func (r Repository) FindInviteByToken(ctx context.Context, tokenHash string) (OrgInvite, error) {
	var inv OrgInvite
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, email, role, token_hash, expires_at, created_at
		 FROM org_invites
		 WHERE token_hash = $1 AND accepted_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.Role, &inv.Token, &inv.ExpiresAt, &inv.CreatedAt)
	return inv, err
}

func (r Repository) ListUserOrgs(ctx context.Context, userID string) ([]Org, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT o.id, o.name, o.slug, o.plan, o.max_seats, o.max_projects, o.max_simulations_per_day, o.suspended_at, o.created_at
		 FROM organizations o
		 JOIN org_memberships om ON om.org_id = o.id
		 WHERE om.user_id = $1
		 ORDER BY o.created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Plan, &o.MaxSeats, &o.MaxProjects, &o.MaxSimulationsPerDay, &o.SuspendedAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, nil
}

func (r Repository) SwitchUserOrg(ctx context.Context, userID, orgID string) error {
	// Verify user is a member of the target org
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM org_memberships WHERE org_id = $1 AND user_id = $2)`,
		orgID, userID,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("not a member of this organization")
	}
	return nil
}

func (r Repository) CountOrgMembers(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM org_memberships WHERE org_id = $1`, orgID,
	).Scan(&count)
	return count, err
}

func (r Repository) EnsureOrgTables(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS org_invites (
			id UUID PRIMARY KEY,
			org_id UUID NOT NULL,
			email TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'engineer',
			token_hash TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			accepted_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_org_invites_org_id ON org_invites(org_id);
		CREATE INDEX IF NOT EXISTS idx_org_invites_token ON org_invites(token_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_org_memberships_unique ON org_memberships(org_id, user_id);
	`)
	return err
}

func (r Repository) DeleteSessionByRefreshHash(ctx context.Context, userID, refreshHash string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1 AND refresh_token_hash = $2`,
		userID, refreshHash,
	)
	return err
}

func (r Repository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM sessions WHERE user_id = $1`,
		userID,
	)
	return err
}

// GetOrCreateOAuthUser finds user by email or creates one (no password) for OAuth flow.
func (r Repository) GetOrCreateOAuthUser(ctx context.Context, email, fullName, provider string) (User, bool, error) {
	// Try to find existing user
	var u User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.password_hash, u.role,
		        COALESCE(u.is_platform_admin, FALSE),
		        COALESCE((SELECT om.org_id FROM org_memberships om WHERE om.user_id = u.id LIMIT 1), '')
		 FROM users u WHERE u.email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsPlatformAdmin, &u.OrgID)
	if err == nil {
		// Existing user — update full_name if empty
		if fullName != "" {
			_, _ = r.pool.Exec(ctx,
				`UPDATE users SET full_name = $2 WHERE id = $1 AND full_name = ''`,
				u.ID, fullName,
			)
		}
		return u, false, nil
	}

	// Create new user with placeholder password hash (OAuth users don't need password)
	id := uuid.NewString()
	oauthHash := "oauth:" + provider // marker — not a valid bcrypt hash
	role := "engineer"
	_, err = r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, role, email_verified)
		 VALUES ($1, $2, $3, $4, $5, TRUE)`,
		id, email, oauthHash, fullName, role,
	)
	if err != nil {
		return User{}, false, err
	}

	// Auto-create personal org
	orgID := uuid.NewString()
	orgSlug := strings.ReplaceAll(strings.Split(email, "@")[0], ".", "-") + "-" + id[:8]
	displayName := fullName
	if displayName == "" {
		displayName = strings.Split(email, "@")[0]
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, plan, max_seats, max_projects, max_simulations_per_day)
		 VALUES ($1, $2, $3, 'trial', 3, 5, 50)
		 ON CONFLICT (slug) DO NOTHING`,
		orgID, displayName+"'s Org", orgSlug,
	)
	if err == nil {
		_, _ = r.pool.Exec(ctx,
			`INSERT INTO org_memberships (id, org_id, user_id, role) VALUES ($1, $2, $3, 'owner')`,
			uuid.NewString(), orgID, id,
		)
	}

	return User{ID: id, Email: email, PasswordHash: oauthHash, Role: role, OrgID: orgID}, true, nil
}
