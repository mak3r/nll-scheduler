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

const seasonSelect = `
	SELECT s.id, s.name, s.start_date::text, s.end_date::text, s.status,
	       s.is_current, s.created_at, s.updated_at,
	       COALESCE(
	         array_agg(sd.division_id::text ORDER BY sd.division_id)
	           FILTER (WHERE sd.division_id IS NOT NULL),
	         ARRAY[]::text[]
	       ) AS division_ids
	FROM seasons s
	LEFT JOIN season_divisions sd ON sd.season_id = s.id`

func scanSeason(row pgx.Row) (*model.Season, error) {
	var s model.Season
	if err := row.Scan(
		&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.Status,
		&s.IsCurrent, &s.CreatedAt, &s.UpdatedAt, &s.DivisionIDs,
	); err != nil {
		return nil, err
	}
	if s.DivisionIDs == nil {
		s.DivisionIDs = []string{}
	}
	return &s, nil
}

func (r *SeasonRepo) List(ctx context.Context) ([]model.Season, error) {
	rows, err := r.db.Query(ctx,
		seasonSelect+` GROUP BY s.id ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []model.Season
	for rows.Next() {
		s, err := scanSeason(rows)
		if err != nil {
			return nil, err
		}
		seasons = append(seasons, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if seasons == nil {
		seasons = []model.Season{}
	}
	return seasons, nil
}

func (r *SeasonRepo) Create(ctx context.Context, s model.Season) (*model.Season, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO seasons (name, start_date, end_date, status)
		 VALUES ($1, $2, $3, 'draft')
		 RETURNING id`,
		s.Name, s.StartDate, s.EndDate,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err := insertDivisions(ctx, tx, id, s.DivisionIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.Get(ctx, id)
}

func (r *SeasonRepo) Get(ctx context.Context, id string) (*model.Season, error) {
	row := r.db.QueryRow(ctx,
		seasonSelect+` WHERE s.id=$1 GROUP BY s.id`,
		id,
	)
	s, err := scanSeason(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *SeasonRepo) Update(ctx context.Context, id string, s model.Season) (*model.Season, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	result, err := tx.Exec(ctx,
		`UPDATE seasons SET name=$1, start_date=$2, end_date=$3, updated_at=NOW()
		 WHERE id=$4`,
		s.Name, s.StartDate, s.EndDate, id,
	)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	if _, err := tx.Exec(ctx, `DELETE FROM season_divisions WHERE season_id=$1`, id); err != nil {
		return nil, err
	}
	if err := insertDivisions(ctx, tx, id, s.DivisionIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.Get(ctx, id)
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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO seasons (id, name, start_date, end_date, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE
		   SET name       = EXCLUDED.name,
		       start_date = EXCLUDED.start_date,
		       end_date   = EXCLUDED.end_date,
		       status     = EXCLUDED.status,
		       updated_at = EXCLUDED.updated_at`,
		s.ID, s.Name, s.StartDate, s.EndDate,
		s.Status, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM season_divisions WHERE season_id=$1`, s.ID); err != nil {
		return err
	}
	if err := insertDivisions(ctx, tx, s.ID, s.DivisionIDs); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SeasonRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE seasons SET status=$1, updated_at=NOW() WHERE id=$2`,
		status, id)
	return err
}

func (r *SeasonRepo) SetCurrentSeason(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `UPDATE seasons SET is_current = false WHERE is_current = true`); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE seasons SET is_current = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// insertDivisions bulk-inserts division_ids for a season within an existing transaction.
func insertDivisions(ctx context.Context, tx pgx.Tx, seasonID string, divisionIDs []string) error {
	for _, divID := range divisionIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO season_divisions (season_id, division_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			seasonID, divID,
		); err != nil {
			return err
		}
	}
	return nil
}
