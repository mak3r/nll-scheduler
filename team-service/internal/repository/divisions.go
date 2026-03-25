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

func scanDivisionRow(scanner interface{ Scan(...interface{}) error }) (model.Division, error) {
	var d model.Division
	var seasonID *string
	err := scanner.Scan(&d.ID, &d.Name, &d.SeasonYear, &seasonID, &d.CreatedAt, &d.UpdatedAt)
	if seasonID != nil {
		d.SeasonID = *seasonID
	}
	return d, err
}

func (r *DivisionRepo) List(ctx context.Context, seasonID string) ([]model.Division, error) {
	query := `SELECT id, name, season_year, season_id, created_at, updated_at FROM divisions`
	args := []interface{}{}
	if seasonID != "" {
		query += ` WHERE season_id = $1`
		args = append(args, seasonID)
	}
	query += ` ORDER BY season_year DESC, name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var divisions []model.Division
	for rows.Next() {
		d, err := scanDivisionRow(rows)
		if err != nil {
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

func (r *DivisionRepo) Create(ctx context.Context, name string, seasonYear int, seasonID string) (*model.Division, error) {
	var sid *string
	if seasonID != "" {
		sid = &seasonID
	}
	row := r.db.QueryRow(ctx,
		`INSERT INTO divisions (name, season_year, season_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, season_year, season_id, created_at, updated_at`,
		name, seasonYear, sid,
	)
	d, err := scanDivisionRow(row)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Get(ctx context.Context, id string) (*model.Division, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, name, season_year, season_id, created_at, updated_at FROM divisions WHERE id = $1`,
		id,
	)
	d, err := scanDivisionRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Update(ctx context.Context, id string, name string, seasonYear int, seasonID string) (*model.Division, error) {
	var sid *string
	if seasonID != "" {
		sid = &seasonID
	}
	row := r.db.QueryRow(ctx,
		`UPDATE divisions
		 SET name = $1, season_year = $2, season_id = $3, updated_at = NOW()
		 WHERE id = $4
		 RETURNING id, name, season_year, season_id, created_at, updated_at`,
		name, seasonYear, sid, id,
	)
	d, err := scanDivisionRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *DivisionRepo) Upsert(ctx context.Context, d model.Division) error {
	var sid *string
	if d.SeasonID != "" {
		sid = &d.SeasonID
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO divisions (id, name, season_year, season_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE
		   SET name        = EXCLUDED.name,
		       season_year = EXCLUDED.season_year,
		       season_id   = EXCLUDED.season_id,
		       updated_at  = EXCLUDED.updated_at`,
		d.ID, d.Name, d.SeasonYear, sid, d.CreatedAt, d.UpdatedAt,
	)
	return err
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
