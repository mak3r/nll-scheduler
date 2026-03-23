# Architecture

## Service map

```
                         ┌─────────────────────────────────────────────┐
                         │              Kubernetes cluster               │
                         │              (nll-scheduler-dev)              │
                         │                                               │
Browser ──── Ingress ────┼──/api/teams/    ──► team-service    + pg     │
             (nginx)     │  /api/fields/   ──► field-service   + pg     │
                         │  /api/schedule/ ──► schedule-service + pg    │
                         │  /              ──► frontend                  │
                         │                         │                     │
                         │              schedule-service                  │
                         │                 ├──► team-service             │
                         │                 ├──► field-service            │
                         │                 └──► scheduler-engine         │
                         │                      (ClusterIP only)         │
                         └─────────────────────────────────────────────┘
```

## Services

### team-service — Go + PostgreSQL · port 8081

Owns teams, divisions, and matchup rules. The canonical source of truth for who is playing.

Key endpoints:
- `GET/POST /divisions` and `/divisions/{id}`
- `GET /divisions/{id}/teams-with-rules` — used by schedule-service during generation
- `GET/POST /teams` and `/teams/{id}`
- `GET/POST /teams/{id}/matchup-rules`

### field-service — Go + PostgreSQL · port 8082

Owns fields and their availability. Materialized availability (slots filtered by blackout dates) is computed on demand.

Key endpoints:
- `GET/POST /fields` and `/fields/{id}`
- `GET/POST /fields/{id}/availability-windows`
- `GET/POST /fields/{id}/blackout-dates`
- `GET /fields/available-dates-bulk?start=&end=&field_ids=` — used by schedule-service during generation; returns a map of `field_id → []AvailableSlot`

Availability logic lives in `field-service/internal/domain/availability_calculator.go`. It materializes recurring windows into concrete date/time slots and filters out blackout dates.

### schedule-service — Go + PostgreSQL · port 8083

Owns seasons and games. Orchestrates schedule generation by coordinating with the other three services.

Key endpoints:
- `GET/POST /seasons` and `/seasons/{id}`
- `GET/POST /seasons/{id}/blackout-dates`
- `GET/POST /seasons/{id}/preferred-interleague-dates`
- `GET/POST /seasons/{id}/constraints`
- `GET/POST /seasons/{id}/games` and `/seasons/{id}/games/{id}`
- `POST /seasons/{id}/games/check-conflicts`
- `POST /seasons/{id}/generate` — triggers async generation, returns `{run_id}`
- `GET /seasons/{id}/generate/{runId}` — polls generation status
- `GET /seasons/{id}/export?format=json`

### scheduler-engine — Python + OR-Tools · port 8084 (ClusterIP only)

Stateless CP-SAT constraint solver. Accepts a `SolveRequest` and returns a `SolveResponse` with scheduled games. Never stores state; all context comes from the request.

Only reachable from within the cluster (ClusterIP service). Scales horizontally via HPA (1–4 replicas, target 60% CPU).

Key endpoint:
- `POST /solve`

### frontend — React + TypeScript + Vite · port 3000

Single-page application with four pages: Teams, Fields, Seasons, Schedule. Typed API clients in `src/api/` make requests through the ingress (production) or Vite's dev proxy (local development).

---

## Postgres sidecar pattern

Each Go service runs a Postgres container **in the same pod**:

```
┌─── pod: team-service ──────────────────┐
│  container: postgres:16-alpine          │
│    port: 5432 (localhost only)          │
│    data: PVC (10Gi)                     │
│                                         │
│  container: team-service (Go)           │
│    DATABASE_URL: localhost:5432/team_db │
└─────────────────────────────────────────┘
```

The app container connects to `localhost:5432` — no network hop, no TLS, no separate database deployment. Each service's data is fully isolated. Schema migrations run automatically at startup using `golang-migrate` with embedded SQL files.

---

## Schedule generation flow

