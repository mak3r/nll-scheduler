package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/field-service/internal/model"
)

type AvailabilityRepo struct {
	db *pgxpool.Pool
}

func NewAvailabilityRepo(db *pgxpool.Pool) *AvailabilityRepo {
	return &AvailabilityRepo{db: db}
}

func (r *AvailabilityRepo) ListWindows(ctx context.Context, fieldID string) ([]model.AvailabilityWindow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, field_id, window_type, days_of_week,
		        start_date::text, end_date::text, start_time::text, end_time::text, created_at
		 FROM availability_windows
		 WHERE field_id = $1
		 ORDER BY start_date`,
		fieldID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var windows []model.AvailabilityWindow
	for rows.Next() {
		var w model.AvailabilityWindow
		var daysOfWeek []int32
		if err := rows.Scan(
			&w.ID,
			&w.FieldID,
			&w.WindowType,
			&daysOfWeek,
			&w.StartDate,
			&w.EndDate,
			&w.StartTime,
			&w.EndTime,
			&w.CreatedAt,
		); err != nil {
			return nil, err
		}
		w.DaysOfWeek = make([]int, len(daysOfWeek))
		for i, d := range daysOfWeek {
			w.DaysOfWeek[i] = int(d)
		}
		windows = append(windows, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if windows == nil {
		windows = []model.AvailabilityWindow{}
	}
	return windows, nil
}

func (r *AvailabilityRepo) ListAllWindows(ctx context.Context) ([]model.AvailabilityWindow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, field_id, window_type, days_of_week,
		        start_date::text, end_date::text, start_time::text, end_time::text, created_at
		 FROM availability_windows
		 ORDER BY field_id, start_date`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var windows []model.AvailabilityWindow
	for rows.Next() {
		var w model.AvailabilityWindow
		var daysOfWeek []int32
		if err := rows.Scan(
			&w.ID, &w.FieldID, &w.WindowType, &daysOfWeek,
			&w.StartDate, &w.EndDate, &w.StartTime, &w.EndTime, &w.CreatedAt,
		); err != nil {
			return nil, err
		}
		w.DaysOfWeek = make([]int, len(daysOfWeek))
		for i, d := range daysOfWeek {
			w.DaysOfWeek[i] = int(d)
		}
		windows = append(windows, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if windows == nil {
		windows = []model.AvailabilityWindow{}
	}
	return windows, nil
}

func (r *AvailabilityRepo) UpsertWindow(ctx context.Context, w model.AvailabilityWindow) error {
	daysOfWeek := make([]int32, len(w.DaysOfWeek))
	for i, d := range w.DaysOfWeek {
		daysOfWeek[i] = int32(d)
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO availability_windows
		   (id, field_id, window_type, days_of_week, start_date, end_date, start_time, end_time, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO UPDATE
		   SET field_id     = EXCLUDED.field_id,
		       window_type  = EXCLUDED.window_type,
		       days_of_week = EXCLUDED.days_of_week,
		       start_date   = EXCLUDED.start_date,
		       end_date     = EXCLUDED.end_date,
		       start_time   = EXCLUDED.start_time,
		       end_time     = EXCLUDED.end_time`,
		w.ID, w.FieldID, w.WindowType, daysOfWeek,
		w.StartDate, w.EndDate, w.StartTime, w.EndTime, w.CreatedAt,
	)
	return err
}

func (r *AvailabilityRepo) ListAllBlackouts(ctx context.Context) ([]model.BlackoutDate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, field_id, blackout_date::text, reason, created_at
		 FROM blackout_dates
		 ORDER BY field_id, blackout_date`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blackouts []model.BlackoutDate
	for rows.Next() {
		var b model.BlackoutDate
		if err := rows.Scan(&b.ID, &b.FieldID, &b.BlackoutDate, &b.Reason, &b.CreatedAt); err != nil {
			return nil, err
		}
		blackouts = append(blackouts, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if blackouts == nil {
		blackouts = []model.BlackoutDate{}
	}
	return blackouts, nil
}

func (r *AvailabilityRepo) UpsertBlackout(ctx context.Context, b model.BlackoutDate) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO blackout_dates (id, field_id, blackout_date, reason, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE
		   SET field_id      = EXCLUDED.field_id,
		       blackout_date = EXCLUDED.blackout_date,
		       reason        = EXCLUDED.reason`,
		b.ID, b.FieldID, b.BlackoutDate, b.Reason, b.CreatedAt,
	)
	return err
}

func (r *AvailabilityRepo) CreateWindow(ctx context.Context, w model.AvailabilityWindow) (*model.AvailabilityWindow, error) {
	daysOfWeek := make([]int32, len(w.DaysOfWeek))
	for i, d := range w.DaysOfWeek {
		daysOfWeek[i] = int32(d)
	}

	var result model.AvailabilityWindow
	var resultDays []int32
	err := r.db.QueryRow(ctx,
		`INSERT INTO availability_windows (field_id, window_type, days_of_week, start_date, end_date, start_time, end_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, field_id, window_type, days_of_week,
		           start_date::text, end_date::text, start_time::text, end_time::text, created_at`,
		w.FieldID, w.WindowType, daysOfWeek, w.StartDate, w.EndDate, w.StartTime, w.EndTime,
	).Scan(
		&result.ID,
		&result.FieldID,
		&result.WindowType,
		&resultDays,
		&result.StartDate,
		&result.EndDate,
		&result.StartTime,
		&result.EndTime,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	result.DaysOfWeek = make([]int, len(resultDays))
	for i, d := range resultDays {
		result.DaysOfWeek[i] = int(d)
	}
	return &result, nil
}

func (r *AvailabilityRepo) UpdateWindow(ctx context.Context, w model.AvailabilityWindow) (*model.AvailabilityWindow, error) {
	daysOfWeek := make([]int32, len(w.DaysOfWeek))
	for i, d := range w.DaysOfWeek {
		daysOfWeek[i] = int32(d)
	}

	var result model.AvailabilityWindow
	var resultDays []int32
	err := r.db.QueryRow(ctx,
		`UPDATE availability_windows
		 SET window_type=$1, days_of_week=$2, start_date=$3, end_date=$4, start_time=$5, end_time=$6
		 WHERE id=$7
		 RETURNING id, field_id, window_type, days_of_week,
		           start_date::text, end_date::text, start_time::text, end_time::text, created_at`,
		w.WindowType, daysOfWeek, w.StartDate, w.EndDate, w.StartTime, w.EndTime, w.ID,
	).Scan(
		&result.ID, &result.FieldID, &result.WindowType, &resultDays,
		&result.StartDate, &result.EndDate, &result.StartTime, &result.EndTime, &result.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result.DaysOfWeek = make([]int, len(resultDays))
	for i, d := range resultDays {
		result.DaysOfWeek[i] = int(d)
	}
	return &result, nil
}

func (r *AvailabilityRepo) DeleteWindow(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM availability_windows WHERE id = $1 RETURNING id`,
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

func (r *AvailabilityRepo) ListBlackouts(ctx context.Context, fieldID string) ([]model.BlackoutDate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, field_id, blackout_date::text, reason, created_at
		 FROM blackout_dates
		 WHERE field_id = $1
		 ORDER BY blackout_date`,
		fieldID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blackouts []model.BlackoutDate
	for rows.Next() {
		var b model.BlackoutDate
		if err := rows.Scan(
			&b.ID,
			&b.FieldID,
			&b.BlackoutDate,
			&b.Reason,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}
		blackouts = append(blackouts, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if blackouts == nil {
		blackouts = []model.BlackoutDate{}
	}
	return blackouts, nil
}

func (r *AvailabilityRepo) CreateBlackout(ctx context.Context, b model.BlackoutDate) (*model.BlackoutDate, error) {
	var result model.BlackoutDate
	err := r.db.QueryRow(ctx,
		`INSERT INTO blackout_dates (field_id, blackout_date, reason)
		 VALUES ($1, $2, $3)
		 RETURNING id, field_id, blackout_date::text, reason, created_at`,
		b.FieldID, b.BlackoutDate, b.Reason,
	).Scan(
		&result.ID,
		&result.FieldID,
		&result.BlackoutDate,
		&result.Reason,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *AvailabilityRepo) DeleteBlackout(ctx context.Context, id string) error {
	var deletedID string
	err := r.db.QueryRow(ctx,
		`DELETE FROM blackout_dates WHERE id = $1 RETURNING id`,
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
