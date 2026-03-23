package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/team-service/internal/model"
)

type DivisionRepo struct {
	db *pgxpool.Pool
}

func NewDivisionRepo(db *pgxpool.Pool) *DivisionRepo {
	return &DivisionRepo{db: db}
}

func (r *DivisionRepo) List(ctx context.Context) ([]model.Division, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, season_year, created_at, updated_at
		 FROM divisions
		 ORDER BY season_year DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var divisions []model.Division
	for rows.Next() {
		var d model.Division
		if err := rows.Scan(&d.ID, &d.Name, &d.SeasonYear, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		divisions = append(divisions, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if divisions == nil {
		divisions = []model.Division{}
	}
	return divisions, nil
}

func (r *DivisionRepo) Create(ctx context.Context, name string, seasonYear int) (*model.Division, error) {
	var d model.Division
	err := r.db.QueryRow(ctx,
		`INSERT INTO divisions (name, season_year)
		 VALUES ($1, $2)
		 RETURNING id, name, season_year, created_at, updated_at`,
		name, seasonYear,
	).Scan(&d.ID, &d.Name, &d.SeasonYear, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Get(ctx context.Context, id string) (*model.Division, error) {
	var d model.Division
	err := r.db.QueryRow(ctx,
		`SELECT id, name, season_year, created_at, updated_at
		 FROM divisions WHERE id = $1`,
		id,
	).Scan(&d.ID, &d.Name, &d.SeasonYear, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Update(ctx context.Context, id string, name string, seasonYear int) (*model.Division, error) {
	var d model.Division
	err := r.db.QueryRow(ctx,
		`UPDATE divisions
		 SET name = $1, season_year = $2, updated_at = NOW()
		 WHERE id = $3
		 RETURNING id, name, season_year, created_at, updated_at`,
		name, seasonYear, id,
	).Scan(&d.ID, &d.Name, &d.SeasonYear, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM divisions WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
