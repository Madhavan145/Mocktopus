package model

import "time"

// Organization represents a B2B tenant.
type Organization struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Slug                 string         `json:"slug"`
	Plan                 string         `json:"plan"`
	MaxSeats             int            `json:"maxSeats"`
	MaxProjects          int            `json:"maxProjects"`
	MaxSimulationsPerDay int            `json:"maxSimulationsPerDay"`
	BillingEmail         string         `json:"billingEmail,omitempty"`
	StripeCustomerID     string         `json:"stripeCustomerId,omitempty"`
	Settings             map[string]any `json:"settings,omitempty"`
	SuspendedAt          *time.Time     `json:"suspendedAt,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

// OrgMembership links a user to an organization with a role.
type OrgMembership struct {
	ID       string    `json:"id"`
	OrgID    string    `json:"orgId"`
	UserID   string    `json:"userId"`
	Role     string    `json:"role"` // owner, admin, member, viewer
	Email    string    `json:"email,omitempty"`
	FullName string    `json:"fullName,omitempty"`
	JoinedAt time.Time `json:"joinedAt"`
}

// APIKey represents a programmatic access key for an org.
type APIKey struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"orgId"`
	Name            string     `json:"name"`
	KeyPrefix       string     `json:"keyPrefix"`
	Scopes          []string   `json:"scopes"`
	RateLimitPerMin int        `json:"rateLimitPerMinute"`
	LastUsedAt      *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
	RevokedAt       *time.Time `json:"revokedAt,omitempty"`
	CreatedBy       string     `json:"createdBy"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID           int64          `json:"id"`
	OrgID        string         `json:"orgId,omitempty"`
	UserID       string         `json:"userId,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

// UserSummary is a minimal user representation for admin views.
type UserSummary struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	FullName        string    `json:"fullName"`
	Role            string    `json:"role"`
	IsPlatformAdmin bool      `json:"isPlatformAdmin"`
	EmailVerified   bool      `json:"emailVerified"`
	CreatedAt       time.Time `json:"createdAt"`
}

// UsageStats provides quota tracking for an org.
type UsageStats struct {
	OrgID             string `json:"orgId"`
	SimulationsToday  int    `json:"simulationsToday"`
	SimulationsLimit  int    `json:"simulationsLimit"`
	ActiveSeats       int    `json:"activeSeats"`
	SeatsLimit        int    `json:"seatsLimit"`
	ActiveProjects    int    `json:"activeProjects"`
	ProjectsLimit     int    `json:"projectsLimit"`
	LLMCallsThisMonth int    `json:"llmCallsThisMonth"`
}

// Plan defines a billing plan tier.
type Plan struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	MaxSeats             int      `json:"maxSeats"`
	MaxProjects          int      `json:"maxProjects"`
	MaxSimulationsPerDay int      `json:"maxSimulationsPerDay"`
	PriceMonthly         float64  `json:"priceMonthly"`
	Features             []string `json:"features"`
}

// AvailablePlans returns the product catalog.
func AvailablePlans() []Plan {
	return []Plan{
		{ID: "trial", Name: "Trial", MaxSeats: 3, MaxProjects: 3, MaxSimulationsPerDay: 20, PriceMonthly: 0, Features: []string{"basic_simulation", "cost_estimation"}},
		{ID: "starter", Name: "Starter", MaxSeats: 10, MaxProjects: 25, MaxSimulationsPerDay: 200, PriceMonthly: 99, Features: []string{"basic_simulation", "cost_estimation", "recommendations", "replay", "api_access"}},
		{ID: "professional", Name: "Professional", MaxSeats: 50, MaxProjects: 100, MaxSimulationsPerDay: 2000, PriceMonthly: 499, Features: []string{"basic_simulation", "cost_estimation", "recommendations", "replay", "api_access", "llm_recommendations", "chaos_injection", "terraform_import", "priority_support"}},
		{ID: "enterprise", Name: "Enterprise", MaxSeats: 500, MaxProjects: 1000, MaxSimulationsPerDay: 50000, PriceMonthly: 0, Features: []string{"all", "sso", "audit_log", "custom_sla", "dedicated_support", "on_premise"}},
	}
}
