package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type PeriodGenerator struct{}

func NewPeriodGenerator() *PeriodGenerator {
	return &PeriodGenerator{}
}

func (g *PeriodGenerator) Generate(source models.IncomeSource, from, to time.Time) ([]time.Time, error) {
	switch source.PaySchedule {
	case "weekly":
		return g.generateWeekly(source.ScheduleDetail, from, to)
	case "biweekly":
		return g.generateBiweekly(source.ScheduleDetail, from, to)
	case "semimonthly":
		return g.generateSemiMonthly(source.ScheduleDetail, from, to)
	case "one_time":
		return g.generateOneTime(source.ScheduleDetail, from, to)
	default:
		return nil, fmt.Errorf("unknown pay schedule: %s", source.PaySchedule)
	}
}

func (g *PeriodGenerator) generateWeekly(detail json.RawMessage, from, to time.Time) ([]time.Time, error) {
	var schedule models.WeeklySchedule
	if err := json.Unmarshal(detail, &schedule); err != nil {
		return nil, fmt.Errorf("parsing weekly schedule: %w", err)
	}

	// time.Weekday() only ever returns 0-6. Without this guard an out-of-range
	// weekday makes the search loop below spin forever, hanging the request.
	if schedule.Weekday < 0 || schedule.Weekday > 6 {
		return nil, fmt.Errorf("weekly schedule weekday must be 0-6, got %d", schedule.Weekday)
	}

	targetWeekday := time.Weekday(schedule.Weekday)
	var dates []time.Time

	// Find first target weekday on or after from
	current := from
	for current.Weekday() != targetWeekday {
		current = current.AddDate(0, 0, 1)
	}

	for !current.After(to) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 7)
	}

	return dates, nil
}

func (g *PeriodGenerator) generateBiweekly(detail json.RawMessage, from, to time.Time) ([]time.Time, error) {
	var schedule models.BiweeklySchedule
	if err := json.Unmarshal(detail, &schedule); err != nil {
		return nil, fmt.Errorf("parsing biweekly schedule: %w", err)
	}

	anchor, err := time.Parse("2006-01-02", schedule.AnchorDate)
	if err != nil {
		return nil, fmt.Errorf("parsing anchor date: %w", err)
	}

	var dates []time.Time

	// Calculate which biweekly cycle we're in relative to anchor
	daysDiff := from.Sub(anchor).Hours() / 24
	cycleOffset := int(daysDiff) / 14
	if daysDiff < 0 {
		cycleOffset--
	}

	current := anchor.AddDate(0, 0, cycleOffset*14)
	// Back up if we went too far
	for current.After(from) {
		current = current.AddDate(0, 0, -14)
	}
	// Advance to first date on or after from
	for current.Before(from) {
		current = current.AddDate(0, 0, 14)
	}

	for !current.After(to) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 14)
	}

	return dates, nil
}

func (g *PeriodGenerator) generateSemiMonthly(detail json.RawMessage, from, to time.Time) ([]time.Time, error) {
	var schedule models.SemiMonthlySchedule
	if err := json.Unmarshal(detail, &schedule); err != nil {
		return nil, fmt.Errorf("parsing semimonthly schedule: %w", err)
	}

	if len(schedule.Days) != 2 {
		return nil, fmt.Errorf("semimonthly schedule must have exactly 2 days, got %d", len(schedule.Days))
	}
	// Day 0 (or negative) rolls back into the previous month via time.Date's
	// normalisation, silently producing pay dates outside the requested month.
	for _, day := range schedule.Days {
		if day < 1 || day > 31 {
			return nil, fmt.Errorf("semimonthly schedule days must be 1-31, got %d", day)
		}
	}

	var dates []time.Time

	current := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, from.Location())

	for !current.After(to) {
		year, month := current.Year(), current.Month()
		for _, day := range schedule.Days {
			// Clamp to last day of month
			lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, current.Location()).Day()
			actualDay := day
			if actualDay > lastDay {
				actualDay = lastDay
			}
			d := time.Date(year, month, actualDay, 0, 0, 0, 0, current.Location())

			// Check if the original scheduled date is in range BEFORE adjusting.
			// This ensures that if a weekend pay date adjusts to a Friday that falls
			// before 'from', the period is still included (with the adjusted date).
			originalInRange := !d.Before(from) && !d.After(to)

			// Adjust for weekends: move to preceding Friday
			if schedule.AdjustForWeekends {
				d = adjustToWeekday(d)
			}

			// Include if either:
			// 1. Original date was in range (even if adjusted date falls before 'from'), or
			// 2. Adjusted date is in range (to catch dates that adjust INTO the range)
			if originalInRange || (!d.Before(from) && !d.After(to)) {
				dates = append(dates, d)
			}
		}
		current = current.AddDate(0, 1, 0)
	}

	return dates, nil
}

// adjustToWeekday moves weekend dates to the preceding Friday
func adjustToWeekday(d time.Time) time.Time {
	switch d.Weekday() {
	case time.Saturday:
		return d.AddDate(0, 0, -1) // Friday
	case time.Sunday:
		return d.AddDate(0, 0, -2) // Friday
	default:
		return d
	}
}

func (g *PeriodGenerator) generateOneTime(detail json.RawMessage, from, to time.Time) ([]time.Time, error) {
	var schedule models.OneTimeSchedule
	if err := json.Unmarshal(detail, &schedule); err != nil {
		return nil, fmt.Errorf("parsing one-time schedule: %w", err)
	}

	date, err := time.Parse("2006-01-02", schedule.Date)
	if err != nil {
		return nil, fmt.Errorf("parsing one-time date: %w", err)
	}

	if !date.Before(from) && !date.After(to) {
		return []time.Time{date}, nil
	}

	return nil, nil
}
