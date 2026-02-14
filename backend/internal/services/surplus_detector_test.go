package services

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ptrFloat64(v float64) *float64 { return &v }

func mustJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return b
}

func weeklySource(t *testing.T, name string, weekday int, amount *float64) models.IncomeSource {
	t.Helper()
	return models.IncomeSource{
		Name:        name,
		PaySchedule: "weekly",
		ScheduleDetail: mustJSON(t, models.WeeklySchedule{
			Weekday: weekday,
		}),
		DefaultAmount: amount,
		IsActive:      true,
	}
}

func biweeklySource(t *testing.T, name string, weekday int, anchor string, amount *float64) models.IncomeSource {
	t.Helper()
	return models.IncomeSource{
		Name:        name,
		PaySchedule: "biweekly",
		ScheduleDetail: mustJSON(t, models.BiweeklySchedule{
			Weekday:    weekday,
			AnchorDate: anchor,
		}),
		DefaultAmount: amount,
		IsActive:      true,
	}
}

func semimonthlySource(t *testing.T, name string, days []int, amount *float64) models.IncomeSource {
	t.Helper()
	return models.IncomeSource{
		Name:        name,
		PaySchedule: "semimonthly",
		ScheduleDetail: mustJSON(t, models.SemiMonthlySchedule{
			Days: days,
		}),
		DefaultAmount: amount,
		IsActive:      true,
	}
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

// findSurplusForMonth returns the SurplusMonth entry matching the given month
// label and source name, or nil if none was found.
func findSurplusForMonth(result *SurplusResult, monthLabel, sourceName string) *SurplusMonth {
	for i := range result.SurplusMonths {
		if result.SurplusMonths[i].Month == monthLabel && result.SurplusMonths[i].Source == sourceName {
			return &result.SurplusMonths[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// TestNewSurplusDetector
// ---------------------------------------------------------------------------

func TestNewSurplusDetector(t *testing.T) {
	d := NewSurplusDetector()
	if d == nil {
		t.Fatal("NewSurplusDetector returned nil")
	}
	if d.generator == nil {
		t.Fatal("SurplusDetector.generator is nil")
	}
}

// ---------------------------------------------------------------------------
// TestDetect_EmptySources
// ---------------------------------------------------------------------------

func TestDetect_EmptySources(t *testing.T) {
	d := NewSurplusDetector()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected 0 surplus months, got %d", len(result.SurplusMonths))
	}
	if result.AnnualSurplus != 0 {
		t.Errorf("expected annual surplus 0, got %f", result.AnnualSurplus)
	}
}

func TestDetect_NilSources(t *testing.T) {
	d := NewSurplusDetector()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect(nil, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected 0 surplus months, got %d", len(result.SurplusMonths))
	}
}

// ---------------------------------------------------------------------------
// TestDetect_WeeklySurplus — weekly pays 52 times/year; some months get 5
// ---------------------------------------------------------------------------

func TestDetect_WeeklySurplus(t *testing.T) {
	d := NewSurplusDetector()

	// Friday = weekday 5. In 2025 there are 52 Fridays.
	// Expected per month is 4 for weekly, so any month with 5 Fridays is surplus.
	source := weeklySource(t, "Weekly Job", 5, ptrFloat64(1000.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 52 Fridays distributed across 12 months at an expected 4 per month
	// (48 expected), there must be 4 extra pays -> 4 surplus months with 1 extra each.
	totalExtra := 0
	for _, sm := range result.SurplusMonths {
		if sm.Source != "Weekly Job" {
			t.Errorf("unexpected source: %s", sm.Source)
		}
		if sm.ExtraChecks < 1 {
			t.Errorf("extra checks should be >= 1, got %d", sm.ExtraChecks)
		}
		totalExtra += sm.ExtraChecks
	}

	// 52 pays - 12*4 expected = 4 extra pays total
	if totalExtra != 4 {
		t.Errorf("expected 4 total extra checks across all surplus months, got %d", totalExtra)
	}

	expectedSurplus := 4 * 1000.0
	if !floatEqual(result.AnnualSurplus, expectedSurplus) {
		t.Errorf("expected annual surplus %.2f, got %.2f", expectedSurplus, result.AnnualSurplus)
	}

	// Each surplus month should have ExtraChecks=1 and SurplusAmount=1000
	for _, sm := range result.SurplusMonths {
		if sm.ExtraChecks != 1 {
			t.Errorf("month %s: expected 1 extra check, got %d", sm.Month, sm.ExtraChecks)
		}
		if !floatEqual(sm.SurplusAmount, 1000.0) {
			t.Errorf("month %s: expected surplus amount 1000, got %.2f", sm.Month, sm.SurplusAmount)
		}
	}
}

// ---------------------------------------------------------------------------
// TestDetect_BiweeklySurplus — biweekly pays 26 times/year; some months get 3
// ---------------------------------------------------------------------------

func TestDetect_BiweeklySurplus(t *testing.T) {
	d := NewSurplusDetector()

	// Anchor on a known Friday: 2025-01-03 (Friday).
	// Biweekly expected per month = 2, so months with 3 are surplus.
	source := biweeklySource(t, "Biweekly Job", 5, "2025-01-03", ptrFloat64(2000.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 26 biweekly pays - 12*2 expected = 2 extra pays across the year
	totalExtra := 0
	for _, sm := range result.SurplusMonths {
		if sm.Source != "Biweekly Job" {
			t.Errorf("unexpected source: %s", sm.Source)
		}
		totalExtra += sm.ExtraChecks
	}
	if totalExtra != 2 {
		t.Errorf("expected 2 total extra biweekly checks, got %d", totalExtra)
	}

	expectedSurplus := 2 * 2000.0
	if !floatEqual(result.AnnualSurplus, expectedSurplus) {
		t.Errorf("expected annual surplus %.2f, got %.2f", expectedSurplus, result.AnnualSurplus)
	}

	// Each surplus month should have exactly 1 extra check
	for _, sm := range result.SurplusMonths {
		if sm.ExtraChecks != 1 {
			t.Errorf("month %s: expected 1 extra check, got %d", sm.Month, sm.ExtraChecks)
		}
		if !floatEqual(sm.SurplusAmount, 2000.0) {
			t.Errorf("month %s: expected surplus amount 2000, got %.2f", sm.Month, sm.SurplusAmount)
		}
	}
}

// ---------------------------------------------------------------------------
// TestDetect_SemiMonthlyNeverSurplus — always exactly 2 per month
// ---------------------------------------------------------------------------

func TestDetect_SemiMonthlyNeverSurplus(t *testing.T) {
	d := NewSurplusDetector()

	source := semimonthlySource(t, "Semi Job", []int{1, 15}, ptrFloat64(3000.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 0 {
		t.Errorf("semimonthly should never have surplus months, got %d", len(result.SurplusMonths))
		for _, sm := range result.SurplusMonths {
			t.Logf("  surplus: %s source=%s extra=%d amount=%.2f",
				sm.Month, sm.Source, sm.ExtraChecks, sm.SurplusAmount)
		}
	}

	if result.AnnualSurplus != 0 {
		t.Errorf("semimonthly annual surplus should be 0, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_NoSurplusInPartialRange — short range avoids surplus
// ---------------------------------------------------------------------------

func TestDetect_NoSurplusInPartialRange(t *testing.T) {
	d := NewSurplusDetector()

	// Pick a month where Friday only appears 4 times.
	// February 2025: Fridays are Feb 7, 14, 21, 28 = exactly 4 (no surplus).
	source := weeklySource(t, "Job", 5, ptrFloat64(500.00))

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected no surplus months for Feb 2025 weekly Friday, got %d", len(result.SurplusMonths))
	}
	if result.AnnualSurplus != 0 {
		t.Errorf("expected annual surplus 0, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_FiveWeekMonth — target a specific month with 5 occurrences
// ---------------------------------------------------------------------------

func TestDetect_FiveWeekMonth(t *testing.T) {
	d := NewSurplusDetector()

	// January 2025: Fridays are Jan 3, 10, 17, 24, 31 = 5 Fridays
	source := weeklySource(t, "Job", 5, ptrFloat64(750.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 1 {
		t.Fatalf("expected 1 surplus month, got %d", len(result.SurplusMonths))
	}

	sm := result.SurplusMonths[0]
	if sm.Month != "January 2025" {
		t.Errorf("expected month 'January 2025', got '%s'", sm.Month)
	}
	if sm.Source != "Job" {
		t.Errorf("expected source 'Job', got '%s'", sm.Source)
	}
	if sm.ExtraChecks != 1 {
		t.Errorf("expected 1 extra check, got %d", sm.ExtraChecks)
	}
	if !floatEqual(sm.SurplusAmount, 750.0) {
		t.Errorf("expected surplus amount 750, got %.2f", sm.SurplusAmount)
	}
	if !floatEqual(result.AnnualSurplus, 750.0) {
		t.Errorf("expected annual surplus 750, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_NilDefaultAmount — surplus detected but amount is 0
// ---------------------------------------------------------------------------

func TestDetect_NilDefaultAmount(t *testing.T) {
	d := NewSurplusDetector()

	// Weekly Friday, nil amount. January 2025 has 5 Fridays -> surplus detected.
	source := weeklySource(t, "Volunteer", 5, nil)

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 1 {
		t.Fatalf("expected 1 surplus month, got %d", len(result.SurplusMonths))
	}

	sm := result.SurplusMonths[0]
	if sm.ExtraChecks != 1 {
		t.Errorf("expected 1 extra check, got %d", sm.ExtraChecks)
	}
	if sm.SurplusAmount != 0 {
		t.Errorf("expected surplus amount 0 for nil default amount, got %.2f", sm.SurplusAmount)
	}
	if result.AnnualSurplus != 0 {
		t.Errorf("expected annual surplus 0 for nil default amount, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_AnnualSurplusAcrossMultipleSources
// ---------------------------------------------------------------------------

func TestDetect_AnnualSurplusAcrossMultipleSources(t *testing.T) {
	d := NewSurplusDetector()

	weekly := weeklySource(t, "Main Job", 5, ptrFloat64(1000.00))
	biweekly := biweeklySource(t, "Side Gig", 5, "2025-01-03", ptrFloat64(500.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{weekly, biweekly}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Weekly: 52 - 48 = 4 extra at $1000 = $4000
	// Biweekly: 26 - 24 = 2 extra at $500 = $1000
	// Total: $5000
	weeklyExtra := 0
	biweeklyExtra := 0
	for _, sm := range result.SurplusMonths {
		switch sm.Source {
		case "Main Job":
			weeklyExtra += sm.ExtraChecks
		case "Side Gig":
			biweeklyExtra += sm.ExtraChecks
		default:
			t.Errorf("unexpected source: %s", sm.Source)
		}
	}

	if weeklyExtra != 4 {
		t.Errorf("expected 4 weekly extras, got %d", weeklyExtra)
	}
	if biweeklyExtra != 2 {
		t.Errorf("expected 2 biweekly extras, got %d", biweeklyExtra)
	}

	expectedTotal := 4*1000.0 + 2*500.0
	if !floatEqual(result.AnnualSurplus, expectedTotal) {
		t.Errorf("expected annual surplus %.2f, got %.2f", expectedTotal, result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_InvalidScheduleDetailSkipped — source with bad JSON is skipped
// ---------------------------------------------------------------------------

func TestDetect_InvalidScheduleDetailSkipped(t *testing.T) {
	d := NewSurplusDetector()

	badSource := models.IncomeSource{
		Name:           "Bad Source",
		PaySchedule:    "weekly",
		ScheduleDetail: json.RawMessage(`{invalid json`),
		DefaultAmount:  ptrFloat64(100.0),
		IsActive:       true,
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{badSource}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected 0 surplus months for invalid source, got %d", len(result.SurplusMonths))
	}
	if result.AnnualSurplus != 0 {
		t.Errorf("expected annual surplus 0, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_UnknownScheduleSkipped — unknown pay schedule is skipped
// ---------------------------------------------------------------------------

func TestDetect_UnknownScheduleSkipped(t *testing.T) {
	d := NewSurplusDetector()

	unknownSource := models.IncomeSource{
		Name:           "Unknown",
		PaySchedule:    "quarterly",
		ScheduleDetail: json.RawMessage(`{}`),
		DefaultAmount:  ptrFloat64(100.0),
		IsActive:       true,
	}

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{unknownSource}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// PeriodGenerator.Generate returns an error for unknown schedules,
	// and Detect skips sources that error.
	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected 0 surplus months for unknown schedule, got %d", len(result.SurplusMonths))
	}
}

// ---------------------------------------------------------------------------
// TestExpectedPerMonth
// ---------------------------------------------------------------------------

func TestExpectedPerMonth(t *testing.T) {
	d := NewSurplusDetector()

	tests := []struct {
		schedule string
		expected int
	}{
		{"weekly", 4},
		{"biweekly", 2},
		{"semimonthly", 2},
		{"monthly", 1},
		{"", 1},
		{"unknown", 1},
	}

	for _, tt := range tests {
		t.Run(tt.schedule, func(t *testing.T) {
			source := models.IncomeSource{PaySchedule: tt.schedule}
			got := d.expectedPerMonth(source)
			if got != tt.expected {
				t.Errorf("expectedPerMonth(%q) = %d, want %d", tt.schedule, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGeneratePayDatesForYear
// ---------------------------------------------------------------------------

func TestGeneratePayDatesForYear_Weekly(t *testing.T) {
	d := NewSurplusDetector()

	// Friday weekly for 2025
	source := weeklySource(t, "Job", 5, ptrFloat64(1000.0))

	dates, err := d.GeneratePayDatesForYear(source, 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2025 has 52 Fridays (Jan 3 is first, Dec 26 is last)
	if len(dates) != 52 {
		t.Errorf("expected 52 weekly pay dates in 2025, got %d", len(dates))
	}

	// All dates should be Fridays
	for _, d := range dates {
		if d.Weekday() != time.Friday {
			t.Errorf("expected Friday, got %s on %s", d.Weekday(), d.Format("2006-01-02"))
		}
	}

	// All dates should be in 2025
	for _, d := range dates {
		if d.Year() != 2025 {
			t.Errorf("expected year 2025, got %d for date %s", d.Year(), d.Format("2006-01-02"))
		}
	}
}

func TestGeneratePayDatesForYear_Biweekly(t *testing.T) {
	d := NewSurplusDetector()

	// Biweekly starting from Jan 3, 2025 (a Friday)
	source := biweeklySource(t, "Job", 5, "2025-01-03", ptrFloat64(2000.0))

	dates, err := d.GeneratePayDatesForYear(source, 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 26 biweekly pay dates in a year
	if len(dates) != 26 {
		t.Errorf("expected 26 biweekly pay dates in 2025, got %d", len(dates))
	}

	// All dates should be Fridays
	for _, d := range dates {
		if d.Weekday() != time.Friday {
			t.Errorf("expected Friday, got %s on %s", d.Weekday(), d.Format("2006-01-02"))
		}
	}

	// Dates should be 14 days apart
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1]).Hours() / 24
		if diff != 14 {
			t.Errorf("expected 14 days between pay dates, got %.0f between %s and %s",
				diff, dates[i-1].Format("2006-01-02"), dates[i].Format("2006-01-02"))
		}
	}
}

func TestGeneratePayDatesForYear_SemiMonthly(t *testing.T) {
	d := NewSurplusDetector()

	source := semimonthlySource(t, "Job", []int{1, 15}, ptrFloat64(3000.0))

	dates, err := d.GeneratePayDatesForYear(source, 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 24 semimonthly pay dates in a year (2 per month * 12)
	if len(dates) != 24 {
		t.Errorf("expected 24 semimonthly pay dates in 2025, got %d", len(dates))
	}

	// All dates should be either the 1st or 15th
	for _, d := range dates {
		if d.Day() != 1 && d.Day() != 15 {
			t.Errorf("expected day 1 or 15, got %d on %s", d.Day(), d.Format("2006-01-02"))
		}
	}
}

func TestGeneratePayDatesForYear_DateBoundaries(t *testing.T) {
	d := NewSurplusDetector()

	// Use weekly Friday to verify boundaries
	source := weeklySource(t, "Job", 5, ptrFloat64(100.0))
	dates, err := d.GeneratePayDatesForYear(source, 2025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	firstDate := dates[0]
	lastDate := dates[len(dates)-1]

	// First date should be on or after Jan 1
	jan1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if firstDate.Before(jan1) {
		t.Errorf("first pay date %s is before Jan 1", firstDate.Format("2006-01-02"))
	}

	// Last date should be on or before Dec 31
	dec31 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	if lastDate.After(dec31) {
		t.Errorf("last pay date %s is after Dec 31", lastDate.Format("2006-01-02"))
	}
}

// ---------------------------------------------------------------------------
// TestScheduleDescription
// ---------------------------------------------------------------------------

func TestScheduleDescription_Weekly(t *testing.T) {
	tests := []struct {
		weekday  int
		expected string
	}{
		{0, "Every Sunday"},
		{1, "Every Monday"},
		{5, "Every Friday"},
		{6, "Every Saturday"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			source := models.IncomeSource{
				PaySchedule: "weekly",
				ScheduleDetail: mustJSON(t, models.WeeklySchedule{
					Weekday: tt.weekday,
				}),
			}
			got := ScheduleDescription(source)
			if got != tt.expected {
				t.Errorf("ScheduleDescription() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestScheduleDescription_Biweekly(t *testing.T) {
	tests := []struct {
		weekday  int
		expected string
	}{
		{5, "Every other Friday"},
		{1, "Every other Monday"},
		{3, "Every other Wednesday"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			source := models.IncomeSource{
				PaySchedule: "biweekly",
				ScheduleDetail: mustJSON(t, models.BiweeklySchedule{
					Weekday:    tt.weekday,
					AnchorDate: "2025-01-03",
				}),
			}
			got := ScheduleDescription(source)
			if got != tt.expected {
				t.Errorf("ScheduleDescription() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestScheduleDescription_SemiMonthly(t *testing.T) {
	t.Run("two days", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule: "semimonthly",
			ScheduleDetail: mustJSON(t, models.SemiMonthlySchedule{
				Days: []int{1, 15},
			}),
		}
		got := ScheduleDescription(source)
		expected := "1th and 15th of each month"
		if got != expected {
			t.Errorf("ScheduleDescription() = %q, want %q", got, expected)
		}
	})

	t.Run("different days", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule: "semimonthly",
			ScheduleDetail: mustJSON(t, models.SemiMonthlySchedule{
				Days: []int{5, 20},
			}),
		}
		got := ScheduleDescription(source)
		expected := "5th and 20th of each month"
		if got != expected {
			t.Errorf("ScheduleDescription() = %q, want %q", got, expected)
		}
	})

	t.Run("not two days falls back", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule: "semimonthly",
			ScheduleDetail: mustJSON(t, models.SemiMonthlySchedule{
				Days: []int{1},
			}),
		}
		got := ScheduleDescription(source)
		if got != "Twice monthly" {
			t.Errorf("ScheduleDescription() = %q, want %q", got, "Twice monthly")
		}
	})

	t.Run("empty days falls back", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule: "semimonthly",
			ScheduleDetail: mustJSON(t, models.SemiMonthlySchedule{
				Days: []int{},
			}),
		}
		got := ScheduleDescription(source)
		if got != "Twice monthly" {
			t.Errorf("ScheduleDescription() = %q, want %q", got, "Twice monthly")
		}
	})
}

func TestScheduleDescription_Default(t *testing.T) {
	tests := []struct {
		schedule string
	}{
		{"monthly"},
		{"quarterly"},
		{"annual"},
	}

	for _, tt := range tests {
		t.Run(tt.schedule, func(t *testing.T) {
			source := models.IncomeSource{
				PaySchedule:    tt.schedule,
				ScheduleDetail: json.RawMessage(`{}`),
			}
			got := ScheduleDescription(source)
			if got != tt.schedule {
				t.Errorf("ScheduleDescription() = %q, want %q", got, tt.schedule)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDetect_SurplusMonthFormatting — verify month label formatting
// ---------------------------------------------------------------------------

func TestDetect_SurplusMonthFormatting(t *testing.T) {
	d := NewSurplusDetector()

	// March 2025 has 5 Mondays: Mar 3, 10, 17, 24, 31
	source := weeklySource(t, "Job", 1, ptrFloat64(100.0))

	from := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 1 {
		t.Fatalf("expected 1 surplus month, got %d", len(result.SurplusMonths))
	}

	// Verify the month label uses "January 2006" format
	if result.SurplusMonths[0].Month != "March 2025" {
		t.Errorf("expected month label 'March 2025', got '%s'", result.SurplusMonths[0].Month)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_BiweeklyThreePaycheckMonth — verify specific 3-paycheck month
// ---------------------------------------------------------------------------

func TestDetect_BiweeklyThreePaycheckMonth(t *testing.T) {
	d := NewSurplusDetector()

	// Anchor on Jan 3, 2025 (Friday). Biweekly pays: Jan 3, 17, 31 => 3 in Jan.
	source := biweeklySource(t, "Corp", 5, "2025-01-03", ptrFloat64(1500.00))

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 1 {
		t.Fatalf("expected 1 surplus month for Jan 2025 biweekly, got %d", len(result.SurplusMonths))
	}

	sm := result.SurplusMonths[0]
	if sm.Month != "January 2025" {
		t.Errorf("expected month 'January 2025', got '%s'", sm.Month)
	}
	if sm.ExtraChecks != 1 {
		t.Errorf("expected 1 extra check, got %d", sm.ExtraChecks)
	}
	if !floatEqual(sm.SurplusAmount, 1500.0) {
		t.Errorf("expected surplus amount 1500, got %.2f", sm.SurplusAmount)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_MixedSourcesWithNilAmount — one source nil, one with amount
// ---------------------------------------------------------------------------

func TestDetect_MixedSourcesWithNilAmount(t *testing.T) {
	d := NewSurplusDetector()

	nilAmountSource := weeklySource(t, "Volunteer", 5, nil)
	paidSource := weeklySource(t, "Main Job", 5, ptrFloat64(1000.0))

	// January 2025 has 5 Fridays, so both sources produce surplus.
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{nilAmountSource, paidSource}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.SurplusMonths) != 2 {
		t.Fatalf("expected 2 surplus months, got %d", len(result.SurplusMonths))
	}

	volunteerSurplus := findSurplusForMonth(result, "January 2025", "Volunteer")
	mainJobSurplus := findSurplusForMonth(result, "January 2025", "Main Job")

	if volunteerSurplus == nil {
		t.Fatal("expected surplus entry for Volunteer")
	}
	if mainJobSurplus == nil {
		t.Fatal("expected surplus entry for Main Job")
	}

	if volunteerSurplus.SurplusAmount != 0 {
		t.Errorf("volunteer surplus should be 0, got %.2f", volunteerSurplus.SurplusAmount)
	}
	if !floatEqual(mainJobSurplus.SurplusAmount, 1000.0) {
		t.Errorf("main job surplus should be 1000, got %.2f", mainJobSurplus.SurplusAmount)
	}

	// Annual surplus should only include the paid source
	if !floatEqual(result.AnnualSurplus, 1000.0) {
		t.Errorf("annual surplus should be 1000, got %.2f", result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_SurplusMonthsSliceInitialized — result always has non-nil slice
// ---------------------------------------------------------------------------

func TestDetect_SurplusMonthsSliceInitialized(t *testing.T) {
	d := NewSurplusDetector()
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SurplusMonths == nil {
		t.Error("SurplusMonths should be initialized to empty slice, not nil")
	}
}

// ---------------------------------------------------------------------------
// TestDetect_LeapYear — semimonthly with day 29 in Feb of leap year
// ---------------------------------------------------------------------------

func TestDetect_LeapYear(t *testing.T) {
	d := NewSurplusDetector()

	// 2024 is a leap year. Semi-monthly on 15th and 29th.
	source := semimonthlySource(t, "Leap Job", []int{15, 29}, ptrFloat64(1000.0))

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Semimonthly expected is 2, should always be exactly 2 per month.
	if len(result.SurplusMonths) != 0 {
		t.Errorf("semimonthly should have no surplus even in leap year, got %d surplus months", len(result.SurplusMonths))
		for _, sm := range result.SurplusMonths {
			t.Logf("  surplus: %s extra=%d", sm.Month, sm.ExtraChecks)
		}
	}
}

// ---------------------------------------------------------------------------
// TestDetect_NonLeapYear — semimonthly with day 29 in Feb clamped to 28
// ---------------------------------------------------------------------------

func TestDetect_NonLeapYear(t *testing.T) {
	d := NewSurplusDetector()

	// 2025 is not a leap year. Semi-monthly on 15th and 29th.
	// In Feb, day 29 should be clamped to 28.
	source := semimonthlySource(t, "Job", []int{15, 29}, ptrFloat64(1000.0))

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still be exactly 2 pay dates in Feb (15th and 28th), no surplus.
	if len(result.SurplusMonths) != 0 {
		t.Errorf("expected no surplus in non-leap Feb with clamped day, got %d", len(result.SurplusMonths))
	}
}

// ---------------------------------------------------------------------------
// TestGeneratePayDatesForYear_InvalidSchedule
// ---------------------------------------------------------------------------

func TestGeneratePayDatesForYear_InvalidSchedule(t *testing.T) {
	d := NewSurplusDetector()

	source := models.IncomeSource{
		Name:           "Bad",
		PaySchedule:    "unknown",
		ScheduleDetail: json.RawMessage(`{}`),
	}

	_, err := d.GeneratePayDatesForYear(source, 2025)
	if err == nil {
		t.Error("expected error for unknown schedule, got nil")
	}
}

// ---------------------------------------------------------------------------
// TestDetect_FullYear2026 — different year to verify generality
// ---------------------------------------------------------------------------

func TestDetect_FullYear2026(t *testing.T) {
	d := NewSurplusDetector()

	// Thursday weekly for 2026.
	// 2026-01-01 is a Thursday, so Jan has 5 Thursdays (1,8,15,22,29).
	source := weeklySource(t, "Job", 4, ptrFloat64(800.0))

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	result, err := d.Detect([]models.IncomeSource{source}, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2026-01-01 is a Thursday. Months with 5 Thursdays: Jan, Apr, Jul, Oct, Dec = 5 extra checks.
	totalExtra := 0
	for _, sm := range result.SurplusMonths {
		totalExtra += sm.ExtraChecks
	}
	if totalExtra != 5 {
		t.Errorf("expected 5 total extra checks in 2026 for Thursday weekly, got %d", totalExtra)
	}

	if !floatEqual(result.AnnualSurplus, 5*800.0) {
		t.Errorf("expected annual surplus %.2f, got %.2f", 5*800.0, result.AnnualSurplus)
	}
}

// ---------------------------------------------------------------------------
// TestScheduleDescription_InvalidJSON — handles malformed JSON gracefully
// ---------------------------------------------------------------------------

func TestScheduleDescription_InvalidJSON(t *testing.T) {
	// The function calls json.Unmarshal and ignores the error.
	// With invalid JSON, the struct fields will be zero-valued.
	t.Run("weekly invalid json", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule:    "weekly",
			ScheduleDetail: json.RawMessage(`{bad`),
		}
		got := ScheduleDescription(source)
		// Weekday 0 = Sunday
		if got != "Every Sunday" {
			t.Errorf("ScheduleDescription() = %q, want %q", got, "Every Sunday")
		}
	})

	t.Run("biweekly invalid json", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule:    "biweekly",
			ScheduleDetail: json.RawMessage(`{bad`),
		}
		got := ScheduleDescription(source)
		if got != "Every other Sunday" {
			t.Errorf("ScheduleDescription() = %q, want %q", got, "Every other Sunday")
		}
	})

	t.Run("semimonthly invalid json", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule:    "semimonthly",
			ScheduleDetail: json.RawMessage(`{bad`),
		}
		got := ScheduleDescription(source)
		// Days will be nil, len(nil) == 0, so fallback to "Twice monthly"
		if got != "Twice monthly" {
			t.Errorf("ScheduleDescription() = %q, want %q", got, "Twice monthly")
		}
	})
}
