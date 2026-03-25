"""
Home field preference soft constraint.

PreferHomeField: adds objective bonus when the home team plays at their
designated home_field_id. This constraint is auto-injected by the
schedule-service orchestrator when any team has a home_field_id set.

It is always soft — it never blocks scheduling if the home field is
unavailable or restricted.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class PreferHomeFieldConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "prefer_home_field"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        # team_home_fields: {team_id: field_id}
        team_home_fields: dict[str, str] = dict(params.get("team_home_fields", {}) or {})
        if not team_home_fields:
            return

        bonus = int(params.get("bonus_per_game", 10))
        teams = variables["teams"]
        x = variables["x"]
        slots = variables["slots"]
        obj_terms = variables["objective_terms"]

        for (i, j, s_idx), var in x.items():
            fid = slots[s_idx][0]
            home_team = teams[i]
            # Award bonus when the home team plays at their home field
            if team_home_fields.get(home_team.id) == fid:
                obj_terms.append(var * bonus)
