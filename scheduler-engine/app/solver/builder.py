"""
CP-SAT model builder — full implementation.

Variable formulation:
  x[home_idx, away_idx, slot_idx] ∈ {0, 1}
  = 1 if team[home_idx] hosts team[away_idx] at slots[slot_idx]

A "slot" is (field_id, date_str, start_time_str, end_time_str) from field-service.
"""
from __future__ import annotations

import logging
from collections import defaultdict
from datetime import date, timedelta

from ortools.sat.python import cp_model

from app.constraints import REGISTRY
from app.schemas.solve import FieldSlot, GameResult, SolveRequest, SolveResponse

logger = logging.getLogger(__name__)


def build_stub_response(request: SolveRequest) -> SolveResponse:
    """Phase 0 stub — kept for backwards compatibility."""
    return SolveResponse(
        status="stub",
        games=[],
        solver_stats={
            "teams": len(request.teams),
            "fields": len(request.fields),
            "note": "stub response",
        },
        unmet_constraints=[],
    )


def solve(request: SolveRequest) -> SolveResponse:
    """Build and solve the CP-SAT scheduling model."""
    teams = request.teams
    n_teams = len(teams)

    if n_teams < 2:
        return SolveResponse(status="infeasible", solver_stats={"reason": "need at least 2 teams"})

    # --- Collect and filter slots ---
    blackout_set: set[str] = {d.isoformat() for d in request.blackout_dates}

    all_slots: list[tuple[str, str, str, str]] = []  # (field_id, date, start_time, end_time)
    for field in request.fields:
        for slot in field.available_slots:
            date_str = slot.date.isoformat()
            if date_str not in blackout_set:
                all_slots.append((
                    field.id,
                    date_str,
                    str(slot.start_time),
                    str(slot.end_time),
                ))
    all_slots.sort(key=lambda s: (s[1], s[2], s[0]))  # sort by date, time, field

    n_slots = len(all_slots)
    if n_slots == 0:
        return SolveResponse(status="infeasible", solver_stats={"reason": "no available slots after blackout filtering"})

    logger.info("Building model: %d teams, %d slots", n_teams, n_slots)

    # --- Index structures ---
    team_idx: dict[str, int] = {t.id: i for i, t in enumerate(teams)}

    # Map field_id -> Field object for max_games_per_day lookup
    field_map = {f.id: f for f in request.fields}

    # Group slot indices by (field_id, date)
    slots_by_field_date: dict[tuple[str, str], list[int]] = defaultdict(list)
    # Group slot indices by (team_idx, iso_week)
    slots_by_date: dict[str, list[int]] = defaultdict(list)
    slot_dates: list[date] = []

    for s_idx, (fid, d_str, st, et) in enumerate(all_slots):
        slots_by_field_date[(fid, d_str)].append(s_idx)
        slots_by_date[d_str].append(s_idx)
        slot_dates.append(date.fromisoformat(d_str))

    # Group slot indices by (team, week_number) — for max-per-week constraints
    # week key = (year, isoweek)
    def week_key(d: date) -> tuple[int, int]:
        iso = d.isocalendar()
        return (iso.year, iso.week)

    slots_by_week: dict[tuple[int, int], list[int]] = defaultdict(list)
    for s_idx, d in enumerate(slot_dates):
        slots_by_week[week_key(d)].append(s_idx)

    # Build matchup index: for each unordered pair {i, j}, get (min_games, max_games)
    matchup_limits: dict[tuple[int, int], tuple[int, int]] = {}
    for rule in request.matchup_rules:
        ai = team_idx.get(rule.team_a_id)
        bi = team_idx.get(rule.team_b_id)
        if ai is not None and bi is not None:
            pair = (min(ai, bi), max(ai, bi))
            matchup_limits[pair] = (rule.min_games, rule.max_games)

    # --- Build per-team allowed field sets from division restrictions ---
    div_restrictions: dict[str, list[str]] = dict(request.division_field_restrictions or {})
    team_allowed_fields: list[set[str] | None] = []
    for team in teams:
        allowed = div_restrictions.get(team.division_id)
        team_allowed_fields.append(set(allowed) if allowed is not None else None)

    # --- Build CP-SAT model ---
    model = cp_model.CpModel()

    # Decision variables — sparse: only create for valid team-field combinations.
    # Cross-division pairs are only created if an explicit matchup rule exists.
    # x[i, j, s] = team i (home) vs team j (away) at slot s
    x: dict[tuple[int, int, int], cp_model.IntVar] = {}
    for i in range(n_teams):
        for j in range(n_teams):
            if i == j:
                continue
            # Cross-division pairs only if there's an explicit matchup rule
            if teams[i].division_id != teams[j].division_id:
                pair = (min(i, j), max(i, j))
                if pair not in matchup_limits:
                    continue
            ta = team_allowed_fields[i]
            tb = team_allowed_fields[j]
            for s in range(n_slots):
                fid = all_slots[s][0]
                if ta is not None and fid not in ta:
                    continue
                if tb is not None and fid not in tb:
                    continue
                x[i, j, s] = model.new_bool_var(f"x_{i}_{j}_{s}")

    # Helper: all vars where team i plays (home or away) at slot s
    def team_plays_at(i: int, s: int) -> list[cp_model.IntVar]:
        result = []
        for j in range(n_teams):
            if i == j:
                continue
            if (i, j, s) in x:
                result.append(x[i, j, s])
            if (j, i, s) in x:
                result.append(x[j, i, s])
        return result

    # Helper: total games between pair (i, j) regardless of home/away
    def pair_game_vars(i: int, j: int) -> list[cp_model.IntVar]:
        result = []
        for s in range(n_slots):
            if (i, j, s) in x:
                result.append(x[i, j, s])
            if (j, i, s) in x:
                result.append(x[j, i, s])
        return result

    # --- Variables dict for constraint handlers ---
    variables = {
        "x": x,
        "teams": teams,
        "slots": all_slots,
        "slot_dates": slot_dates,
        "team_idx": team_idx,
        "field_map": field_map,
        "matchup_limits": matchup_limits,
        "slots_by_field_date": slots_by_field_date,
        "slots_by_date": slots_by_date,
        "slots_by_week": slots_by_week,
        "team_plays_at": team_plays_at,
        "pair_game_vars": pair_game_vars,
        "objective_terms": [],  # soft constraint handlers append here
    }

    # --- Apply constraints from registry ---
    registered_types = set()
    for cfg in request.constraints:
        handler = REGISTRY.get(cfg.type)
        if handler:
            logger.info("Applying constraint: %s (hard=%s)", cfg.type, cfg.is_hard)
            handler.apply(model, variables, request, cfg.params)
            registered_types.add(cfg.type)
        else:
            logger.warning("Unknown constraint type: %s — skipping", cfg.type)

    # --- Apply built-in defaults if not explicitly configured ---
    # Always apply these core constraints if not already registered
    for builtin_type in ("round_robin_matchup", "max_games_per_field_per_day",
                         "max_games_per_team_per_week", "min_rest_days_between_games"):
        if builtin_type not in registered_types:
            handler = REGISTRY.get(builtin_type)
            if handler:
                logger.info("Applying built-in default constraint: %s", builtin_type)
                handler.apply(model, variables, request, {})

    # --- One game per slot: each slot hosts at most 1 game ---
    # Build per-slot variable lists from sparse x dict (avoids iterating all i,j for each slot)
    vars_by_slot: dict[int, list] = defaultdict(list)
    for (i, j, s), var in x.items():
        vars_by_slot[s].append(var)

    for s in range(n_slots):
        slot_vars = vars_by_slot.get(s, [])
        if slot_vars:
            model.add(sum(slot_vars) <= 1)

    # --- Build objective ---
    obj_terms = variables["objective_terms"]
    if obj_terms:
        model.maximize(sum(obj_terms))

    # --- Solve ---
    solver = cp_model.CpSolver()
    solver.parameters.max_time_in_seconds = float(request.time_limit_seconds)
    solver.parameters.log_search_progress = False

    logger.info("Starting CP-SAT solve (time limit: %ds)", request.time_limit_seconds)
    status = solver.solve(model)

    status_map = {
        cp_model.OPTIMAL: "optimal",
        cp_model.FEASIBLE: "feasible",
        cp_model.INFEASIBLE: "infeasible",
        cp_model.UNKNOWN: "timeout",
        cp_model.MODEL_INVALID: "model_invalid",
    }
    status_str = status_map.get(status, "unknown")
    logger.info("Solver finished: %s (wall_time=%.2fs)", status_str, solver.wall_time)

    games: list[GameResult] = []
    if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
        for (i, j, s), var in x.items():
            if solver.value(var) == 1:
                fid, d_str, st, et = all_slots[s]
                ti = teams[i]
                tj = teams[j]
                games.append(GameResult(
                    home_team_id=ti.id,
                    away_team_id=tj.id,
                    field_id=fid,
                    game_date=date.fromisoformat(d_str),
                    start_time=st,
                    is_interleague=(
                        ti.team_type == "interleague" or tj.team_type == "interleague"
                    ),
                ))
        games.sort(key=lambda g: (str(g.game_date), str(g.start_time)))

    obj_val = None
    try:
        if status in (cp_model.OPTIMAL, cp_model.FEASIBLE):
            obj_val = solver.objective_value
    except Exception:
        pass

    return SolveResponse(
        status=status_str,
        games=games,
        solver_stats={
            "wall_time_s": round(solver.wall_time, 2),
            "num_conflicts": solver.num_conflicts,
            "num_branches": solver.num_branches,
            "num_games_scheduled": len(games),
            "objective_value": obj_val,
        },
        unmet_constraints=[],
    )
