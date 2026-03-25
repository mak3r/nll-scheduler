---
name: review-issues
description: Review open GitHub issues for this project, analyze dependencies and refactoring risk, and produce a prioritized resolution plan. Use when the user wants to plan work, triage issues, or understand what to tackle next.
argument-hint: [focus-area]
allowed-tools: Bash(gh *), Read, Grep, Glob
---

## Current open issues (active sprint)

!`gh issue list --state open --json number,title,labels,body --limit 50 | jq '[.[] | select(.labels | map(.name) | contains(["backlog"]) | not)]'`

## Backlog issues (deferred — do not include in the plan)

!`gh issue list --state open --label 'backlog' --json number,title,labels --limit 50`

## Context

This is the **NLL Scheduler** — a little league game scheduling web app with the following services:
- `team-service` (Go + PostgreSQL) — teams, divisions, matchup rules
- `field-service` (Go + PostgreSQL) — fields, availability windows, blackout dates
- `schedule-service` (Go + PostgreSQL) — seasons, games, orchestrates the solver
- `scheduler-engine` (Python + OR-Tools / CP-SAT) — stateless constraint solver
- `frontend` (React + TypeScript + Vite) — SPA with pages: Teams, Fields, Seasons, Schedule

Focus area (if specified): $ARGUMENTS

## Your task

Analyze the active sprint issues above and produce a structured plan. Follow these steps:

### 1. Dependency analysis
Identify which issues depend on others being completed first. Look for:
- Data model changes that multiple issues rely on (e.g., a new field that several features need)
- API changes that front-end issues depend on
- Issues that share the same code area and would cause merge conflicts if done in parallel
- Issues where one fix would make another issue trivially easy (or obsolete)

### 2. Refactoring risk assessment
Flag any issue that is likely to require **major refactoring** — defined as:
- Changes to shared data models (DB schema, API request/response shapes, Pydantic schemas)
- Changes to the solver constraint system or the orchestrator flow
- Changes to the frontend API client layer (`src/api/`)
- Any change that touches 3+ services

For flagged issues, note *what* would be refactored and *which other issues or services* would be affected.

### 3. Prioritized resolution plan
Produce a numbered list of issues in recommended resolution order, respecting:
- Unblock dependencies first
- Do high-refactoring-risk items early (before other issues build on the old shape)
- Group issues that touch the same service/file to minimize context switching
- Quick wins (isolated, low-risk) can be batched at the end

For each item include:
- Issue number and title
- Service(s) affected
- Why it's in this position
- Any prerequisite issues that must land first

### 4. Risks and open questions
Call out anything ambiguous, missing context, or that warrants a quick sync with the user before starting.

### Output format
Use clear markdown headers for each section. Keep the dependency graph concise (a simple list of "X blocks Y" statements is fine). The plan should be actionable — someone should be able to read it and immediately know what to pick up next.
