"""
Rest day constraints.

MaxGamesPerTeamPerWeek: hard — a team plays at most N games per calendar week.
MinRestDaysBetweenGames: hard — a team must rest at least R days between games.
"""
from collections import defaultdict
from datetime import date
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class MaxGamesPerTeamPerWeekConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "max_games_per_team_per_week"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        teams = variables["teams"]
        n_teams = len(teams)
        slots_by_week: dict = variables["slots_by_week"]
        team_plays_at = variables["team_plays_at"]

        max_per_week = int(params.get("max_games_per_week", 2))

        for week, slot_indices in slots_by_week.items():
            for i in range(n_teams):
                plays_in_week = [v for s in slot_indices for v in team_plays_at(i, s)]
                if plays_in_week:
                    model.add(sum(plays_in_week) <= max_per_week)


class MinRestDaysBetweenGamesConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "min_rest_days_between_games"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        teams = variables["teams"]
        n_teams = len(teams)
        slots_by_date: dict = variables["slots_by_date"]
        slot_dates: list = variables["slot_dates"]
        team_plays_at = variables["team_plays_at"]

        min_rest = int(params.get("min_rest_days", 1))

        # Group slot indices by date for quick lookup
        sorted_dates = sorted(slots_by_date.keys())

        for i in range(n_teams):
            # For each date d1, for each date d2 within [d1+1, d1+min_rest]:
            # team i cannot play on both d1 and d2
            for idx1, d1_str in enumerate(sorted_dates):
                d1 = date.fromisoformat(d1_str)
                slots_d1 = slots_by_date[d1_str]
                plays_d1 = [v for s in slots_d1 for v in team_plays_at(i, s)]
                if not plays_d1:
                    continue

                for idx2 in range(idx1 + 1, len(sorted_dates)):
                    d2_str = sorted_dates[idx2]
                    d2 = date.fromisoformat(d2_str)
                    gap = (d2 - d1).days
                    if gap > min_rest:
                        break  # dates are sorted, no need to check further
                    slots_d2 = slots_by_date[d2_str]
                    plays_d2 = [v for s in slots_d2 for v in team_plays_at(i, s)]
                    if plays_d2:
                        # Cannot play on both d1 and d2
                        model.add(sum(plays_d1) + sum(plays_d2) <= 1)
