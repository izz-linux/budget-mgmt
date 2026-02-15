package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

// helper to build a date without time components.
func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// helper to marshal schedule detail to json.RawMessage.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal schedule detail: %v", err)
	}
	return b
}

// helper to build an IncomeSource with the given pay schedule and detail.
func makeSource(t *testing.T, schedule string, detail interface{}) models.IncomeSource {
	t.Helper()
	return models.IncomeSource{
		PaySchedule:    schedule,
		ScheduleDetail: mustMarshal(t, detail),
	}
}

// assertDates compares generated dates against expected dates and reports mismatches.
func assertDates(t *testing.T, got []time.Time, expected []time.Time) {
	t.Helper()
	if len(got) != len(expected) {
		t.Errorf("expected %d dates, got %d", len(expected), len(got))
		t.Logf("  expected: %v", expected)
		t.Logf("  got:      %v", got)
		return
	}
	for i := range expected {
		if !got[i].Equal(expected[i]) {
			t.Errorf("date[%d]: expected %s, got %s", i, expected[i].Format("2006-01-02"), got[i].Format("2006-01-02"))
		}
	}
}

func TestNewPeriodGenerator(t *testing.T) {
	gen := NewPeriodGenerator()
	if gen == nil {
		t.Fatal("NewPeriodGenerator() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Weekly schedule tests
// ---------------------------------------------------------------------------

func TestGenerateWeekly_Friday(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5}) // Friday

	from := date(2025, time.January, 1)  // Wednesday
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 3),
		date(2025, time.January, 10),
		date(2025, time.January, 17),
		date(2025, time.January, 24),
		date(2025, time.January, 31),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_Monday(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 1}) // Monday

	from := date(2025, time.March, 1)  // Saturday
	to := date(2025, time.March, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.March, 3),
		date(2025, time.March, 10),
		date(2025, time.March, 17),
		date(2025, time.March, 24),
		date(2025, time.March, 31),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_Sunday(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 0}) // Sunday

	from := date(2025, time.February, 1)  // Saturday
	to := date(2025, time.February, 28)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.February, 2),
		date(2025, time.February, 9),
		date(2025, time.February, 16),
		date(2025, time.February, 23),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_FromIsTargetWeekday(t *testing.T) {
	gen := NewPeriodGenerator()
	// Wednesday = 3. 2025-01-01 is a Wednesday.
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 3})

	from := date(2025, time.January, 1) // Wednesday
	to := date(2025, time.January, 22)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 1),
		date(2025, time.January, 8),
		date(2025, time.January, 15),
		date(2025, time.January, 22),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_ToIsTargetWeekday(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5}) // Friday

	from := date(2025, time.January, 1)
	to := date(2025, time.January, 10) // Friday

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 3),
		date(2025, time.January, 10),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_SingleDay_Match(t *testing.T) {
	gen := NewPeriodGenerator()
	// 2025-01-03 is a Friday
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5})

	from := date(2025, time.January, 3)
	to := date(2025, time.January, 3)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 3),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_SingleDay_NoMatch(t *testing.T) {
	gen := NewPeriodGenerator()
	// 2025-01-02 is Thursday, looking for Friday
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5})

	from := date(2025, time.January, 2)
	to := date(2025, time.January, 2)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates, got %d: %v", len(dates), dates)
	}
}

func TestGenerateWeekly_NarrowRangeNoMatch(t *testing.T) {
	gen := NewPeriodGenerator()
	// Mon-Thu range, looking for Friday
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5})

	from := date(2025, time.January, 6) // Monday
	to := date(2025, time.January, 9)   // Thursday

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates, got %d", len(dates))
	}
}

func TestGenerateWeekly_Saturday(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 6}) // Saturday

	from := date(2025, time.June, 1) // Sunday
	to := date(2025, time.June, 30)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.June, 7),
		date(2025, time.June, 14),
		date(2025, time.June, 21),
		date(2025, time.June, 28),
	}
	assertDates(t, dates, expected)
}

