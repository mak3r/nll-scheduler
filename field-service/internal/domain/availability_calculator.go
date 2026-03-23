package domain

import (
	"time"

	"github.com/nll-scheduler/field-service/internal/model"
)

// MaterializeSlots expands recurring and one-off availability windows
// into concrete date/time slots, subtracting blackout dates.
func MaterializeSlots(
	fieldID string,
	windows []model.AvailabilityWindow,
	blackouts []model.BlackoutDate,
	start, end time.Time,
) []model.AvailableSlot {
	blackoutSet := make(map[string]bool)
	for _, b := range blackouts {
		blackoutSet[b.BlackoutDate] = true
	}

	var slots []model.AvailableSlot

	for _, w := range windows {
		wStart, _ := time.Parse("2006-01-02", w.StartDate)
		wEnd, _ := time.Parse("2006-01-02", w.EndDate)

		// Clamp to requested range
		if wStart.Before(start) {
			wStart = start
		}
		if wEnd.After(end) {
			wEnd = end
		}

		if w.WindowType == "recurring" {
			// Expand each day in [wStart, wEnd] that matches days_of_week
			daySet := make(map[int]bool)
			for _, d := range w.DaysOfWeek {
				daySet[d] = true
			}
			for d := wStart; !d.After(wEnd); d = d.AddDate(0, 0, 1) {
				// time.Weekday: Sunday=0, Monday=1, ... Saturday=6
				if daySet[int(d.Weekday())] {
					dateStr := d.Format("2006-01-02")
					if !blackoutSet[dateStr] {
						slots = append(slots, model.AvailableSlot{
							FieldID:   fieldID,
							Date:      dateStr,
							StartTime: w.StartTime,
							EndTime:   w.EndTime,
						})
					}
				}
			}
		} else {
			// oneoff: just the start_date itself
			dateStr := wStart.Format("2006-01-02")
			if !blackoutSet[dateStr] {
				slots = append(slots, model.AvailableSlot{
					FieldID:   fieldID,
					Date:      dateStr,
					StartTime: w.StartTime,
					EndTime:   w.EndTime,
				})
			}
		}
	}

	return slots
}
