package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type SurplusMonth struct {
	Month         string  `json:"month"`
	Source        string  `json:"source"`
	ExtraChecks   int     `json:"extra_checks"`
	SurplusAmount float64 `json:"surplus_amount"`
}

type SurplusResult struct {
	SurplusMonths []SurplusMonth `json:"surplus_months"`
	AnnualSurplus float64        `json:"annual_surplus"`
}

type SurplusDetector struct {
	generator *PeriodGenerator
}

func NewSurplusDetector() *SurplusDetector {
	return &SurplusDetector{
		generator: NewPeriodGenerator(),
	}
}

func (d *SurplusDetector) Detect(sources []models.IncomeSource, from, to time.Time) (*SurplusResult, error) {
	result := &SurplusResult{
		SurplusMonths: []SurplusMonth{},
	}

	for _, source := range sources {
		dates, err := d.generator.Generate(source, from, to)
		if err != nil {
			continue
		}

		amount := 0.0
		if source.DefaultAmount != nil {
			amount = *source.DefaultAmount
		}

		// Group by month
		monthCounts := make(map[string]int)
		for _, date := range dates {
			key := date.Format("2006-01")
			monthCounts[key]++
		}

		// Determine expected checks per month
		expectedPerMonth := d.expectedPerMonth(source)

		for month, count := range monthCounts {
			if count > expectedPerMonth {
				extra := count - expectedPerMonth
				t, _ := time.Parse("2006-01", month)
				monthLabel := t.Format("January 2006")

				surplus := SurplusMonth{
					Month:         monthLabel,
					Source:        source.Name,
					ExtraChecks:   extra,
					SurplusAmount: float64(extra) * amount,
				}
				result.SurplusMonths = append(result.SurplusMonths, surplus)
				result.AnnualSurplus += surplus.SurplusAmount
			}
		}
	}

	return result, nil
}

func (d *SurplusDetector) expectedPerMonth(source models.IncomeSource) int {
	switch source.PaySchedule {
	case "weekly":
		return 4 // 4 weeks per month normally
	case "biweekly":
		return 2 // 2 checks per month normally
	case "semimonthly":
		return 2 // always exactly 2
	default:
		return 1
	}
}

// GeneratePayDatesForYear is a convenience wrapper.
func (d *SurplusDetector) GeneratePayDatesForYear(source models.IncomeSource, year int) ([]time.Time, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
	return d.generator.Generate(source, from, to)
}

// Helper to extract schedule info for display
func ScheduleDescription(source models.IncomeSource) string {
	switch source.PaySchedule {
	case "weekly":
		var sched models.WeeklySchedule
		json.Unmarshal(source.ScheduleDetail, &sched)
		return fmt.Sprintf("Every %s", time.Weekday(sched.Weekday))
	case "biweekly":
		var sched models.BiweeklySchedule
		json.Unmarshal(source.ScheduleDetail, &sched)
		return fmt.Sprintf("Every other %s", time.Weekday(sched.Weekday))
	case "semimonthly":
		var sched models.SemiMonthlySchedule
		json.Unmarshal(source.ScheduleDetail, &sched)
		if len(sched.Days) == 2 {
			return fmt.Sprintf("%dth and %dth of each month", sched.Days[0], sched.Days[1])
		}
		return "Twice monthly"
	default:
		return source.PaySchedule
	}
}