func TestGenerateWeekly_InvalidJSON(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "weekly",
		ScheduleDetail: json.RawMessage(`{invalid`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGenerateWeekly_SpansMultipleMonths(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 2}) // Tuesday

	from := date(2025, time.January, 27) // Monday
	to := date(2025, time.February, 11)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 28),
		date(2025, time.February, 4),
		date(2025, time.February, 11),
	}
	assertDates(t, dates, expected)
}

// ---------------------------------------------------------------------------
// Biweekly schedule tests
// ---------------------------------------------------------------------------

func TestGenerateBiweekly_Basic(t *testing.T) {
	gen := NewPeriodGenerator()
	// Anchor on 2025-01-03 (Friday), looking forward
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2025, time.January, 1)
	to := date(2025, time.February, 28)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 3),
		date(2025, time.January, 17),
		date(2025, time.January, 31),
		date(2025, time.February, 14),
		date(2025, time.February, 28),
	}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_AnchorIsFrom(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2025, time.January, 3)
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 3),
		date(2025, time.January, 17),
		date(2025, time.January, 31),
	}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_AnchorInPast(t *testing.T) {
	gen := NewPeriodGenerator()
	// Anchor was 2024-12-06, querying Feb 2025
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2024-12-06",
	})

	from := date(2025, time.February, 1)
	to := date(2025, time.February, 28)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2024-12-06 + 14*4 = 2025-01-31, +14 = 2025-02-14, +14 = 2025-02-28
	expected := []time.Time{
		date(2025, time.February, 14),
		date(2025, time.February, 28),
	}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_AnchorInFuture(t *testing.T) {
	gen := NewPeriodGenerator()
	// Anchor is 2025-03-07, but we query January 2025
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-03-07",
	})

	from := date(2025, time.January, 1)
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Working backwards from 2025-03-07 by 14-day intervals:
	// 2025-03-07 - 14 = 2025-02-21
	// 2025-02-21 - 14 = 2025-02-07
	// 2025-02-07 - 14 = 2025-01-24
	// 2025-01-24 - 14 = 2025-01-10
	// 2025-01-10 - 14 = 2024-12-27
	expected := []time.Time{
		date(2025, time.January, 10),
		date(2025, time.January, 24),
	}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_FromOnCycleBoundary(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	// From is exactly on a biweekly date
	from := date(2025, time.January, 17)
	to := date(2025, time.February, 14)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 17),
		date(2025, time.January, 31),
		date(2025, time.February, 14),
	}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_NoMatchInRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	// A range that falls between two biweekly dates
	from := date(2025, time.January, 4)
	to := date(2025, time.January, 16)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates, got %d: %v", len(dates), dates)
	}
}

func TestGenerateBiweekly_InvalidJSON(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "biweekly",
		ScheduleDetail: json.RawMessage(`not json`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGenerateBiweekly_InvalidAnchorDate(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "not-a-date",
	})

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for invalid anchor date, got nil")
	}
}

func TestGenerateBiweekly_SingleDayMatch(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2025, time.January, 3)
	to := date(2025, time.January, 3)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{date(2025, time.January, 3)}
	assertDates(t, dates, expected)
}

func TestGenerateBiweekly_LongRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 365 days / 14 = 26.07, so we expect 26 or 27 pay dates
	if len(dates) < 26 || len(dates) > 27 {
		t.Errorf("expected 26-27 dates in a year, got %d", len(dates))
	}

	// Verify all dates are 14 days apart
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1])
		if diff != 14*24*time.Hour {
			t.Errorf("dates[%d] - dates[%d] = %v, expected 14 days", i, i-1, diff)
		}
	}
}

// ---------------------------------------------------------------------------
// Semi-monthly schedule tests
// ---------------------------------------------------------------------------

