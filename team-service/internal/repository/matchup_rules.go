package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/team-service/internal/model"
)

type MatchupRuleRepo struct {
	db *pgxpool.Pool
}

func NewMatchupRuleRepo(db *pgxpool.Pool) *MatchupRuleRepo {
	return &MatchupRuleRepo{db: db}
}

func (r *MatchupRuleRepo) List(ctx context.Context, teamID string) ([]model.MatchupRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, team_a_id, team_b_id, min_games, max_games, created_at
		 FROM matchup_rules
		 WHERE team_a_id = $1 OR team_b_id = $1
		 ORDER BY created_at`,
		teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.MatchupRule
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
	if rules == nil {
		rules = []model.MatchupRule{}
	}
	return rules, nil
}

func (r *MatchupRuleRepo) ListAll(ctx context.Context) ([]model.MatchupRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, team_a_id, team_b_id, min_games, max_games, created_at
		 FROM matchup_rules
		 ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.MatchupRule
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
	if rules == nil {
		rules = []model.MatchupRule{}
	}
	return rules, nil
}

func (r *MatchupRuleRepo) Upsert(ctx context.Context, mr model.MatchupRule) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO matchup_rules (id, team_a_id, team_b_id, min_games, max_games, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE
		   SET team_a_id = EXCLUDED.team_a_id,
		       team_b_id = EXCLUDED.team_b_id,
		       min_games = EXCLUDED.min_games,
		       max_games = EXCLUDED.max_games`,
		mr.ID, mr.TeamAID, mr.TeamBID, mr.MinGames, mr.MaxGames, mr.CreatedAt,
	)
	return err
}

func (r *MatchupRuleRepo) Create(ctx context.Context, teamAID, teamBID string, minGames, maxGames int) (*model.MatchupRule, error) {
	var mr model.MatchupRule
	err := r.db.QueryRow(ctx,
		`INSERT INTO matchup_rules (team_a_id, team_b_id, min_games, max_games)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, team_a_id, team_b_id, min_games, max_games, created_at`,
		teamAID, teamBID, minGames, maxGames,
	).Scan(&mr.ID, &mr.TeamAID, &mr.TeamBID, &mr.MinGames, &mr.MaxGames, &mr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &mr, nil
}

func (r *MatchupRuleRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM matchup_rules WHERE id = $1`,
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

