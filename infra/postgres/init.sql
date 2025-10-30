-- Moktopus Production Database Schema
-- All service IDs use TEXT (hex timestamps, run-xxx strings)
-- No cross-service FK constraints for microservice independence

-- ============================================================
-- ORGANIZATIONS (Multi-tenant B2B)
-- ============================================================

CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT UNIQUE NOT NULL,
  plan TEXT NOT NULL DEFAULT 'trial',  -- trial, starter, professional, enterprise
  max_seats INT NOT NULL DEFAULT 5,
  max_projects INT NOT NULL DEFAULT 10,
  max_simulations_per_day INT NOT NULL DEFAULT 100,
  billing_email TEXT,
  stripe_customer_id TEXT,
  settings JSONB NOT NULL DEFAULT '{}',
  suspended_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_organizations_plan ON organizations(plan);

-- ============================================================
-- AUTH (self-contained, uses UUIDs internally)
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  full_name TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'engineer',
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- ============================================================
-- ORG MEMBERSHIPS (users belong to orgs with roles)
-- ============================================================

CREATE TABLE IF NOT EXISTS org_memberships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'member',  -- owner, admin, member, viewer
  invited_by UUID,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_memberships_user ON org_memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_org_memberships_org ON org_memberships(org_id);

-- ============================================================
-- API KEYS (programmatic access per org)
-- ============================================================

CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  key_hash TEXT NOT NULL UNIQUE,
  key_prefix TEXT NOT NULL,  -- first 8 chars for identification (mkt_xxxx)
  scopes TEXT[] NOT NULL DEFAULT '{}',  -- simulation:run, topology:read, etc.
  rate_limit_per_minute INT NOT NULL DEFAULT 60,
  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(org_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);

-- ============================================================
-- AUDIT LOG (compliance + security)
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  org_id UUID,
  user_id UUID,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT,
  metadata JSONB NOT NULL DEFAULT '{}',
  ip_address TEXT,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_org ON audit_log(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action, created_at DESC);

-- ============================================================
-- ORG INVITES (email-based invite system)
-- ============================================================