func TestGenerateSemiMonthly_1stAnd16th(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 1)
	to := date(2025, time.March, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 1),
		date(2025, time.January, 16),
		date(2025, time.February, 1),
		date(2025, time.February, 16),
		date(2025, time.March, 1),
		date(2025, time.March, 16),
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_15thAnd30th(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 30}})

	from := date(2025, time.January, 1)
	to := date(2025, time.March, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 15),
		date(2025, time.January, 30),
		date(2025, time.February, 15),
		date(2025, time.February, 28), // clamped: Feb has 28 days in 2025
		date(2025, time.March, 15),
		date(2025, time.March, 30),
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_EndOfMonthClamping_Feb28(t *testing.T) {
	gen := NewPeriodGenerator()
	// Day 31 should clamp to last day of month
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 31}})

	from := date(2025, time.February, 1)
	to := date(2025, time.February, 28)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.February, 15),
		date(2025, time.February, 28), // 31 clamped to 28
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_EndOfMonthClamping_Feb29_LeapYear(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 31}})

	// 2024 is a leap year
	from := date(2024, time.February, 1)
	to := date(2024, time.February, 29)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2024, time.February, 15),
		date(2024, time.February, 29), // 31 clamped to 29 in leap year
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_EndOfMonthClamping_April30(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 31}})

	from := date(2025, time.April, 1)
	to := date(2025, time.April, 30)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.April, 15),
		date(2025, time.April, 30), // 31 clamped to 30
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_FromMidMonth(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 10)
	to := date(2025, time.February, 28)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Jan 1 is before from, so excluded
	expected := []time.Time{
		date(2025, time.January, 16),
		date(2025, time.February, 1),
		date(2025, time.February, 16),
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_ToMidMonth(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 1)
	to := date(2025, time.February, 10)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 1),
		date(2025, time.January, 16),
		date(2025, time.February, 1),
		// Feb 16 is after to, excluded
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_SingleDayRange_Match(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 16)
	to := date(2025, time.January, 16)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{date(2025, time.January, 16)}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_SingleDayRange_NoMatch(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 10)
	to := date(2025, time.January, 10)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates, got %d: %v", len(dates), dates)
	}
}

func TestGenerateSemiMonthly_InvalidDayCount_One(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1}})

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for 1 day in semimonthly schedule, got nil")
	}
}

func TestGenerateSemiMonthly_InvalidDayCount_Three(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 10, 20}})

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for 3 days in semimonthly schedule, got nil")
	}
}

func TestGenerateSemiMonthly_InvalidDayCount_Zero(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{}})

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for 0 days in semimonthly schedule, got nil")
	}
}

func TestGenerateSemiMonthly_InvalidJSON(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "semimonthly",
		ScheduleDetail: json.RawMessage(`???`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGenerateSemiMonthly_CrossYearBoundary(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2024, time.December, 1)
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2024, time.December, 1),
		date(2024, time.December, 16),
		date(2025, time.January, 1),
		date(2025, time.January, 16),
	}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_MultipleMonthlyClamping(t *testing.T) {
	gen := NewPeriodGenerator()
	// 31st should clamp differently across months
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 31}})

	from := date(2025, time.January, 1)
	to := date(2025, time.April, 30)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{
		date(2025, time.January, 1),
		date(2025, time.January, 31), // 31 days
		date(2025, time.February, 1),
		date(2025, time.February, 28), // clamped
		date(2025, time.March, 1),
		date(2025, time.March, 31), // 31 days
		date(2025, time.April, 1),
		date(2025, time.April, 30), // clamped (April has 30)
	}
	assertDates(t, dates, expected)
}

// ---------------------------------------------------------------------------
// Unknown schedule type
// ---------------------------------------------------------------------------

func TestGenerate_UnknownSchedule(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "monthly",
		ScheduleDetail: json.RawMessage(`{}`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for unknown schedule, got nil")
	}
}

func TestGenerate_EmptyScheduleType(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "",
		ScheduleDetail: json.RawMessage(`{}`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.January, 31))
	if err == nil {
		t.Fatal("expected error for empty schedule type, got nil")
	}
}

// ---------------------------------------------------------------------------
// Edge cases: from == to (same day)
// ---------------------------------------------------------------------------

func TestGenerate_FromEqualsTo_Weekly_Match(t *testing.T) {
	gen := NewPeriodGenerator()
	// 2025-01-03 is a Friday
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5})

	d := date(2025, time.January, 3)
	dates, err := gen.Generate(source, d, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []time.Time{d}
	assertDates(t, dates, expected)
}

