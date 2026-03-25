package model

import "time"

type Division struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SeasonYear int       `json:"season_year"`
	SeasonID   string    `json:"season_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Team struct {
	ID          string    `json:"id"`
	DivisionID  string    `json:"division_id"`
	Name        string    `json:"name"`
	ShortCode   string    `json:"short_code"`
	TeamType    string    `json:"team_type"` // "local" | "interleague"
	HomeFieldID *string   `json:"home_field_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MatchupRule struct {
	ID        string    `json:"id"`
	TeamAID   string    `json:"team_a_id"`
	TeamBID   string    `json:"team_b_id"`
	MinGames  int       `json:"min_games"`
	MaxGames  int       `json:"max_games"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamsWithRules struct {
	Teams        []Team        `json:"teams"`
	MatchupRules []MatchupRule `json:"matchup_rules"`
}

type DivisionFieldRule struct {
	ID         string    `json:"id"`
	DivisionID string    `json:"division_id"`
	FieldID    string    `json:"field_id"`
	RuleType   string    `json:"rule_type"` // "allowed" | "preferred"
	CreatedAt  time.Time `json:"created_at"`
}
