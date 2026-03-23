package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
)

type SeasonExtrasRepo struct{ db *pgxpool.Pool }

func NewSeasonExtrasRepo(db *pgxpool.Pool) *SeasonExtrasRepo {
	return &SeasonExtrasRepo{db: db}
}

// Blackouts

func (r *SeasonExtrasRepo) ListBlackouts(ctx context.Context, seasonID string) ([]model.SeasonBlackout, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, season_id, blackout_date::text, created_at
		 FROM season_blackout_dates WHERE season_id=$1 ORDER BY blackout_date`,
		seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blackouts []model.SeasonBlackout
	for rows.Next() {
		var b model.SeasonBlackout
		if err := rows.Scan(&b.ID, &b.SeasonID, &b.BlackoutDate, &b.CreatedAt); err != nil {
			return nil, err
		}
		blackouts = append(blackouts, b)
	}
	if blackouts == nil {
		blackouts = []model.SeasonBlackout{}
	}
	return blackouts, rows.Err()
}

func (r *SeasonExtrasRepo) CreateBlackout(ctx context.Context, seasonID, date string) (*model.SeasonBlackout, error) {
	var b model.SeasonBlackout
	err := r.db.QueryRow(ctx,
		`INSERT INTO season_blackout_dates (season_id, blackout_date)
		 VALUES ($1, $2)
		 RETURNING id, season_id, blackout_date::text, created_at`,
		seasonID, date,
	).Scan(&b.ID, &b.SeasonID, &b.BlackoutDate, &b.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *SeasonExtrasRepo) DeleteBlackout(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM season_blackout_dates WHERE id=$1 RETURNING id`, id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Preferred dates

func (r *SeasonExtrasRepo) ListPreferredDates(ctx context.Context, seasonID string) ([]model.PreferredDate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, season_id, preferred_date::text, weight, created_at
		 FROM preferred_interleague_dates WHERE season_id=$1 ORDER BY preferred_date`,
		seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []model.PreferredDate
	for rows.Next() {
		var d model.PreferredDate
		if err := rows.Scan(&d.ID, &d.SeasonID, &d.PreferredDate, &d.Weight, &d.CreatedAt); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	if dates == nil {
		dates = []model.PreferredDate{}
	}
	return dates, rows.Err()
}

func (r *SeasonExtrasRepo) CreatePreferredDate(ctx context.Context, seasonID, date string, weight float64) (*model.PreferredDate, error) {
	var d model.PreferredDate
	err := r.db.QueryRow(ctx,
		`INSERT INTO preferred_interleague_dates (season_id, preferred_date, weight)
		 VALUES ($1, $2, $3)
		 RETURNING id, season_id, preferred_date::text, weight, created_at`,
		seasonID, date, weight,
	).Scan(&d.ID, &d.SeasonID, &d.PreferredDate, &d.Weight, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *SeasonExtrasRepo) DeletePreferredDate(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM preferred_interleague_dates WHERE id=$1 RETURNING id`, id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Constraints

func (r *SeasonExtrasRepo) ListConstraints(ctx context.Context, seasonID string) ([]model.SeasonConstraint, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, season_id, type, params, is_hard, weight, created_at
		 FROM season_constraints WHERE season_id=$1 ORDER BY created_at`,
		seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []model.SeasonConstraint
	for rows.Next() {
		var c model.SeasonConstraint
		var params []byte
		if err := rows.Scan(&c.ID, &c.SeasonID, &c.Type, &params, &c.IsHard, &c.Weight, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Params = params
		constraints = append(constraints, c)
	}
	if constraints == nil {
		constraints = []model.SeasonConstraint{}
	}
	return constraints, rows.Err()
}

func (r *SeasonExtrasRepo) CreateConstraint(ctx context.Context, c model.SeasonConstraint) (*model.SeasonConstraint, error) {
	var out model.SeasonConstraint
	var params []byte
	err := r.db.QueryRow(ctx,
		`INSERT INTO season_constraints (season_id, type, params, is_hard, weight)
		 VALUES ($1, $2, $3::jsonb, $4, $5)
		 RETURNING id, season_id, type, params, is_hard, weight, created_at`,
		c.SeasonID, c.Type, []byte(c.Params), c.IsHard, c.Weight,
	).Scan(&out.ID, &out.SeasonID, &out.Type, &params, &out.IsHard, &out.Weight, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	out.Params = params
	return &out, nil
}

func (r *SeasonExtrasRepo) UpdateConstraint(ctx context.Context, id string, c model.SeasonConstraint) (*model.SeasonConstraint, error) {
	var out model.SeasonConstraint
	var params []byte
	err := r.db.QueryRow(ctx,
		`UPDATE season_constraints SET type=$1, params=$2::jsonb, is_hard=$3, weight=$4
		 WHERE id=$5
		 RETURNING id, season_id, type, params, is_hard, weight, created_at`,
		c.Type, []byte(c.Params), c.IsHard, c.Weight, id,
	).Scan(&out.ID, &out.SeasonID, &out.Type, &params, &out.IsHard, &out.Weight, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	out.Params = params
	return &out, nil
}

func (r *SeasonExtrasRepo) DeleteConstraint(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM season_constraints WHERE id=$1 RETURNING id`, id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