func TestGenerate_FromEqualsTo_Biweekly_Match(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	d := date(2025, time.January, 17) // exactly 14 days after anchor
	dates, err := gen.Generate(source, d, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []time.Time{d}
	assertDates(t, dates, expected)
}

func TestGenerate_FromEqualsTo_SemiMonthly_Match(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	d := date(2025, time.January, 1)
	dates, err := gen.Generate(source, d, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []time.Time{d}
	assertDates(t, dates, expected)
}

// ---------------------------------------------------------------------------
// Table-driven tests for comprehensive weekday coverage
// ---------------------------------------------------------------------------

func TestGenerateWeekly_AllWeekdays(t *testing.T) {
	gen := NewPeriodGenerator()

	// 2025-01-05 is a Sunday (weekday=0)
	// We pick a range that contains exactly one of each weekday
	tests := []struct {
		name       string
		weekday    int
		from       time.Time
		to         time.Time
		firstMatch time.Time
	}{
		{"Sunday", 0, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 5)},
		{"Monday", 1, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 6)},
		{"Tuesday", 2, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 7)},
		{"Wednesday", 3, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 8)},
		{"Thursday", 4, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 9)},
		{"Friday", 5, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 10)},
		{"Saturday", 6, date(2025, time.January, 5), date(2025, time.January, 11), date(2025, time.January, 11)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: tc.weekday})
			dates, err := gen.Generate(source, tc.from, tc.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(dates) != 1 {
				t.Fatalf("expected 1 date, got %d: %v", len(dates), dates)
			}
			if !dates[0].Equal(tc.firstMatch) {
				t.Errorf("expected %s, got %s", tc.firstMatch.Format("2006-01-02"), dates[0].Format("2006-01-02"))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Biweekly: verify anchor alignment invariant
// ---------------------------------------------------------------------------

func TestGenerateBiweekly_AlignmentInvariant(t *testing.T) {
	gen := NewPeriodGenerator()
	anchor := date(2025, time.January, 3) // Friday
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2024, time.January, 1)
	to := date(2026, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every generated date must be an even multiple of 14 days from the anchor
	for _, d := range dates {
		daysDiff := d.Sub(anchor).Hours() / 24
		cycles := int(daysDiff) % 14
		if cycles != 0 {
			t.Errorf("date %s is not aligned with anchor %s (diff=%v days, remainder=%d)",
				d.Format("2006-01-02"), anchor.Format("2006-01-02"), daysDiff, cycles)
		}
	}
}

// ---------------------------------------------------------------------------
// Semi-monthly: specific month length edge cases
// ---------------------------------------------------------------------------

func TestGenerateSemiMonthly_FebruaryLeapVsNonLeap(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 29}})

	// Non-leap year: Feb 2025 has 28 days, so 29 clamps to 28
	t.Run("NonLeapYear", func(t *testing.T) {
		from := date(2025, time.February, 1)
		to := date(2025, time.February, 28)
		dates, err := gen.Generate(source, from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []time.Time{
			date(2025, time.February, 15),
			date(2025, time.February, 28),
		}
		assertDates(t, dates, expected)
	})

	// Leap year: Feb 2024 has 29 days, 29 is exact
	t.Run("LeapYear", func(t *testing.T) {
		from := date(2024, time.February, 1)
		to := date(2024, time.February, 29)
		dates, err := gen.Generate(source, from, to)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := []time.Time{
			date(2024, time.February, 15),
			date(2024, time.February, 29),
		}
		assertDates(t, dates, expected)
	})
}

func TestGenerateSemiMonthly_Day31_AcrossMonths(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{15, 31}})

	tests := []struct {
		name     string
		month    time.Month
		year     int
		expected int // expected clamped day for "31"
	}{
		{"January_31", time.January, 2025, 31},
		{"February_28", time.February, 2025, 28},
		{"February_29_Leap", time.February, 2024, 29},
		{"March_31", time.March, 2025, 31},
		{"April_30", time.April, 2025, 30},
		{"May_31", time.May, 2025, 31},
		{"June_30", time.June, 2025, 30},
		{"July_31", time.July, 2025, 31},
		{"August_31", time.August, 2025, 31},
		{"September_30", time.September, 2025, 30},
		{"October_31", time.October, 2025, 31},
		{"November_30", time.November, 2025, 30},
		{"December_31", time.December, 2025, 31},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			from := date(tc.year, tc.month, 1)
			lastDay := date(tc.year, tc.month+1, 0)
			to := lastDay

			dates, err := gen.Generate(source, from, to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(dates) != 2 {
				t.Fatalf("expected 2 dates, got %d: %v", len(dates), dates)
			}

			if dates[1].Day() != tc.expected {
				t.Errorf("expected clamped day %d, got %d", tc.expected, dates[1].Day())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Empty date ranges (from > to)
// ---------------------------------------------------------------------------

func TestGenerateWeekly_EmptyRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5})

	from := date(2025, time.January, 31)
	to := date(2025, time.January, 1) // to before from

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates for reversed range, got %d", len(dates))
	}
}

