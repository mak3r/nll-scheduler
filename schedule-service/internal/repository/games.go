package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
)

type GamesRepo struct{ db *pgxpool.Pool }

func NewGamesRepo(db *pgxpool.Pool) *GamesRepo { return &GamesRepo{db: db} }

func (r *GamesRepo) List(ctx context.Context, seasonID string) ([]model.Game, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, season_id, home_team_id, away_team_id, field_id,
		        game_date::text, start_time::text, status, is_interleague, manually_edited,
		        created_at, updated_at
		 FROM games WHERE season_id=$1 ORDER BY game_date, start_time`,
		seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []model.Game
	for rows.Next() {
		var g model.Game
		if err := rows.Scan(
			&g.ID, &g.SeasonID, &g.HomeTeamID, &g.AwayTeamID, &g.FieldID,
			&g.GameDate, &g.StartTime, &g.Status, &g.IsInterleague, &g.ManuallyEdited,
			&g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, err
		}
		games = append(games, g)
	}
	if games == nil {
		games = []model.Game{}
	}
	return games, rows.Err()
}

func (r *GamesRepo) Create(ctx context.Context, g model.Game) (*model.Game, error) {
	var out model.Game
	err := r.db.QueryRow(ctx,
		`INSERT INTO games (season_id, home_team_id, away_team_id, field_id, game_date, start_time, status, is_interleague, manually_edited)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, season_id, home_team_id, away_team_id, field_id,
		           game_date::text, start_time::text, status, is_interleague, manually_edited,
		           created_at, updated_at`,
		g.SeasonID, g.HomeTeamID, g.AwayTeamID, g.FieldID,
		g.GameDate, g.StartTime, g.Status, g.IsInterleague, g.ManuallyEdited,
	).Scan(
		&out.ID, &out.SeasonID, &out.HomeTeamID, &out.AwayTeamID, &out.FieldID,
		&out.GameDate, &out.StartTime, &out.Status, &out.IsInterleague, &out.ManuallyEdited,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *GamesRepo) Get(ctx context.Context, id string) (*model.Game, error) {
	var g model.Game
	err := r.db.QueryRow(ctx,
		`SELECT id, season_id, home_team_id, away_team_id, field_id,
		        game_date::text, start_time::text, status, is_interleague, manually_edited,
		        created_at, updated_at
		 FROM games WHERE id=$1`,
		id,
	).Scan(
		&g.ID, &g.SeasonID, &g.HomeTeamID, &g.AwayTeamID, &g.FieldID,
		&g.GameDate, &g.StartTime, &g.Status, &g.IsInterleague, &g.ManuallyEdited,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *GamesRepo) Update(ctx context.Context, id string, g model.Game) (*model.Game, error) {
	var out model.Game
	err := r.db.QueryRow(ctx,
		`UPDATE games
		 SET home_team_id=$1, away_team_id=$2, field_id=$3, game_date=$4, start_time=$5,
		     status=$6, is_interleague=$7, manually_edited=true, updated_at=NOW()
		 WHERE id=$8
		 RETURNING id, season_id, home_team_id, away_team_id, field_id,
		           game_date::text, start_time::text, status, is_interleague, manually_edited,
		           created_at, updated_at`,
		g.HomeTeamID, g.AwayTeamID, g.FieldID, g.GameDate, g.StartTime,
		g.Status, g.IsInterleague, id,
	).Scan(
		&out.ID, &out.SeasonID, &out.HomeTeamID, &out.AwayTeamID, &out.FieldID,
		&out.GameDate, &out.StartTime, &out.Status, &out.IsInterleague, &out.ManuallyEdited,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *GamesRepo) Delete(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM games WHERE id=$1 RETURNING id`, id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *GamesRepo) BulkCreate(ctx context.Context, games []model.Game) error {
	for _, g := range games {
		if _, err := r.Create(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

func (r *GamesRepo) DeleteBySeason(ctx context.Context, seasonID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM games WHERE season_id=$1`, seasonID)
	return err
}
