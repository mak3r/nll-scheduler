package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
)

type DivisionGamesRequiredRepo struct {
	db *pgxpool.Pool
}

func NewDivisionGamesRequiredRepo(db *pgxpool.Pool) *DivisionGamesRequiredRepo {
	return &DivisionGamesRequiredRepo{db: db}
}

func (r *DivisionGamesRequiredRepo) Get(ctx context.Context, seasonID, divisionID string) (*model.DivisionGamesRequired, error) {
	var d model.DivisionGamesRequired
	err := r.db.QueryRow(ctx,
		`SELECT id, season_id, division_id, games_required, created_at, updated_at
		 FROM division_games_required
		 WHERE season_id = $1 AND division_id = $2`,
		seasonID, divisionID,
	).Scan(&d.ID, &d.SeasonID, &d.DivisionID, &d.GamesRequired, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DivisionGamesRequiredRepo) ListBySeason(ctx context.Context, seasonID string) ([]model.DivisionGamesRequired, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, season_id, division_id, games_required, created_at, updated_at
		 FROM division_games_required
		 WHERE season_id = $1
		 ORDER BY division_id`,
		seasonID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.DivisionGamesRequired
	for rows.Next() {
		var d model.DivisionGamesRequired
		if err := rows.Scan(&d.ID, &d.SeasonID, &d.DivisionID, &d.GamesRequired, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []model.DivisionGamesRequired{}
	}
	return result, nil
}

func (r *DivisionGamesRequiredRepo) Upsert(ctx context.Context, seasonID, divisionID string, gamesRequired int) (*model.DivisionGamesRequired, error) {
	var d model.DivisionGamesRequired
	err := r.db.QueryRow(ctx,
		`INSERT INTO division_games_required (season_id, division_id, games_required)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (season_id, division_id) DO UPDATE
		   SET games_required = EXCLUDED.games_required,
		       updated_at     = NOW()
		 RETURNING id, season_id, division_id, games_required, created_at, updated_at`,
		seasonID, divisionID, gamesRequired,
	).Scan(&d.ID, &d.SeasonID, &d.DivisionID, &d.GamesRequired, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