func TestGenerateBiweekly_EmptyRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2025-01-03",
	})

	from := date(2025, time.February, 28)
	to := date(2025, time.January, 1) // to before from

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates for reversed range, got %d", len(dates))
	}
}

func TestGenerateSemiMonthly_EmptyRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.March, 1)
	to := date(2025, time.January, 1) // to before from

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates for reversed range, got %d", len(dates))
	}
}

// ---------------------------------------------------------------------------
// Error message content checks
// ---------------------------------------------------------------------------

func TestGenerate_ErrorMessages(t *testing.T) {
	gen := NewPeriodGenerator()
	from := date(2025, time.January, 1)
	to := date(2025, time.January, 31)

	t.Run("UnknownScheduleContainsType", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule:    "quarterly",
			ScheduleDetail: json.RawMessage(`{}`),
		}
		_, err := gen.Generate(source, from, to)
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if errMsg != "unknown pay schedule: quarterly" {
			t.Errorf("unexpected error message: %s", errMsg)
		}
	})

	t.Run("WeeklyParseError", func(t *testing.T) {
		source := models.IncomeSource{
			PaySchedule:    "weekly",
			ScheduleDetail: json.RawMessage(`{"weekday": "not_a_number"}`),
		}
		_, err := gen.Generate(source, from, to)
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if len(errMsg) == 0 {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("BiweeklyAnchorDateError", func(t *testing.T) {
		source := makeSource(t, "biweekly", models.BiweeklySchedule{
			Weekday:    5,
			AnchorDate: "01/03/2025", // wrong format
		})
		_, err := gen.Generate(source, from, to)
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		if len(errMsg) == 0 {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("SemiMonthlyWrongDayCount", func(t *testing.T) {
		source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 10, 20}})
		_, err := gen.Generate(source, from, to)
		if err == nil {
			t.Fatal("expected error")
		}
		errMsg := err.Error()
		expected := "semimonthly schedule must have exactly 2 days, got 3"
		if errMsg != expected {
			t.Errorf("expected error %q, got %q", expected, errMsg)
		}
	})
}

// ---------------------------------------------------------------------------
// Biweekly: anchor far in the past
// ---------------------------------------------------------------------------

func TestGenerateBiweekly_AnchorFarInPast(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2020-01-03", // 5 years ago
	})

	from := date(2025, time.January, 1)
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still produce valid dates, and they should be 14 days apart
	if len(dates) == 0 {
		t.Fatal("expected some dates, got 0")
	}
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1])
		if diff != 14*24*time.Hour {
			t.Errorf("dates[%d] - dates[%d] = %v, expected 14 days", i, i-1, diff)
		}
	}

	// Verify alignment with anchor
	anchor := date(2020, time.January, 3)
	for _, d := range dates {
		daysDiff := d.Sub(anchor).Hours() / 24
		remainder := int(daysDiff) % 14
		if remainder != 0 {
			t.Errorf("date %s not aligned with anchor (remainder=%d)", d.Format("2006-01-02"), remainder)
		}
	}
}

// ---------------------------------------------------------------------------
// Biweekly: anchor far in the future
// ---------------------------------------------------------------------------

