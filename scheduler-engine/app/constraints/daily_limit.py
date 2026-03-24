"""
Daily limit constraints.

NoSameDayRepeatMatchup: hard — the same two teams cannot play each other
    more than once on the same calendar day.

PreferMaxOneGamePerDay: soft — penalizes any team that plays more than one
    game per day (double-headers). Weight defaults to 10.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class NoSameDayRepeatMatchupConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "no_same_day_repeat_matchup"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        x = variables["x"]
        teams = variables["teams"]
        n_teams = len(teams)
        slots_by_date = variables["slots_by_date"]

        for d_str, slot_indices in slots_by_date.items():
            for i in range(n_teams):
                for j in range(i + 1, n_teams):
                    pair_vars = []
                    for s in slot_indices:
                        if (i, j, s) in x:
                            pair_vars.append(x[i, j, s])
                        if (j, i, s) in x:
                            pair_vars.append(x[j, i, s])
                    if len(pair_vars) > 1:
                        model.add(sum(pair_vars) <= 1)


class PreferMaxOneGamePerDayConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "prefer_max_one_game_per_day"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        teams = variables["teams"]
        n_teams = len(teams)
        slots_by_date = variables["slots_by_date"]
        team_plays_at = variables["team_plays_at"]
        obj_terms = variables["objective_terms"]

        weight = int(params.get("weight", 10))

        for d_str, slot_indices in slots_by_date.items():
            for i in range(n_teams):
                plays = [v for s in slot_indices for v in team_plays_at(i, s)]
                if len(plays) > 1:
                    # excess >= plays_today - 1; penalise each game above 1
                    excess = model.new_int_var(0, len(plays) - 1, f"dh_{i}_{d_str}")
                    model.add(sum(plays) <= 1 + excess)
                    obj_terms.append(-weight * excess)
