"""
Interleague date preference soft constraint.

PreferInterleagueDates: adds objective bonus when interleague games
fall on preferred interleague dates. Weight scales the bonus.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class PreferInterleagueDatesConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "prefer_interleague_dates"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        if not problem.preferred_interleague_dates:
            return

        teams = variables["teams"]
        n_teams = len(teams)
        x = variables["x"]
        slots = variables["slots"]

        preferred_dates: set[str] = {d.isoformat() for d in problem.preferred_interleague_dates}
        interleague_team_indices = {
            i for i, t in enumerate(teams) if t.team_type == "interleague"
        }

        # Bonus weight (scaled to integer for CP-SAT)
        bonus = int(params.get("bonus_per_game", 10))
        obj_terms = variables["objective_terms"]

        for s_idx, (fid, date_str, st, et) in enumerate(slots):
            if date_str not in preferred_dates:
                continue
            for i in range(n_teams):
                for j in range(n_teams):
                    if i == j:
                        continue
                    # Only interleague games get the bonus
                    if i in interleague_team_indices or j in interleague_team_indices:
                        obj_terms.append(x[i, j, s_idx] * bonus)