func TestGenerateBiweekly_AnchorFarInFuture(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "biweekly", models.BiweeklySchedule{
		Weekday:    5,
		AnchorDate: "2030-01-04", // 5 years in future
	})

	from := date(2025, time.January, 1)
	to := date(2025, time.January, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) == 0 {
		t.Fatal("expected some dates, got 0")
	}

	// All dates should be 14 days apart
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1])
		if diff != 14*24*time.Hour {
			t.Errorf("dates[%d] - dates[%d] = %v, expected 14 days", i, i-1, diff)
		}
	}

	// Verify alignment with anchor
	anchor := date(2030, time.January, 4)
	for _, d := range dates {
		daysDiff := d.Sub(anchor).Hours() / 24
		remainder := int(daysDiff) % 14
		if remainder != 0 && remainder != -0 {
			// Handle negative remainders
			if remainder < 0 {
				remainder += 14
			}
			if remainder != 0 {
				t.Errorf("date %s not aligned with anchor (remainder=%d)", d.Format("2006-01-02"), remainder)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Weekly: long range spanning a full year
// ---------------------------------------------------------------------------

func TestGenerateWeekly_FullYear(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "weekly", models.WeeklySchedule{Weekday: 5}) // Friday

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2025 has 52 Fridays (Jan 3 is first, Dec 26 is 52nd)
	if len(dates) != 52 {
		t.Errorf("expected 52 Fridays in 2025, got %d", len(dates))
	}

	// Verify all are Fridays
	for i, d := range dates {
		if d.Weekday() != time.Friday {
			t.Errorf("dates[%d] = %s is %s, expected Friday",
				i, d.Format("2006-01-02"), d.Weekday())
		}
	}

	// Verify 7-day spacing
	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1])
		if diff != 7*24*time.Hour {
			t.Errorf("dates[%d] - dates[%d] = %v, expected 7 days", i, i-1, diff)
		}
	}
}

// ---------------------------------------------------------------------------
// Semi-monthly: full year with 1st and 16th
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// One-time schedule tests
// ---------------------------------------------------------------------------

func TestGenerateOneTime_InRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2025-03-15"})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{date(2025, time.March, 15)}
	assertDates(t, dates, expected)
}

func TestGenerateOneTime_ExactMatch(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2025-06-01"})

	d := date(2025, time.June, 1)
	dates, err := gen.Generate(source, d, d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{d}
	assertDates(t, dates, expected)
}

func TestGenerateOneTime_OutOfRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2025-06-15"})

	from := date(2025, time.January, 1)
	to := date(2025, time.March, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates when out of range, got %d: %v", len(dates), dates)
	}
}

func TestGenerateOneTime_BeforeRange(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2024-12-31"})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dates) != 0 {
		t.Errorf("expected 0 dates when before range, got %d", len(dates))
	}
}

func TestGenerateOneTime_InvalidJSON(t *testing.T) {
	gen := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "one_time",
		ScheduleDetail: json.RawMessage(`{bad json`),
	}

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.December, 31))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGenerateOneTime_InvalidDate(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "not-a-date"})

	_, err := gen.Generate(source, date(2025, time.January, 1), date(2025, time.December, 31))
	if err == nil {
		t.Fatal("expected error for invalid date, got nil")
	}
}

func TestGenerateOneTime_OnFromBoundary(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2025-01-01"})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{date(2025, time.January, 1)}
	assertDates(t, dates, expected)
}

func TestGenerateOneTime_OnToBoundary(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "one_time", models.OneTimeSchedule{Date: "2025-12-31"})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []time.Time{date(2025, time.December, 31)}
	assertDates(t, dates, expected)
}

func TestGenerateSemiMonthly_FullYear(t *testing.T) {
	gen := NewPeriodGenerator()
	source := makeSource(t, "semimonthly", models.SemiMonthlySchedule{Days: []int{1, 16}})

	from := date(2025, time.January, 1)
	to := date(2025, time.December, 31)

	dates, err := gen.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 12 months * 2 = 24 pay dates
	if len(dates) != 24 {
		t.Errorf("expected 24 dates, got %d", len(dates))
	}

	// Verify dates alternate between 1st and 16th
	for i, d := range dates {
		if i%2 == 0 {
			if d.Day() != 1 {
				t.Errorf("dates[%d] = %s, expected day 1", i, d.Format("2006-01-02"))
			}
		} else {
			if d.Day() != 16 {
				t.Errorf("dates[%d] = %s, expected day 16", i, d.Format("2006-01-02"))
			}
		}
	}
}
