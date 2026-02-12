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
	default:
		return nil, fmt.Errorf("unknown pay schedule: %s", source.PaySchedule)
	}
}

func (g *PeriodGenerator) generateWeekly(detail json.RawMessage, from, to time.Time) ([]time.Time, error) {
	var schedule models.WeeklySchedule
	if err := json.Unmarshal(detail, &schedule); err != nil {
		return nil, fmt.Errorf("parsing weekly schedule: %w", err)
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
			if !d.Before(from) && !d.After(to) {
				dates = append(dates, d)
			}
		}
		current = current.AddDate(0, 1, 0)
	}

	return dates, nil
}
