# Developer Quickstart

Get the full NLL Scheduler stack running locally in under 10 minutes.

## Prerequisites

| Tool | Purpose | Install |
|---|---|---|
| Kubernetes cluster | Any local cluster `kubectl` can reach | [Rancher Desktop](https://rancherdesktop.io/), [k3s](https://k3s.io/), [kind](https://kind.sigs.k8s.io/), or [k3d](https://k3d.io/) |
| `kubectl` | Kubernetes CLI | Bundled with Rancher Desktop, or `brew install kubectl` |
| [Podman](https://podman.io/) | Container runtime — builds and pushes images | `brew install podman` |
| `vfkit` | macOS Virtualization Framework driver for Podman | `brew install vfkit` |
| [Tilt](https://docs.tilt.dev/install.html) | Dev orchestration (builds, deploys, port-forwards, live reload) | `brew install tilt-dev/tap/tilt` |
| [GitHub CLI](https://cli.github.com/) | Authenticates with ghcr.io | `brew install gh` |

> Images are pushed to `ghcr.io/mak3r/nll-scheduler` during development, so any cluster with internet access can pull them — no local registry required.

### Verify your setup

```bash
kubectl config current-context   # should show your local cluster
tilt version                     # should print a version number
```

## Container runtime setup

**One-time Podman machine initialization** (downloads a Fedora CoreOS VM, ~700 MB):

```bash
softwareupdate --install-rosetta   # required on Apple Silicon for VM bootstrap
brew install vfkit podman
podman machine init
podman machine start
```

### Authenticate with ghcr.io

Tilt pushes images to `ghcr.io/mak3r/nll-scheduler` — authenticate once before running `tilt up`:

```bash
gh auth login                    # if not already logged in to GitHub CLI
echo $(gh auth token) | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

## Start the stack

On macOS with Podman, use the wrapper script instead of `tilt up` directly. It creates an SSH tunnel from a local socket to the Podman VM, launches Tilt, and tears the tunnel down on exit:

```bash
git clone https://github.com/mak3r/nll-scheduler.git
cd nll-scheduler
./scripts/tilt-up.sh
```

The script also starts the Podman machine automatically if it isn't already running.

Tilt will:
1. Build Docker images for all five services
2. Apply the Kubernetes manifests to the `nll-scheduler-dev` namespace
3. Set up port-forwards so you can reach every service from `localhost`

Watch the Tilt UI (opened automatically in your browser) for build and pod status.

## Port-forwards

Once all services are green in Tilt, the following are available:

| Service | URL |
|---|---|
| Frontend (React SPA) | http://localhost:3000 |
| team-service API | http://localhost:8081 |
| field-service API | http://localhost:8082 |
| schedule-service API | http://localhost:8083 |
| scheduler-engine API | http://localhost:8084 |

## Smoke-test

```bash
curl http://localhost:8081/health
curl http://localhost:8082/health
curl http://localhost:8083/health
curl http://localhost:8084/health
```

Each should return `{"status":"ok","service":"<name>"}`.

## End-to-end walkthrough

Open [http://localhost:3000](http://localhost:3000) and follow these steps:

### 1. Teams tab
- Create a **division** (name + season year)
- Add at least two **teams** to the division
  - Set `team_type` to `local` for most teams; use `interleague` for teams that visit from other leagues
  - `games_required` controls how many games each team should play in the season
- Optionally add **matchup rules** to control min/max games between specific pairs

### 2. Fields tab
- Create a **field** (name, address, max games per day)
- Add at least one **availability window**:
  - `recurring` — repeats on selected days of the week within a date range (e.g. every Saturday/Sunday from April through June)
  - `oneoff` — a single specific date/time block
- Optionally add **blackout dates** (holidays, field maintenance, etc.)

### 3. Seasons tab
- Create a **season** linked to your division, with start/end dates
- Add at least one **constraint** — at minimum `round_robin_matchup` with `{"default_games_per_pair": 2}`
- Optionally add:
  - Season-level **blackout dates** (applied on top of field blackouts)
  - **Preferred interleague dates** (the solver will prefer to schedule interleague games on these dates)
  - Additional constraints (rest days, weekly game limits, home/away balance)
- Click **Generate Schedule** — this is async; the season status changes to `generating`
- Poll the status or wait for the UI to update. Generation typically completes within 60 seconds

### 4. Schedule tab
- Select your season from the dropdown
- Review the generated games (home team, away team, field, date, time)
- Use **Check Conflicts** to detect any field double-bookings
- Edit individual games if needed (date, time, field, status)
- Use **Export** to download the schedule as JSON

## Tear down

```bash
tilt down
```

This removes all Kubernetes resources in `nll-scheduler-dev`. Persistent volumes (Postgres data) are retained unless you delete the namespace manually.

## Hot reload

Changes you make to source files are picked up automatically:

| Service | Behavior |
|---|---|
| Go services | File change → image rebuild → pod restart (~5s) |
| scheduler-engine | `.py` files sync into the running container; `uvicorn --reload` picks up changes instantly |
| Frontend | `src/` files sync; Vite HMR hot-reloads the browser without a page refresh |

## Running tests without Tilt

If you want to run tests outside of the K8s environment:

```bash
# Go services (from repo root)
cd team-service     && go test ./... -race -count=1
cd ../field-service    && go test ./... -race -count=1
cd ../schedule-service && go test ./... -race -count=1

# Python scheduler-engine
cd scheduler-engine
pip install -r requirements.txt ruff pytest pytest-asyncio httpx
ruff check app/
pytest tests/ -v

# Frontend
cd frontend
npm ci
npm run lint
npx tsc --noEmit
npm run build
```

## Next steps

- [Architecture](architecture.md) — understand how the services interact
- [Design](design.md) — understand why the stack was built this way
