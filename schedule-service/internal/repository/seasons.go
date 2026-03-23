package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/schedule-service/internal/model"
)

type SeasonRepo struct{ db *pgxpool.Pool }

func NewSeasonRepo(db *pgxpool.Pool) *SeasonRepo { return &SeasonRepo{db: db} }

func (r *SeasonRepo) List(ctx context.Context) ([]model.Season, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, division_id, start_date::text, end_date::text, status, created_at, updated_at
		 FROM seasons ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []model.Season
	for rows.Next() {
		var s model.Season
		if err := rows.Scan(&s.ID, &s.Name, &s.DivisionID, &s.StartDate, &s.EndDate,
			&s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	if seasons == nil {
		seasons = []model.Season{}
	}
	return seasons, rows.Err()
}

func (r *SeasonRepo) Create(ctx context.Context, s model.Season) (*model.Season, error) {
	var out model.Season
	err := r.db.QueryRow(ctx,
		`INSERT INTO seasons (name, division_id, start_date, end_date, status)
		 VALUES ($1, $2, $3, $4, 'draft')
		 RETURNING id, name, division_id, start_date::text, end_date::text, status, created_at, updated_at`,
		s.Name, s.DivisionID, s.StartDate, s.EndDate,
	).Scan(&out.ID, &out.Name, &out.DivisionID, &out.StartDate, &out.EndDate,
		&out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *SeasonRepo) Get(ctx context.Context, id string) (*model.Season, error) {
	var s model.Season
	err := r.db.QueryRow(ctx,
		`SELECT id, name, division_id, start_date::text, end_date::text, status, created_at, updated_at
		 FROM seasons WHERE id=$1`,
		id,
	).Scan(&s.ID, &s.Name, &s.DivisionID, &s.StartDate, &s.EndDate,
		&s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *SeasonRepo) Update(ctx context.Context, id string, s model.Season) (*model.Season, error) {
	var out model.Season
	err := r.db.QueryRow(ctx,
		`UPDATE seasons SET name=$1, division_id=$2, start_date=$3, end_date=$4, updated_at=NOW()
		 WHERE id=$5
		 RETURNING id, name, division_id, start_date::text, end_date::text, status, created_at, updated_at`,
		s.Name, s.DivisionID, s.StartDate, s.EndDate, id,
	).Scan(&out.ID, &out.Name, &out.DivisionID, &out.StartDate, &out.EndDate,
		&out.Status, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *SeasonRepo) Delete(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM seasons WHERE id=$1 RETURNING id`, id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *SeasonRepo) Upsert(ctx context.Context, s model.Season) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO seasons (id, name, division_id, start_date, end_date, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO UPDATE
		   SET name        = EXCLUDED.name,
		       division_id = EXCLUDED.division_id,
		       start_date  = EXCLUDED.start_date,
		       end_date    = EXCLUDED.end_date,
		       status      = EXCLUDED.status,
		       updated_at  = EXCLUDED.updated_at`,
		s.ID, s.Name, s.DivisionID, s.StartDate, s.EndDate,
		s.Status, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *SeasonRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE seasons SET status=$1, updated_at=NOW() WHERE id=$2`,
		status, id)
	return err
}
