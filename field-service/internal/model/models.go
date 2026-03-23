package model

import "time"

type Field struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Address        *string   `json:"address,omitempty"`
	MaxGamesPerDay int       `json:"max_games_per_day"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AvailabilityWindow struct {
	ID         string    `json:"id"`
	FieldID    string    `json:"field_id"`
	WindowType string    `json:"window_type"` // "recurring" | "oneoff"
	DaysOfWeek []int     `json:"days_of_week"`
	StartDate  string    `json:"start_date"` // "2025-03-01"
	EndDate    string    `json:"end_date"`
	StartTime  string    `json:"start_time"` // "09:00:00"
	EndTime    string    `json:"end_time"`
	CreatedAt  time.Time `json:"created_at"`
}

type BlackoutDate struct {
	ID           string    `json:"id"`
	FieldID      string    `json:"field_id"`
	BlackoutDate string    `json:"blackout_date"` // "2025-07-04"
	Reason       *string   `json:"reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AvailableSlot struct {
	FieldID   string `json:"field_id"`
	Date      string `json:"date"`       // "2025-06-07"
	StartTime string `json:"start_time"` // "09:00:00"
	EndTime   string `json:"end_time"`
}