CREATE TABLE IF NOT EXISTS org_invites (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'engineer',
  token_hash TEXT NOT NULL,
  invited_by UUID REFERENCES users(id),
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_org_invites_org ON org_invites(org_id);
CREATE INDEX IF NOT EXISTS idx_org_invites_token ON org_invites(token_hash);
CREATE INDEX IF NOT EXISTS idx_org_invites_email ON org_invites(email);

-- ============================================================
-- SESSIONS
-- ============================================================

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  refresh_token_hash TEXT NOT NULL,
  ip_address TEXT,
  device_info TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- ============================================================
-- PROJECTS
-- ============================================================

CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_by TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_projects_org ON projects(org_id);

-- ============================================================
-- TOPOLOGY / ARCHITECTURES
-- ============================================================

CREATE TABLE IF NOT EXISTS architectures (
  project_id TEXT NOT NULL,
  version INT NOT NULL DEFAULT 1,
  graph_json JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
  ctm_json JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (project_id, version)
);

CREATE INDEX IF NOT EXISTS idx_architectures_project ON architectures(project_id);

-- ============================================================
-- SIMULATION
-- ============================================================

CREATE TABLE IF NOT EXISTS simulation_runs (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  seed BIGINT NOT NULL DEFAULT 42,
  traffic_config JSONB NOT NULL DEFAULT '{}',
  topology_snapshot JSONB NOT NULL DEFAULT '{}',
  result JSONB,
  error_message TEXT,
  duration_ms BIGINT,
  total_requests INT DEFAULT 0,
  total_completed INT DEFAULT 0,
  total_failed INT DEFAULT 0,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sim_runs_project ON simulation_runs(project_id);
CREATE INDEX IF NOT EXISTS idx_sim_runs_status ON simulation_runs(status) WHERE status IN ('pending', 'running');
CREATE INDEX IF NOT EXISTS idx_sim_runs_created ON simulation_runs(created_at DESC);

CREATE TABLE IF NOT EXISTS simulation_events (
  id BIGSERIAL PRIMARY KEY,
  run_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  node_id TEXT,
  request_id TEXT,
  timestamp_ms BIGINT NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sim_events_run ON simulation_events(run_id);
CREATE INDEX IF NOT EXISTS idx_sim_events_run_ts ON simulation_events(run_id, timestamp_ms);

-- ============================================================
-- METRICS
-- ============================================================

CREATE TABLE IF NOT EXISTS metric_snapshots (
  id BIGSERIAL PRIMARY KEY,
  run_id TEXT NOT NULL,
  timestamp_ms BIGINT NOT NULL,
  rps DOUBLE PRECISION NOT NULL DEFAULT 0,
  latency_p50 DOUBLE PRECISION NOT NULL DEFAULT 0,
  latency_p95 DOUBLE PRECISION NOT NULL DEFAULT 0,
  latency_p99 DOUBLE PRECISION NOT NULL DEFAULT 0,
  error_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  throughput DOUBLE PRECISION NOT NULL DEFAULT 0,
  cpu_estimate DOUBLE PRECISION NOT NULL DEFAULT 0,
  queue_depth INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_metric_snapshots_run ON metric_snapshots(run_id);
CREATE INDEX IF NOT EXISTS idx_metric_snapshots_run_ts ON metric_snapshots(run_id, timestamp_ms);

CREATE TABLE IF NOT EXISTS node_metrics (
  id BIGSERIAL PRIMARY KEY,
  run_id TEXT NOT NULL,
  node_id TEXT NOT NULL,
  node_label TEXT,
  node_type TEXT,
  timestamp_ms BIGINT NOT NULL,
  requests_in INT NOT NULL DEFAULT 0,
  requests_out INT NOT NULL DEFAULT 0,
  requests_failed INT NOT NULL DEFAULT 0,
  requests_queued INT NOT NULL DEFAULT 0,
  latency_avg DOUBLE PRECISION NOT NULL DEFAULT 0,
  latency_p95 DOUBLE PRECISION NOT NULL DEFAULT 0,
  cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
  memory_usage DOUBLE PRECISION NOT NULL DEFAULT 0,
  queue_depth INT NOT NULL DEFAULT 0,
  error_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
  saturation DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_metrics_run_node ON node_metrics(run_id, node_id);
CREATE INDEX IF NOT EXISTS idx_node_metrics_run_ts ON node_metrics(run_id, timestamp_ms);

-- ============================================================
-- COST
-- ============================================================

CREATE TABLE IF NOT EXISTS cost_estimates (
  id BIGSERIAL PRIMARY KEY,
  project_id TEXT NOT NULL,
  run_id TEXT,
  provider TEXT NOT NULL,
  region TEXT NOT NULL DEFAULT 'us-east-1',
  monthly_total DOUBLE PRECISION NOT NULL DEFAULT 0,
  annual_estimate DOUBLE PRECISION NOT NULL DEFAULT 0,
  breakdown JSONB NOT NULL DEFAULT '{}',
  line_items JSONB NOT NULL DEFAULT '[]',
  comparison JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cost_estimates_project ON cost_estimates(project_id);

-- ============================================================
-- REPLAY
-- ============================================================

CREATE TABLE IF NOT EXISTS replay_snapshots (
  id BIGSERIAL PRIMARY KEY,
  run_id TEXT NOT NULL,
  timestamp_ms BIGINT NOT NULL,
  node_states JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_replay_snapshots_run ON replay_snapshots(run_id, timestamp_ms);

-- ============================================================
-- LLM PROVIDER CONFIGURATION (per-org)
-- ============================================================

CREATE TABLE IF NOT EXISTS llm_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,                -- display name: "OpenAI GPT-4o", "Claude Sonnet"
  provider TEXT NOT NULL,            -- openai, anthropic, azure_openai, ollama, custom
  model TEXT NOT NULL,               -- gpt-4o, claude-sonnet-4-20250514, llama3, etc.
  endpoint TEXT NOT NULL,            -- https://api.openai.com/v1, http://localhost:11434, etc.
  api_key_encrypted TEXT,            -- encrypted at rest
  max_tokens INT NOT NULL DEFAULT 4096,
  temperature DOUBLE PRECISION NOT NULL DEFAULT 0.3,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_llm_providers_org ON llm_providers(org_id);

-- ============================================================
-- USAGE TRACKING (billing, quota enforcement)
-- ============================================================

CREATE TABLE IF NOT EXISTS usage_records (
  id BIGSERIAL PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id UUID,
  resource_type TEXT NOT NULL,       -- simulation_run, llm_call, api_request
  quantity INT NOT NULL DEFAULT 1,
  metadata JSONB NOT NULL DEFAULT '{}',
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_records_org_date ON usage_records(org_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_records_type ON usage_records(resource_type, recorded_at DESC);

-- ============================================================
-- PLATFORM SEED DATA
-- ============================================================

-- Default platform admin (change password immediately in production)
INSERT INTO users (email, password_hash, full_name, role, is_platform_admin, email_verified)
VALUES ('admin@moktopus.io', '$argon2id$v=19$m=65536,t=3,p=2$c2VlZHNhbHQ$placeholder', 'Platform Admin', 'owner', TRUE, TRUE)
ON CONFLICT (email) DO NOTHING;

-- Default organization for self-hosted / trial
INSERT INTO organizations (id, name, slug, plan, max_seats, max_projects, max_simulations_per_day)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Organization', 'default', 'enterprise', 100, 1000, 10000)
ON CONFLICT (slug) DO NOTHING;
