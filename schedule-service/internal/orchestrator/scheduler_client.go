// Package orchestrator handles the scheduling generation workflow:
// 1. Fetch teams + matchup rules from team-service
// 2. Fetch field availability from field-service
// 3. Build solver request and call scheduler-engine
// 4. Persist resulting games to DB
package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SchedulerClient calls the scheduler-engine /solve endpoint.
type SchedulerClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewSchedulerClient(baseURL string) *SchedulerClient {
	return &SchedulerClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // solver can take a while
		},
	}
}

// SolveRequest mirrors app/schemas/solve.py SolveRequest.
type SolveRequest struct {
	SeasonID                  string             `json:"season_id"`
	StartDate                 string             `json:"start_date"`
	EndDate                   string             `json:"end_date"`
	Teams                     []SolverTeam       `json:"teams"`
	MatchupRules              []SolverMatchupRule `json:"matchup_rules"`
	Fields                    []SolverField      `json:"fields"`
	BlackoutDates             []string           `json:"blackout_dates"`
	PreferredInterleagueDates []string           `json:"preferred_interleague_dates"`
	Constraints               []SolverConstraint `json:"constraints"`
	TimeLimitSeconds            int                 `json:"time_limit_seconds"`
	DivisionFieldRestrictions   map[string][]string `json:"division_field_restrictions"`
	DivisionPreferredFields     map[string][]string `json:"division_preferred_fields"`
}

type SolverTeam struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DivisionID    string `json:"division_id"`
	TeamType      string `json:"team_type"`
	GamesRequired int    `json:"games_required"`
}

type SolverMatchupRule struct {
	TeamAID  string `json:"team_a_id"`
	TeamBID  string `json:"team_b_id"`
	MinGames int    `json:"min_games"`
	MaxGames int    `json:"max_games"`
}

type SolverField struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	MaxGamesPerDay int               `json:"max_games_per_day"`
	AvailableSlots []SolverFieldSlot `json:"available_slots"`
}

type SolverFieldSlot struct {
	FieldID   string `json:"field_id"`
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type SolverConstraint struct {
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params"`
	IsHard bool            `json:"is_hard"`
	Weight float64         `json:"weight"`
}

// SolveResponse mirrors app/schemas/solve.py SolveResponse.
type SolveResponse struct {
	Status           string          `json:"status"`
	Games            []SolverGame    `json:"games"`
	SolverStats      json.RawMessage `json:"solver_stats"`
	UnmetConstraints []string        `json:"unmet_constraints"`
}

type SolverGame struct {
	HomeTeamID    string `json:"home_team_id"`
	AwayTeamID    string `json:"away_team_id"`
	FieldID       string `json:"field_id"`
	GameDate      string `json:"game_date"`
	StartTime     string `json:"start_time"`
	IsInterleague bool   `json:"is_interleague"`
}

func (c *SchedulerClient) Solve(ctx context.Context, req SolveRequest) (*SolveResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/solve", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scheduler-engine returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result SolveResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