```
User                schedule-service         team-service   field-service   scheduler-engine
 │                        │                       │               │                │
 │── POST /generate ──────►│                       │               │                │
 │◄── {run_id} ───────────│                       │               │                │
 │                        │── GET /teams-with-rules►│               │                │
 │                        │◄── teams + rules ──────│               │                │
 │                        │── GET /available-dates-bulk ──────────►│                │
 │                        │◄── field slots ──────────────────────-│                │
 │                        │                       │               │                │
 │                        │────────── POST /solve ────────────────────────────────►│
 │                        │                       │               │  (CP-SAT runs) │
 │                        │◄─────────── {games, stats} ────────────────────────────│
 │                        │                       │               │                │
 │                        │ (persist games, update season/run status)               │
 │── GET /generate/{id} ──►│                       │               │                │
 │◄── {status, stats} ────│                       │               │                │
```

Season status transitions:
- `draft` → `generating` (on start)
- `generating` → `review` (on success)
- `generating` → `draft` (on failure)

Generation run status: `pending` → `running` → `success` | `failed`

---

## Constraint system

The scheduler-engine uses an extensible registry of constraint handlers. Each handler adds rules or objectives to the CP-SAT model.

**CP-SAT variable formulation:**
```
x[home_idx, away_idx, slot_idx] ∈ {0, 1}
  = 1 if home_team plays away_team at that slot
```

**Built-in constraints:**

| Type string | Hard/Soft | Params | Effect |
|---|---|---|---|
| `round_robin_matchup` | Hard | `default_games_per_pair` | Each team pair plays min–max games |
| `max_games_per_field_per_day` | Hard | _(none)_ | Respects field's `max_games_per_day` |
| `max_games_per_team_per_week` | Hard | `max_games_per_week` | Caps games per team per ISO week |
| `min_rest_days_between_games` | Hard | `min_rest_days` | Minimum days between a team's games |
| `prefer_interleague_dates` | Soft | `bonus_per_game` | Rewards interleague games on preferred dates |
| `even_home_away_balance` | Soft | `penalty_weight` | Penalizes unequal home/away game counts |

**Adding a new constraint:**
1. Create `scheduler-engine/app/constraints/my_constraint.py`, subclassing `ConstraintHandler`
2. Implement `constraint_type` (a unique string) and `apply(model, variables, problem, params)`
3. Import and register it in `scheduler-engine/app/constraints/__init__.py`

No other files need to change. Constraints are activated per-season by adding a row to `season_constraints` with the matching `type` string and a `params` JSON object.

---

## Frontend architecture

The SPA communicates with backend services through the ingress. In local development, Vite's dev proxy rewrites `/api/teams/`, `/api/fields/`, and `/api/schedule/` to the corresponding port-forwarded service, matching production ingress routing exactly.

```
src/
├── api/
│   ├── client.ts      # base request helper
│   ├── teams.ts       # typed calls to team-service
│   ├── fields.ts      # typed calls to field-service
│   └── schedule.ts    # typed calls to schedule-service
└── pages/
    ├── TeamsPage.tsx
    ├── FieldsPage.tsx
    ├── SeasonsPage.tsx
    └── SchedulePage.tsx
```

In production, the frontend Dockerfile has a `production` stage that runs `npm run build` and serves the static output from nginx.

---

## Kubernetes topology

```
Namespace: nll-scheduler-dev

Deployments:
  team-service        (1 replica, postgres sidecar, PVC)
  field-service       (1 replica, postgres sidecar, PVC)
  schedule-service    (1 replica, postgres sidecar, PVC)
  scheduler-engine    (1–4 replicas via HPA, no PVC)
  frontend            (1 replica)

Services (all ClusterIP):
  team-service:8080
  field-service:8080
  schedule-service:8080
  scheduler-engine:8080
  frontend:80

HPA:
  scheduler-engine — min 1, max 4, target 60% CPU
  scale-up:   +1 pod per 30s
  scale-down: -1 pod per 60s, 5-minute stabilization window

Ingress (nginx):
  /api/teams(/|$)(.*)    → team-service:8080
  /api/fields(/|$)(.*)   → field-service:8080
  /api/schedule(/|$)(.*) → schedule-service:8080
  /()(.*)                → frontend:80
```
