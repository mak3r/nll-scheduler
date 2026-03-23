"""
Constraint Registry — maps constraint type strings to handler instances.

To register a new constraint:
1. Create a new file in this directory implementing ConstraintHandler
2. Import it here and add to REGISTRY

This is the only file that needs to change when adding new constraints.
"""
from app.constraints.balance import EvenHomeAwayBalanceConstraint
from app.constraints.base import ConstraintHandler
from app.constraints.capacity import MaxGamesPerFieldPerDayConstraint
from app.constraints.interleague import PreferInterleagueDatesConstraint
from app.constraints.prefer_fields import PreferFieldsConstraint
from app.constraints.rest import MaxGamesPerTeamPerWeekConstraint, MinRestDaysBetweenGamesConstraint
from app.constraints.roundrobin import RoundRobinMatchupConstraint

REGISTRY: dict[str, ConstraintHandler] = {
    "round_robin_matchup": RoundRobinMatchupConstraint(),
    "max_games_per_field_per_day": MaxGamesPerFieldPerDayConstraint(),
    "max_games_per_team_per_week": MaxGamesPerTeamPerWeekConstraint(),
    "min_rest_days_between_games": MinRestDaysBetweenGamesConstraint(),
    "prefer_interleague_dates": PreferInterleagueDatesConstraint(),
    "even_home_away_balance": EvenHomeAwayBalanceConstraint(),
    "prefer_fields": PreferFieldsConstraint(),
}


def get_handler(constraint_type: str) -> ConstraintHandler:
    handler = REGISTRY.get(constraint_type)
    if handler is None:
        raise ValueError(
            f"Unknown constraint type: {constraint_type!r}. "
            f"Available: {list(REGISTRY.keys())}"
        )
    return handler
