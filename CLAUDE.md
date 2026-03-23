# NLL Scheduler — Claude Instructions

## Project Overview
Little league game scheduling web app. Microservices on Kubernetes. No authentication required.

## Services
| Service | Language | Port (local) | Role |
|---|---|---|---|
| team-service | Go + PostgreSQL | 8081 | Teams, divisions, matchup rules |
| field-service | Go + PostgreSQL | 8082 | Fields, availability windows, blackout dates |
| schedule-service | Go + PostgreSQL | 8083 | Seasons, games, orchestrates solver |
| scheduler-engine | Python + OR-Tools | 8084 | Stateless CP-SAT solver (ClusterIP only) |
| frontend | React + TypeScript + Vite | 3000 | SPA |

## Local Development
```bash
tilt up          # builds images, deploys to K8s, watches for changes
tilt down        # tear down
```
- Requires: `kubectl` pointing to local dev cluster, `tilt` installed
- Dev namespace: `nll-scheduler-dev`
- **No docker-compose** — K8s is the dev workflow for all environments
- Port-forwards are configured automatically by Tilt

## Repository Structure
```
nll-scheduler/
├── team-service/           # Go: cmd/server/, internal/{api,db,model,repository}/
├── field-service/          # Go: + internal/domain/availability_calculator.go
├── schedule-service/       # Go: + internal/orchestrator/ (generator, clients)
├── scheduler-engine/       # Python FastAPI: app/{constraints,solver,schemas}/
├── frontend/               # React + Vite: src/{api,pages}/
├── k8s/                    # K8s manifests per service + ingress + namespace
├── .github/workflows/      # Path-scoped CI/CD per service
└── Tiltfile
```

## Go Services — Key Conventions
- Module names: `github.com/nll-scheduler/{team,field,schedule}-service`
- Router: `github.com/go-chi/chi/v5`
- Database: `github.com/jackc/pgx/v5` with `pgxpool`
- Migrations: `github.com/golang-migrate/migrate/v4` with `//go:embed migrations/*.sql`
- `go.sum` files are empty placeholders — generated inside Docker via `go mod tidy`
- Postgres runs as a sidecar in the same pod; `DATABASE_URL` uses `localhost:5432`
- Repository sentinel error: `repository.ErrNotFound` → HTTP 404
- pgx `INT[]` columns: scan into `[]int32`, convert to `[]int` for model layer

## scheduler-engine — Key Conventions
- FastAPI + Pydantic v2, Python 3.12
- CP-SAT variable formulation: `x[home_idx, away_idx, slot_idx] ∈ {0,1}`
- **Extensible constraint system**: implement `ConstraintHandler` in `app/constraints/`, register in `app/constraints/__init__.py` — no other changes needed
- Built-in constraints: `round_robin_matchup`, `max_games_per_field_per_day`, `max_games_per_team_per_week`, `min_rest_days_between_games`, `prefer_interleague_dates`, `even_home_away_balance`
- Lint: `ruff check app/`
- Test: `pytest tests/`

## frontend — Key Conventions
- Vite dev server proxies `/api/teams`, `/api/fields`, `/api/schedule` to respective services
- Typed API clients in `src/api/` (client.ts, teams.ts, fields.ts, schedule.ts)
- Pages: TeamsPage, FieldsPage, SeasonsPage, SchedulePage
- Lint: `npm run lint` (ESLint 9 flat config)
- Type check: `npx tsc --noEmit`

## K8s — Key Details
- Each Go service: Deployment (postgres sidecar + app), Service (ClusterIP), ConfigMap, Secret, PVC
- scheduler-engine: no PVC (stateless), CPU requests 500m / limits 2, HPA 1–4 replicas at 60% CPU
- Ingress: nginx path-based routing — `/api/teams/`, `/api/fields/`, `/api/schedule/`, `/` → frontend
- `DATABASE_URL` in secrets uses `localhost:5432` (sidecar pattern)

## CI/CD
Path-scoped GitHub Actions workflows in `.github/workflows/` — each triggers only on changes to its service directory. Pipeline: lint → test → build → push → deploy.

Required GitHub Secrets: `REGISTRY`, `REGISTRY_USER`, `REGISTRY_PASSWORD`, `KUBECONFIG`

## Schedule Generation Flow
1. `POST /seasons/{id}/generate` → creates `generation_runs` row, returns `{run_id}`
2. Poll `GET /seasons/{id}/generate/{runId}` until status = `success` | `failed`
3. Background goroutine: fetches teams (team-service) + field availability (field-service) → builds `SolveRequest` → calls scheduler-engine `/solve` → persists games → updates run status
4. Season status transitions: `draft` → `generating` → `review` (or back to `draft` on failure)
