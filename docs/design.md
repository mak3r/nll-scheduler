# Design Decisions

This document explains the *why* behind the major architectural and technology choices in NLL Scheduler.

---

## Microservices over a monolith

The application is split into five services with clear domain boundaries: teams, fields, scheduling, the solver, and the frontend.

The primary driver is the solver. The `scheduler-engine` is a compute-intensive process (CP-SAT can run for tens of seconds) that benefits from independent horizontal scaling. Running it as a separate service with its own HPA means solving can scale without affecting the CRUD services, and vice versa.

The three Go services (team, field, schedule) also have distinct ownership of their data. Keeping them separate avoids a shared schema and makes it possible to evolve or redeploy each independently. The trade-off is operational complexity — more moving parts to deploy and monitor. That cost is managed by making Kubernetes the only environment (see below).

---

## Kubernetes as the only development environment

There is no docker-compose. Kubernetes is used for local development, and the same manifests run in production.

The key benefit is parity: a bug caused by how services discover each other, how ingress routing works, or how postgres sidecars start up will appear in local development rather than only in production. The "works on my machine" failure mode is largely eliminated.

The ergonomic cost of using Kubernetes locally (slow feedback loops, manual port-forwarding, watching logs across multiple pods) is addressed entirely by Tilt. Tilt provides live reload, automatic port-forwards, a unified log view, and a dependency graph that knows which pods to rebuild when a file changes. With Tilt, the developer experience is comparable to docker-compose for most workflows.

---

## Postgres sidecar pattern

Each Go service runs a Postgres container in the same pod, connected via `localhost:5432`.

The alternatives were: a single shared Postgres cluster (one Deployment, multiple databases), or a separate Postgres Deployment per service.

A shared cluster would couple the services operationally — a Postgres restart or migration failure in one service would affect all of them. Separate Deployments would require service-to-service networking for database connections and more manifest maintenance.

The sidecar pattern gives each service full isolation with minimal overhead. The connection is `localhost`, so there is no TLS, no service discovery, and no latency from a network hop. Data is persisted to a PVC that survives pod restarts.

The main trade-off: a pod restart takes down both the app and the database simultaneously. For a scheduling app with low write frequency, this is acceptable. An app that required high write availability would need a different approach.

---

## CP-SAT for schedule generation

Schedule generation is a constraint satisfaction and optimization problem. Each game must be assigned to a (home team, away team, field, time slot) tuple such that:

- Hard constraints are satisfied (e.g. a field can only host one game at a time, teams need rest between games)
- Soft constraints are optimized (e.g. interleague games on preferred dates, balanced home/away counts)

This is a classic application for constraint programming. Google OR-Tools CP-SAT handles NP-hard scheduling problems efficiently by combining SAT solving with optimization, and it allows a time budget so the solver returns the best solution found within N seconds even if it can't prove optimality.

The alternative — greedy or heuristic algorithms — would be simpler to implement but wouldn't guarantee hard constraint satisfaction or allow soft constraints to be expressed as weighted objectives. CP-SAT gives a single, principled framework for both.

The constraint system is designed to be extensible without touching the solver core. Each constraint type is a self-contained handler that adds rules or objectives to the CP-SAT model. New constraints can be added by implementing a single interface and registering the handler — no changes to the solver or the API schema are required.

---

## Stateless scheduler-engine

The solver is a pure function: `SolveRequest → SolveResponse`. It holds no state between requests.

This was a deliberate choice that enables horizontal scaling. Because there is no shared state, multiple solver replicas can run simultaneously without coordination. The HPA can spin up additional replicas when CPU utilization rises (e.g. when multiple seasons are being generated at once) and scale back down without any draining or handoff logic.

The service is ClusterIP-only — it cannot be reached from outside the cluster. Only `schedule-service` calls it, and only during generation runs.

---

## Go for the backend services

Go was chosen for team-service, field-service, and schedule-service for three reasons:

**Concurrency.** Schedule generation is an async operation driven by a background goroutine. Go's goroutines and channels make this straightforward without pulling in a separate task queue.

**Performance.** The CRUD services need to be fast and lightweight. Go produces small, statically compiled binaries with low memory overhead, which keeps resource requests small in Kubernetes.

**Ecosystem fit.** `pgx/v5` is one of the most capable PostgreSQL drivers available in any language, with native support for `pgxpool`, named parameters, and PostgreSQL-specific types (like `INT[]`). `chi` is a lightweight, idiomatic router with no lock-in. `golang-migrate` with embedded SQL migrations keeps schema management simple and self-contained — migrations run automatically at startup from files compiled into the binary.

---

## React + Vite + TypeScript for the frontend

**TypeScript** catches mismatches between the frontend's API client types and the actual backend response shapes at compile time. Given that the backend schemas are defined in Go structs, having typed clients on the frontend is the most effective way to catch integration bugs without integration tests.

**Vite** provides fast HMR during development. Combined with Tilt's live file sync, source changes appear in the browser in under a second without a full rebuild or pod restart.

**React Router v6** enables SPA navigation (no server round-trips between pages) with URL-based state (e.g. `?season=<id>` on the Schedule page).

React hooks (`useState`, `useEffect`) are sufficient for this application's state complexity. Adding a state management library (Redux, Zustand, etc.) would be over-engineering for an app with four pages and no cross-page shared state.

---

## Path-scoped CI/CD with GITHUB_TOKEN

Each service has its own GitHub Actions workflow that triggers only when files under that service's directory change. A change to `team-service/` does not trigger a build of `schedule-service/`.

This keeps CI fast and predictable — a commit that fixes a frontend typo does not rebuild and redeploy all five backend services.

Container images are published to `ghcr.io` using the built-in `GITHUB_TOKEN` with `packages: write` permission. No long-lived PAT or external registry secret is needed. This is both simpler and more secure — the token is scoped to the workflow run and cannot be exfiltrated or rotated incorrectly.
