"""
Field and team capacity constraints.

MaxGamesPerFieldPerDay: hard — a field can host at most max_games_per_day games per day.
"""
from collections import defaultdict
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class MaxGamesPerFieldPerDayConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "max_games_per_field_per_day"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        teams = variables["teams"]
        n_teams = len(teams)
        x = variables["x"]
        slots_by_field_date: dict = variables["slots_by_field_date"]
        field_map = variables["field_map"]

        for (fid, date_str), slot_indices in slots_by_field_date.items():
            field = field_map.get(fid)
            if field is None:
                continue
            override = params.get("max_games_per_day")
            limit = int(override) if override else field.max_games_per_day

            # Sum all games at this field on this date
            games_at = [
                x[i, j, s]
                for s in slot_indices
                for i in range(n_teams)
                for j in range(n_teams)
                if i != j
            ]
            if games_at:
                model.add(sum(games_at) <= limit)
