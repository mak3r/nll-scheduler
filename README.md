# NLL Scheduler

[![team-service](https://github.com/mak3r/nll-scheduler/actions/workflows/team-service.yml/badge.svg)](https://github.com/mak3r/nll-scheduler/actions/workflows/team-service.yml)
[![field-service](https://github.com/mak3r/nll-scheduler/actions/workflows/field-service.yml/badge.svg)](https://github.com/mak3r/nll-scheduler/actions/workflows/field-service.yml)
[![schedule-service](https://github.com/mak3r/nll-scheduler/actions/workflows/schedule-service.yml/badge.svg)](https://github.com/mak3r/nll-scheduler/actions/workflows/schedule-service.yml)
[![scheduler-engine](https://github.com/mak3r/nll-scheduler/actions/workflows/scheduler-engine.yml/badge.svg)](https://github.com/mak3r/nll-scheduler/actions/workflows/scheduler-engine.yml)
[![frontend](https://github.com/mak3r/nll-scheduler/actions/workflows/frontend.yml/badge.svg)](https://github.com/mak3r/nll-scheduler/actions/workflows/frontend.yml)

A web application for scheduling little league games. Given teams, fields, and a set of scheduling constraints, it uses constraint programming to generate a complete season schedule automatically.

## Features

- **Teams** — Manage divisions, teams (local and interleague), and per-pair matchup rules that control how many times two teams play each other
- **Fields** — Configure playing fields with recurring or one-off availability windows, per-day game limits, and blackout dates
- **Seasons** — Define season date ranges, layered scheduling constraints, season-level blackout dates, and preferred dates for interleague games
- **Schedule generation** — Trigger an async CP-SAT solver run; poll for completion; review the generated schedule in a game table; manually edit games; check for field conflicts; export to JSON

## Architecture

Five services deployed on Kubernetes:

| Service | Language | Role |
|---|---|---|
| team-service | Go + PostgreSQL | Teams, divisions, matchup rules |
| field-service | Go + PostgreSQL | Fields, availability windows, blackout dates |
| schedule-service | Go + PostgreSQL | Seasons, games, orchestrates the solver |
| scheduler-engine | Python + OR-Tools | Stateless CP-SAT constraint solver |
| frontend | React + TypeScript + Vite | Single-page application |

An nginx ingress routes `/api/teams/`, `/api/fields/`, and `/api/schedule/` to the respective backend services, and `/` to the frontend. The scheduler-engine is internal-only (ClusterIP) and scales horizontally via HPA.

## Documentation

- [Quickstart](docs/quickstart.md) — run the full stack locally in under 10 minutes
- [Architecture](docs/architecture.md) — how the services fit together, the schedule generation flow, and the constraint system
- [Design](docs/design.md) — rationale behind the technology choices and architectural decisions

## Local development

**Prerequisites:** [Rancher Desktop](https://rancherdesktop.io/) (or kind/k3d), `kubectl`, and [Tilt](https://docs.tilt.dev/install.html).

```bash
git clone https://github.com/mak3r/nll-scheduler.git
cd nll-scheduler
tilt up
```

Tilt builds all images, applies the Kubernetes manifests, and sets up port-forwards automatically. Open [http://localhost:3000](http://localhost:3000) to use the app.

See [docs/quickstart.md](docs/quickstart.md) for a full walkthrough.

## License

Apache 2.0 — see [LICENSE](LICENSE).
