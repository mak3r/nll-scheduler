"""
Base constraint handler interface.
All scheduling constraints implement this interface.
To add a new constraint: subclass ConstraintHandler, implement apply(), register in __init__.py.
"""
from abc import ABC, abstractmethod
from typing import Any


class ConstraintHandler(ABC):
    """
    Abstract base for all scheduling constraint handlers.

    The apply() method is called during CP-SAT model construction.
    Hard constraints add model constraints directly.
    Soft constraints typically add to the objective function.
    """

    @property
    @abstractmethod
    def constraint_type(self) -> str:
        """Unique string identifier for this constraint type."""
        ...

    @abstractmethod
    def apply(
        self,
        model: Any,       # ortools.sat.python.cp_model.CpModel
        variables: dict,  # solver variable dict built in builder.py
        problem: Any,     # SolveRequest
        params: dict,     # constraint-specific params from season_constraints.params
    ) -> None:
        """
        Add constraints or objective terms to the CP-SAT model.

        Args:
            model: The CP-SAT model being constructed
            variables: Dict of solver variables (game assignment bools, etc.)
            problem: The full SolveRequest for context
            params: Constraint-specific parameters from JSONB storage
        """
        ...
