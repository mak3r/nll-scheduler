package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nll-scheduler/team-service/internal/model"
)

type DivisionFieldRuleRepo struct {
	db *pgxpool.Pool
}

func NewDivisionFieldRuleRepo(db *pgxpool.Pool) *DivisionFieldRuleRepo {
	return &DivisionFieldRuleRepo{db: db}
}

func (r *DivisionFieldRuleRepo) List(ctx context.Context, divisionID string) ([]model.DivisionFieldRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, division_id, field_id, rule_type, created_at
		 FROM division_field_rules
		 WHERE division_id = $1
		 ORDER BY rule_type, field_id`,
		divisionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.DivisionFieldRule
	for rows.Next() {
		var fr model.DivisionFieldRule
		if err := rows.Scan(&fr.ID, &fr.DivisionID, &fr.FieldID, &fr.RuleType, &fr.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []model.DivisionFieldRule{}
	}
	return rules, nil
}

func (r *DivisionFieldRuleRepo) ListAll(ctx context.Context) ([]model.DivisionFieldRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, division_id, field_id, rule_type, created_at
		 FROM division_field_rules
		 ORDER BY division_id, rule_type, field_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.DivisionFieldRule
	for rows.Next() {
		var fr model.DivisionFieldRule
		if err := rows.Scan(&fr.ID, &fr.DivisionID, &fr.FieldID, &fr.RuleType, &fr.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []model.DivisionFieldRule{}
	}
	return rules, nil
}

func (r *DivisionFieldRuleRepo) Create(ctx context.Context, divisionID, fieldID, ruleType string) (*model.DivisionFieldRule, error) {
	var fr model.DivisionFieldRule
	err := r.db.QueryRow(ctx,
		`INSERT INTO division_field_rules (division_id, field_id, rule_type)
		 VALUES ($1, $2, $3)
		 RETURNING id, division_id, field_id, rule_type, created_at`,
		divisionID, fieldID, ruleType,
	).Scan(&fr.ID, &fr.DivisionID, &fr.FieldID, &fr.RuleType, &fr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &fr, nil
}

func (r *DivisionFieldRuleRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx,
		`DELETE FROM division_field_rules WHERE id = $1`,
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

func (r *DivisionFieldRuleRepo) Upsert(ctx context.Context, fr model.DivisionFieldRule) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO division_field_rules (id, division_id, field_id, rule_type, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE
		   SET division_id = EXCLUDED.division_id,
		       field_id    = EXCLUDED.field_id,
		       rule_type   = EXCLUDED.rule_type`,
		fr.ID, fr.DivisionID, fr.FieldID, fr.RuleType, fr.CreatedAt,
	)
	return err
}
