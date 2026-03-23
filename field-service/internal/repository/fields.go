package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/field-service/internal/model"
)

type FieldRepo struct {
	db *pgxpool.Pool
}

func NewFieldRepo(db *pgxpool.Pool) *FieldRepo {
	return &FieldRepo{db: db}
}

func (r *FieldRepo) List(ctx context.Context) ([]model.Field, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, address, max_games_per_day, is_active, created_at, updated_at
		 FROM fields
		 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []model.Field
	for rows.Next() {
		var f model.Field
		if err := rows.Scan(
			&f.ID,
			&f.Name,
			&f.Address,
			&f.MaxGamesPerDay,
			&f.IsActive,
			&f.CreatedAt,
			&f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = []model.Field{}
	}
	return fields, nil
}

func (r *FieldRepo) Create(ctx context.Context, f model.Field) (*model.Field, error) {
	var result model.Field
	err := r.db.QueryRow(ctx,
		`INSERT INTO fields (name, address, max_games_per_day, is_active)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, address, max_games_per_day, is_active, created_at, updated_at`,
		f.Name, f.Address, f.MaxGamesPerDay, f.IsActive,
	).Scan(
		&result.ID,
		&result.Name,
		&result.Address,
		&result.MaxGamesPerDay,
		&result.IsActive,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *FieldRepo) Get(ctx context.Context, id string) (*model.Field, error) {
	var f model.Field
	err := r.db.QueryRow(ctx,
		`SELECT id, name, address, max_games_per_day, is_active, created_at, updated_at
		 FROM fields
		 WHERE id = $1`,
		id,
	).Scan(
		&f.ID,
		&f.Name,
		&f.Address,
		&f.MaxGamesPerDay,
		&f.IsActive,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FieldRepo) Update(ctx context.Context, id string, f model.Field) (*model.Field, error) {
	var result model.Field
	err := r.db.QueryRow(ctx,
		`UPDATE fields
		 SET name=$1, address=$2, max_games_per_day=$3, is_active=$4, updated_at=NOW()
		 WHERE id=$5
		 RETURNING id, name, address, max_games_per_day, is_active, created_at, updated_at`,
		f.Name, f.Address, f.MaxGamesPerDay, f.IsActive, id,
	).Scan(
		&result.ID,
		&result.Name,
		&result.Address,
		&result.MaxGamesPerDay,
		&result.IsActive,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *FieldRepo) Delete(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`UPDATE fields SET is_active=false, updated_at=NOW() WHERE id=$1 RETURNING id`,
		id,
	).Scan(&deletedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
