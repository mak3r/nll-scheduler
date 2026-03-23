package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TeamClient fetches team data from team-service.
type TeamClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTeamClient(baseURL string) *TeamClient {
	return &TeamClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type Team struct {
	ID            string  `json:"id"`
	DivisionID    string  `json:"division_id"`
	Name          string  `json:"name"`
	ShortCode     string  `json:"short_code"`
	TeamType      string  `json:"team_type"`
	HomeFieldID   *string `json:"home_field_id"`
	GamesRequired int     `json:"games_required"`
}

type MatchupRule struct {
	ID       string `json:"id"`
	TeamAID  string `json:"team_a_id"`
	TeamBID  string `json:"team_b_id"`
	MinGames int    `json:"min_games"`
	MaxGames int    `json:"max_games"`
}

type TeamsWithRules struct {
	Teams        []Team        `json:"teams"`
	MatchupRules []MatchupRule `json:"matchup_rules"`
}

func (c *TeamClient) GetTeamsWithRules(ctx context.Context, divisionID string) (*TeamsWithRules, error) {
	url := fmt.Sprintf("%s/divisions/%s/teams-with-rules", c.baseURL, divisionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("team-service returned %d: %s", resp.StatusCode, string(body))
	}

	var result TeamsWithRules
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
