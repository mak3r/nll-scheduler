"""
Round-robin matchup constraint.

Hard constraint: each team pair (i, j) plays exactly the required number of games.
- If a matchup_rule exists for the pair, use (min_games, max_games) from that rule.
- Otherwise, default to exactly 2 games (1 home, 1 away) per pair.

The solver decides which team is home/away for each game.
"""
from typing import Any

from ortools.sat.python import cp_model

from app.constraints.base import ConstraintHandler


class RoundRobinMatchupConstraint(ConstraintHandler):
    @property
    def constraint_type(self) -> str:
        return "round_robin_matchup"

    def apply(
        self,
        model: cp_model.CpModel,
        variables: dict,
        problem: Any,
        params: dict,
    ) -> None:
        teams = variables["teams"]
        n_teams = len(teams)
        matchup_limits: dict = variables["matchup_limits"]
        pair_game_vars = variables["pair_game_vars"]

        default_games_per_pair = int(params.get("default_games_per_pair", 2))

        for i in range(n_teams):
            for j in range(i + 1, n_teams):
                pair = (i, j)
                min_g, max_g = matchup_limits.get(pair, (default_games_per_pair, default_games_per_pair))

                game_vars = pair_game_vars(i, j)
                total = sum(game_vars)
                model.add(total >= min_g)
                model.add(total <= max_g)
