"""
Home/away balance soft constraint.

EvenHomeAwayBalance: adds objective bonus for teams that have balanced
home and away game counts. Uses auxiliary variables to model |home - away|.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class EvenHomeAwayBalanceConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "even_home_away_balance"

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
        slots = variables["slots"]
        n_slots = len(slots)
        obj_terms = variables["objective_terms"]

        # Estimate max imbalance for variable bounds
        # Max games a team can play = n_slots (one per slot, loose upper bound)
        max_games = n_slots

        penalty_weight = int(params.get("penalty_weight", 5))

        for i in range(n_teams):
            # home_i = total home games for team i
            home_i = sum(x[i, j, s] for j in range(n_teams) if i != j for s in range(n_slots) if (i, j, s) in x)
            # away_i = total away games for team i
            away_i = sum(x[j, i, s] for j in range(n_teams) if i != j for s in range(n_slots) if (j, i, s) in x)

            # diff = home - away (can be negative)
            # We want to minimize |diff| = minimize imbalance
            # Model: diff_abs >= diff, diff_abs >= -diff
            diff_abs = model.new_int_var(0, max_games, f"diff_abs_{i}")
            diff = model.new_int_var(-max_games, max_games, f"diff_{i}")
            model.add(diff == home_i - away_i)
            model.add(diff_abs >= diff)
            model.add(diff_abs >= -diff)

            # Add negative penalty to objective (maximizing, so subtract imbalance)
            obj_terms.append(-penalty_weight * diff_abs)
