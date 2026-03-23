"""
Pydantic schemas for the /solve endpoint.
These define the contract between schedule-service and scheduler-engine.
"""
from __future__ import annotations

from datetime import date, time
from pydantic import BaseModel, Field


class TeamSchema(BaseModel):
    id: str
    name: str
    division_id: str
    team_type: str  # "local" | "interleague"
    games_required: int = 20


class MatchupRule(BaseModel):
    team_a_id: str
    team_b_id: str
    min_games: int = 1
    max_games: int = 3


class FieldSlot(BaseModel):
    """A concrete available time slot at a field."""
    field_id: str
    date: date
    start_time: time
    end_time: time


class FieldSchema(BaseModel):
    id: str
    name: str
    max_games_per_day: int = 4
    available_slots: list[FieldSlot] = Field(default_factory=list)


class ConstraintConfig(BaseModel):
    type: str
    params: dict = Field(default_factory=dict)
    is_hard: bool = True
    weight: float = 1.0


class SolveRequest(BaseModel):
    season_id: str
    start_date: date
    end_date: date
    teams: list[TeamSchema]
    matchup_rules: list[MatchupRule] = Field(default_factory=list)
    fields: list[FieldSchema]
    blackout_dates: list[date] = Field(default_factory=list)
    preferred_interleague_dates: list[date] = Field(default_factory=list)
    constraints: list[ConstraintConfig] = Field(default_factory=list)
    time_limit_seconds: int = 60
    division_field_restrictions: dict[str, list[str]] = Field(default_factory=dict)
    division_preferred_fields: dict[str, list[str]] = Field(default_factory=dict)


class GameResult(BaseModel):
    home_team_id: str
    away_team_id: str
    field_id: str
    game_date: date
    start_time: time
    is_interleague: bool = False


class SolveResponse(BaseModel):
    status: str  # "optimal" | "feasible" | "infeasible" | "timeout" | "stub"
    games: list[GameResult] = Field(default_factory=list)
    solver_stats: dict = Field(default_factory=dict)
    unmet_constraints: list[str] = Field(default_factory=list)
