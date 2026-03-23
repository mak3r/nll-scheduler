package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/team-service/internal/model"
)

type TeamRepo struct {
	db *pgxpool.Pool
}

func NewTeamRepo(db *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{db: db}
}

func scanTeam(row pgx.Row) (*model.Team, error) {
	var t model.Team
	if err := row.Scan(
		&t.ID, &t.DivisionID, &t.Name, &t.ShortCode,
		&t.TeamType, &t.HomeFieldID, &t.GamesRequired,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}

func scanTeamRow(rows pgx.Rows) (model.Team, error) {
	var t model.Team
	if err := rows.Scan(
		&t.ID, &t.DivisionID, &t.Name, &t.ShortCode,
		&t.TeamType, &t.HomeFieldID, &t.GamesRequired,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return model.Team{}, err
	}
	return t, nil
}

const teamColumns = `id, division_id, name, short_code, team_type, home_field_id, games_required, created_at, updated_at`

func (r *TeamRepo) List(ctx context.Context, divisionID string) ([]model.Team, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if divisionID != "" {
		rows, err = r.db.Query(ctx,
			fmt.Sprintf(`SELECT %s FROM teams WHERE division_id = $1 ORDER BY name`, teamColumns),
			divisionID,
		)
	} else {
		rows, err = r.db.Query(ctx,
			fmt.Sprintf(`SELECT %s FROM teams ORDER BY name`, teamColumns),
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		t, err := scanTeamRow(rows)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []model.Team{}
	}
	return teams, nil
}

func (r *TeamRepo) Create(ctx context.Context, t model.Team) (*model.Team, error) {
	row := r.db.QueryRow(ctx,
		fmt.Sprintf(`INSERT INTO teams (division_id, name, short_code, team_type, home_field_id, games_required)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING %s`, teamColumns),
		t.DivisionID, t.Name, t.ShortCode, t.TeamType, t.HomeFieldID, t.GamesRequired,
	)
	return scanTeam(row)
}

func (r *TeamRepo) Get(ctx context.Context, id string) (*model.Team, error) {
	row := r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT %s FROM teams WHERE id = $1`, teamColumns),
		id,
	)
	t, err := scanTeam(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *TeamRepo) Update(ctx context.Context, id string, t model.Team) (*model.Team, error) {
	row := r.db.QueryRow(ctx,
		fmt.Sprintf(`UPDATE teams
		 SET division_id = $1, name = $2, short_code = $3, team_type = $4,
		     home_field_id = $5, games_required = $6, updated_at = NOW()
		 WHERE id = $7
		 RETURNING %s`, teamColumns),
		t.DivisionID, t.Name, t.ShortCode, t.TeamType, t.HomeFieldID, t.GamesRequired, id,
	)
	updated, err := scanTeam(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func (r *TeamRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM teams WHERE id = $1`,
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

func (r *TeamRepo) Upsert(ctx context.Context, t model.Team) error {
	_, err := r.db.Exec(ctx,
		fmt.Sprintf(`INSERT INTO teams (%s)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO UPDATE
		   SET division_id    = EXCLUDED.division_id,
		       name           = EXCLUDED.name,
		       short_code     = EXCLUDED.short_code,
		       team_type      = EXCLUDED.team_type,
		       home_field_id  = EXCLUDED.home_field_id,
		       games_required = EXCLUDED.games_required,
		       updated_at     = EXCLUDED.updated_at`, teamColumns),
		t.ID, t.DivisionID, t.Name, t.ShortCode, t.TeamType,
		t.HomeFieldID, t.GamesRequired, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *TeamRepo) GetTeamsWithRules(ctx context.Context, divisionID string) (*model.TeamsWithRules, error) {
	// Fetch all teams for the division.
	teams, err := r.List(ctx, divisionID)
	if err != nil {
		return nil, err
	}

	// Collect team IDs.
	teamIDs := make([]string, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.ID
	}

	// Fetch all matchup rules where either team is in the division.
	var rules []model.MatchupRule
	if len(teamIDs) > 0 {
		rows, err := r.db.Query(ctx,
			`SELECT id, team_a_id, team_b_id, min_games, max_games, created_at
			 FROM matchup_rules
			 WHERE team_a_id = ANY($1) OR team_b_id = ANY($1)
			 ORDER BY created_at`,
			teamIDs,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var mr model.MatchupRule
			if err := rows.Scan(&mr.ID, &mr.TeamAID, &mr.TeamBID, &mr.MinGames, &mr.MaxGames, &mr.CreatedAt); err != nil {
				return nil, err
			}
			rules = append(rules, mr)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if rules == nil {
		rules = []model.MatchupRule{}
	}

	return &model.TeamsWithRules{
		Teams:        teams,
		MatchupRules: rules,
	}, nil
}
