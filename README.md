# Moktopus

Implementation is now being executed against `Final-SRS` in strict phases.


## Local-First Run

Preferred:

```bash
make dev
```

Equivalent:

```bash
docker compose up -d --build
```

Main endpoints:

- Frontend: http://localhost:3000
- Gateway: http://localhost:8080
- Gateway health: http://localhost:8080/health
- Gateway metrics: http://localhost:8080/metrics
- Prometheus (optional): http://localhost:9090
- Grafana (optional): http://localhost:3001

## Current Service Topology (Local)

- `frontend`
- `gateway`
- `auth`
- `admin`
- `project`
- `topology`
- `simulation`
- `metrics`
- `cost`
- `chaos`
- `replay`
- `recommendation`
- `catalog`
- `iac`
- `template`
- `prometheus` (optional observability)
- `grafana` (optional observability)
- `postgres`
- `redis`

## API Routing

Gateway supports both namespaces for transition safety:

- Existing: `/api/*`
- SRS-aligned: `/api/v1/*`

Core v1 endpoints now available include:

- `/api/v1/system/status`
- `/api/v1/auth/*`
- `/api/v1/topology/*`
- `/api/v1/simulation/*`
- `/api/v1/metrics/derive`
- `/api/v1/cost/estimate`
- `/api/v1/replay/reconstruct`
- `/api/v1/recommendation/derive`
- `/api/v1/catalog/*`
- `/api/v1/iac/*`
- `/api/v1/templates/*`
- `/api/v1/admin/*`

## Final-SRS Build Phases

See `docs/architecture/final-srs-phases.md` for execution plan and status.
