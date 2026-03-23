import logging

from fastapi import FastAPI, HTTPException

from app.schemas.solve import SolveRequest, SolveResponse
from app.solver import builder

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="NLL Scheduler Engine", version="0.1.0")


@app.get("/health")
async def health():
    return {"status": "ok", "service": "scheduler-engine"}


@app.post("/solve", response_model=SolveResponse)
async def solve(request: SolveRequest):
    """
    Accepts a scheduling problem and returns a solution via CP-SAT.
    """
    logger.info(
        "Received solve request: season=%s, %d teams, %d fields, time_limit=%ds",
        request.season_id,
        len(request.teams),
        len(request.fields),
        request.time_limit_seconds,
    )
    try:
        response = builder.solve(request)
    except Exception as exc:
        logger.exception("Solver error")
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    logger.info("Solve complete: status=%s, games=%d", response.status, len(response.games))
    return response
