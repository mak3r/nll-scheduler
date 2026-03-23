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

// FieldClient fetches field availability from field-service.
type FieldClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewFieldClient(baseURL string) *FieldClient {
	return &FieldClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type Field struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	MaxGamesPerDay int    `json:"max_games_per_day"`
	IsActive       bool   `json:"is_active"`
}

type AvailableSlot struct {
	FieldID   string `json:"field_id"`
	Date      string `json:"date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func (c *FieldClient) ListFields(ctx context.Context) ([]Field, error) {
	url := c.baseURL + "/fields"
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
		return nil, fmt.Errorf("field-service returned %d: %s", resp.StatusCode, string(body))
	}

	var fields []Field
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return fields, nil
}

// GetAvailableDatesBulk fetches materialized availability slots for the given fields and date range.
// Returns map[fieldID][]AvailableSlot.
func (c *FieldClient) GetAvailableDatesBulk(
	ctx context.Context,
	fieldIDs []string,
	start, end string,
) (map[string][]AvailableSlot, error) {
	ids := strings.Join(fieldIDs, ",")
	url := fmt.Sprintf("%s/fields/available-dates-bulk?start=%s&end=%s&field_ids=%s",
		c.baseURL, start, end, ids)

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
		return nil, fmt.Errorf("field-service returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string][]AvailableSlot
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return result, nil
}
