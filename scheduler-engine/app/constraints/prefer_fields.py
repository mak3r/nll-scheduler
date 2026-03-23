"""
Field preference soft constraint.

PreferFields: adds objective bonus when games are scheduled at fields
preferred by each team's division. Uses division_preferred_fields from the
SolveRequest (a dict mapping division_id → list of preferred field IDs),
falling back to params if not set on the request.

This constraint is auto-injected by the schedule-service orchestrator when
any division has preferred field rules. It can also be added manually via the
season constraint config with explicit division_preferred_fields params.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class PreferFieldsConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "prefer_fields"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        # Prefer top-level SolveRequest field (auto-injected), fall back to params.
        div_preferred: dict[str, list[str]] = dict(
            getattr(problem, "division_preferred_fields", {}) or {}
        )
        if not div_preferred:
            div_preferred = dict(params.get("division_preferred_fields", {}) or {})
        if not div_preferred:
            return

        preferred_sets: dict[str, set[str]] = {
            div_id: set(fids) for div_id, fids in div_preferred.items()
        }

        bonus = int(params.get("bonus_per_game", 10))
        teams = variables["teams"]
        x = variables["x"]
        slots = variables["slots"]
        obj_terms = variables["objective_terms"]

        for (i, j, s_idx), var in x.items():
            fid = slots[s_idx][0]
            div_id = teams[i].division_id
            if fid in preferred_sets.get(div_id, set()):
                obj_terms.append(var * bonus)
