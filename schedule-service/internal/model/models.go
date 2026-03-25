package model

import (
	"encoding/json"
	"time"
)

type Season struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DivisionIDs []string  `json:"division_ids"`
	StartDate   string    `json:"start_date"` // "2025-04-01"
	EndDate     string    `json:"end_date"`
	Status      string    `json:"status"` // draft|generating|review|published
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SeasonBlackout struct {
	ID           string    `json:"id"`
	SeasonID     string    `json:"season_id"`
	BlackoutDate string    `json:"blackout_date"`
	CreatedAt    time.Time `json:"created_at"`
}

type PreferredDate struct {
	ID            string    `json:"id"`
	SeasonID      string    `json:"season_id"`
	PreferredDate string    `json:"preferred_date"`
	Weight        float64   `json:"weight"`
	CreatedAt     time.Time `json:"created_at"`
}

type SeasonConstraint struct {
	ID        string          `json:"id"`
	SeasonID  string          `json:"season_id"`
	Type      string          `json:"type"`
	Params    json.RawMessage `json:"params"`
	IsHard    bool            `json:"is_hard"`
	Weight    float64         `json:"weight"`
	CreatedAt time.Time       `json:"created_at"`
}

type Game struct {
	ID             string    `json:"id"`
	SeasonID       string    `json:"season_id"`
	HomeTeamID     string    `json:"home_team_id"`
	AwayTeamID     string    `json:"away_team_id"`
	FieldID        string    `json:"field_id"`
	GameDate       string    `json:"game_date"`
	StartTime      string    `json:"start_time"`
	Status         string    `json:"status"` // scheduled|cancelled|completed
	DivisionID     string    `json:"division_id"`
	IsInterleague  bool      `json:"is_interleague"`
	ManuallyEdited bool      `json:"manually_edited"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DivisionGamesRequired struct {
	ID           string    `json:"id"`
	SeasonID     string    `json:"season_id"`
	DivisionID   string    `json:"division_id"`
	GamesRequired int      `json:"games_required"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type GenerationRun struct {
	ID           string          `json:"id"`
	SeasonID     string          `json:"season_id"`
	Status       string          `json:"status"` // pending|running|success|failed
	SolverStats  json.RawMessage `json:"solver_stats,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
