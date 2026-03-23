package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
)

type GenerationRunsRepo struct{ db *pgxpool.Pool }

func NewGenerationRunsRepo(db *pgxpool.Pool) *GenerationRunsRepo {
	return &GenerationRunsRepo{db: db}
}

func (r *GenerationRunsRepo) Create(ctx context.Context, seasonID string) (*model.GenerationRun, error) {
	var run model.GenerationRun
	var solverStats []byte
	err := r.db.QueryRow(ctx,
		`INSERT INTO generation_runs (season_id, status)
		 VALUES ($1, 'pending')
		 RETURNING id, season_id, status, solver_stats, error_message, created_at, updated_at`,
		seasonID,
	).Scan(&run.ID, &run.SeasonID, &run.Status, &solverStats, &run.ErrorMessage,
		&run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if solverStats != nil {
		run.SolverStats = solverStats
	}
	return &run, nil
}

func (r *GenerationRunsRepo) Get(ctx context.Context, id string) (*model.GenerationRun, error) {
	var run model.GenerationRun
	var solverStats []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, season_id, status, solver_stats, error_message, created_at, updated_at
		 FROM generation_runs WHERE id=$1`,
		id,
	).Scan(&run.ID, &run.SeasonID, &run.Status, &solverStats, &run.ErrorMessage,
		&run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if solverStats != nil {
		run.SolverStats = solverStats
	}
	return &run, nil
}

func (r *GenerationRunsRepo) UpdateStatus(ctx context.Context, id, status string, solverStats []byte, errMsg *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE generation_runs
		 SET status=$1, solver_stats=$2::jsonb, error_message=$3, updated_at=NOW()
		 WHERE id=$4`,
		status, solverStats, errMsg, id)
	return err
}
