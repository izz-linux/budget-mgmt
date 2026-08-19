package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Period generator: out-of-range weekday used to hang forever
// ---------------------------------------------------------------------------

// TestGenerateWeekly_OutOfRangeWeekdayDoesNotHang guards the worst bug in this
// package: generateWeekly advanced one day at a time until time.Weekday()
// matched the configured weekday, but Weekday() only ever returns 0-6, so a
// stored weekday outside that range never matched and the loop ran forever,
// pinning the request goroutine for the life of the process.
//
// The generator runs on its own goroutine behind a watchdog so that a
// regression fails this test instead of hanging the whole suite.
func TestGenerateWeekly_OutOfRangeWeekdayDoesNotHang(t *testing.T) {
	for _, weekday := range []int{-1, 7, 99} {
		t.Run(fmt.Sprintf("weekday=%d", weekday), func(t *testing.T) {
			g := NewPeriodGenerator()
			source := models.IncomeSource{
				PaySchedule:    "weekly",
				ScheduleDetail: json.RawMessage(fmt.Sprintf(`{"weekday":%d}`, weekday)),
			}
			from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

			type result struct {
				dates []time.Time
				err   error
			}
			done := make(chan result, 1)
			go func() {
				dates, err := g.Generate(source, from, to)
				done <- result{dates, err}
			}()

			select {
			case r := <-done:
				if r.err == nil {
					t.Fatalf("expected an error for weekday %d, got %d dates", weekday, len(r.dates))
				}
			case <-time.After(2 * time.Second):
				// Deliberately not t.Fatal from a helper goroutine — the
				// generator goroutine is stuck and will never return.
				t.Fatalf("generateWeekly did not return for weekday %d — infinite loop regression", weekday)
			}
		})
	}
}

func TestGenerateWeekly_ValidWeekdaysStillWork(t *testing.T) {
	g := NewPeriodGenerator()
	for weekday := 0; weekday <= 6; weekday++ {
		source := models.IncomeSource{
			PaySchedule:    "weekly",
			ScheduleDetail: json.RawMessage(fmt.Sprintf(`{"weekday":%d}`, weekday)),
		}
		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

		dates, err := g.Generate(source, from, to)
		if err != nil {
			t.Fatalf("weekday %d: unexpected error: %v", weekday, err)
		}
		if len(dates) == 0 {
			t.Fatalf("weekday %d: expected some pay dates", weekday)
		}
		for _, d := range dates {
			if int(d.Weekday()) != weekday {
				t.Errorf("weekday %d: got a date on %s", weekday, d.Weekday())
			}
		}
	}
}

func TestGenerateSemiMonthly_RejectsOutOfRangeDays(t *testing.T) {
	g := NewPeriodGenerator()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	// Day 0 is the dangerous one: time.Date normalises it back into the
	// previous month rather than erroring.
	for _, days := range [][]int{{0, 15}, {1, 0}, {-3, 15}, {1, 32}} {
		detail, _ := json.Marshal(map[string]any{"days": days})
		source := models.IncomeSource{PaySchedule: "semimonthly", ScheduleDetail: detail}

		if _, err := g.Generate(source, from, to); err == nil {
			t.Errorf("expected an error for days %v", days)
		}
	}
}

func TestGenerateSemiMonthly_ValidDaysStillWork(t *testing.T) {
	g := NewPeriodGenerator()
	source := models.IncomeSource{
		PaySchedule:    "semimonthly",
		ScheduleDetail: json.RawMessage(`{"days":[1,16]}`),
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	dates, err := g.Generate(source, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dates) != 6 {
		t.Fatalf("expected 6 dates over 3 months, got %d", len(dates))
	}
	for _, d := range dates {
		if d.Day() != 1 && d.Day() != 16 {
			t.Errorf("unexpected pay date %s", d.Format("2006-01-02"))
		}
		if d.Before(from) || d.After(to) {
			t.Errorf("date %s outside requested range", d.Format("2006-01-02"))
		}
	}
}

// ---------------------------------------------------------------------------
// Sinking fund planning
// ---------------------------------------------------------------------------

// period builds a candidate with the given headroom above the buffer.
func period(id int, day string, income, assigned float64) SinkingFundPeriod {
	return SinkingFundPeriod{ID: id, PayDate: day, Income: income, Assigned: assigned}
}

func totalAllocated(plan *SinkingFundPlan) float64 {
	var sum float64
	for _, i := range plan.Installments {
		sum += i.Amount
	}
	return sum
}

// assertNeverExceedsSurplus is the invariant that must hold for every plan: an
// installment may never reserve more than the period's available surplus.
func assertNeverExceedsSurplus(t *testing.T, plan *SinkingFundPlan) {
	t.Helper()
	for _, inst := range plan.Installments {
		available := inst.Surplus - sinkingFundBuffer
		if available < 0 {
			available = 0
		}
		if inst.Amount > available+1e-9 {
			t.Errorf("period %d reserved %.2f but only %.2f was available",
				inst.PeriodID, inst.Amount, available)
		}
	}
}

func TestPlanSinkingFund_EvenSplit(t *testing.T) {
	periods := []SinkingFundPeriod{
		period(1, "2026-01-01", 1000, 0),
		period(2, "2026-01-15", 1000, 0),
		period(3, "2026-02-01", 1000, 0),
	}

	plan := PlanSinkingFund(600, periods)

	if plan.Shortfall != 0 {
		t.Errorf("expected no shortfall, got %.2f", plan.Shortfall)
	}
	if plan.TotalFunded != 600 {
		t.Errorf("expected 600 funded, got %.2f", plan.TotalFunded)
	}
	for _, inst := range plan.Installments {
		if inst.Amount != 200 {
			t.Errorf("period %d: expected 200, got %.2f", inst.PeriodID, inst.Amount)
		}
	}
	assertNeverExceedsSurplus(t, plan)
}

// TestPlanSinkingFund_IndivisibleAmountFullyFunded covers the truncation bug:
// each installment used to be truncated to cents independently, losing a
// fraction of a cent per period so the total never reached the bill amount.
func TestPlanSinkingFund_IndivisibleAmountFullyFunded(t *testing.T) {
	periods := []SinkingFundPeriod{
		period(1, "2026-01-01", 1000, 0),
		period(2, "2026-01-15", 1000, 0),
		period(3, "2026-02-01", 1000, 0),
	}

	// 100.00 / 3 = 33.333... — must still add back up to exactly 100.00
	plan := PlanSinkingFund(100, periods)

	if got := totalAllocated(plan); got != 100 {
		t.Errorf("installments sum to %.2f, expected exactly 100.00", got)
	}
	if plan.TotalFunded != 100 {
		t.Errorf("expected TotalFunded 100.00, got %.2f", plan.TotalFunded)
	}
	if plan.Shortfall != 0 {
		t.Errorf("expected no shortfall, got %.2f", plan.Shortfall)
	}
	assertNeverExceedsSurplus(t, plan)
}

// TestPlanSinkingFund_RedistributesAroundZeroHeadroomPeriod is the headline
// case: $600 across 3 periods where the first has no headroom used to fund only
// $400 and report a $200 shortfall, even though the later two periods had
// plenty of surplus left.
func TestPlanSinkingFund_RedistributesAroundZeroHeadroomPeriod(t *testing.T) {
	periods := []SinkingFundPeriod{
		period(1, "2026-01-01", 1000, 1000), // fully committed: no headroom
		period(2, "2026-01-15", 1000, 0),
		period(3, "2026-02-01", 1000, 0),
	}

	plan := PlanSinkingFund(600, periods)

	if plan.Shortfall != 0 {
		t.Errorf("expected the later periods to absorb the shortfall, got %.2f", plan.Shortfall)
	}
	if got := totalAllocated(plan); got != 600 {
		t.Errorf("expected 600 allocated, got %.2f", got)
	}
	if plan.Installments[0].Amount != 0 {
		t.Errorf("period with no headroom should reserve 0, got %.2f", plan.Installments[0].Amount)
	}
	assertNeverExceedsSurplus(t, plan)
}

// TestPlanSinkingFund_GenuineShortfallStillReported makes sure redistribution
// did not paper over a real funding gap.
func TestPlanSinkingFund_GenuineShortfallStillReported(t *testing.T) {
	periods := []SinkingFundPeriod{
		period(1, "2026-01-01", 150, 0), // 100 available after the 50 buffer
		period(2, "2026-01-15", 150, 0), // 100 available
	}

	plan := PlanSinkingFund(600, periods)

	if plan.TotalFunded != 200 {
		t.Errorf("expected 200 funded, got %.2f", plan.TotalFunded)
	}
	if plan.Shortfall != 400 {
		t.Errorf("expected a 400 shortfall, got %.2f", plan.Shortfall)
	}
	assertNeverExceedsSurplus(t, plan)
}

func TestPlanSinkingFund_NeverExceedsAvailableSurplus(t *testing.T) {
	periods := []SinkingFundPeriod{
		period(1, "2026-01-01", 200, 100), // 100 surplus -> 50 available
		period(2, "2026-01-15", 60, 0),    // 60 surplus -> 10 available
		period(3, "2026-02-01", 40, 0),    // 40 surplus -> 0 available (below buffer)
		period(4, "2026-02-15", 5000, 0),  // plenty
	}

	plan := PlanSinkingFund(2000, periods)

	assertNeverExceedsSurplus(t, plan)
	if plan.Installments[2].Amount != 0 {
		t.Errorf("period below the buffer should reserve 0, got %.2f", plan.Installments[2].Amount)
	}
	if plan.Shortfall != 0 {
		t.Errorf("the last period had room for the remainder; got shortfall %.2f", plan.Shortfall)
	}
}

func TestPlanSinkingFund_NoPeriodsIsAllShortfall(t *testing.T) {
	plan := PlanSinkingFund(500, nil)

	if plan.Shortfall != 500 {
		t.Errorf("expected the full amount as shortfall, got %.2f", plan.Shortfall)
	}
	if len(plan.Installments) != 0 {
		t.Errorf("expected no installments, got %d", len(plan.Installments))
	}
}

// ---------------------------------------------------------------------------
// guessCategory determinism
// ---------------------------------------------------------------------------

// TestGuessCategory_StableAcrossRepeats pins the map-iteration bug: categories
// were stored in a map, and Go randomises map order, so a name matching two
// categories was filed differently between runs of the same import.
func TestGuessCategory_StableAcrossRepeats(t *testing.T) {
	imp := NewXLSXImporter()

	names := []string{
		"Car Insurance", // matches "car insurance" AND "insurance"
		"Car Payment",   // matches "car payment" AND (via "anna"? no) transportation
		"Home Insurance",
		"AL Power",
		"Hulu",
		"Mortgage",
		"Something Unmatched",
	}

	first := make(map[string]string, len(names))
	for _, n := range names {
		first[n] = imp.guessCategory(n)
	}

	// 500 repeats is far more than enough to surface a randomised map order.
	for i := 0; i < 500; i++ {
		for _, n := range names {
			if got := imp.guessCategory(n); got != first[n] {
				t.Fatalf("guessCategory(%q) returned %q on repeat %d but %q initially",
					n, got, i, first[n])
			}
		}
	}
}

// TestGuessCategory_SpecificRulesWinOverGeneral documents the intended
// precedence now that the rules are ordered.
func TestGuessCategory_SpecificRulesWinOverGeneral(t *testing.T) {
	imp := NewXLSXImporter()

	cases := map[string]string{
		"Car Insurance":  "transportation",
		"Car Payment":    "transportation",
		"Home Insurance": "insurance",
		"AL Power":       "utilities",
		"Hulu":           "subscriptions",
		"Mortgage":       "housing",
		"Totally Novel":  "other",
	}

	for name, want := range cases {
		if got := imp.guessCategory(name); got != want {
			t.Errorf("guessCategory(%q) = %q, want %q", name, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Surplus detection ordering
// ---------------------------------------------------------------------------

// TestDetect_MonthsAreChronologicalAndStable pins the second map-ordering bug:
// results were built by ranging over a map keyed by month, so the same request
// returned the months in a different order each time.
func TestDetect_MonthsAreChronologicalAndStable(t *testing.T) {
	d := NewSurplusDetector()
	sources := []models.IncomeSource{{
		Name:           "Job",
		PaySchedule:    "weekly",
		ScheduleDetail: json.RawMessage(`{"weekday":5}`),
		DefaultAmount:  float64Ptr(1000),
	}}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	first, err := d.Detect(sources, from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(first.SurplusMonths) < 2 {
		t.Fatalf("expected at least 2 five-paycheck months in a year, got %d", len(first.SurplusMonths))
	}

	// Chronological
	var prev time.Time
	for _, m := range first.SurplusMonths {
		parsed, err := time.Parse("January 2006", m.Month)
		if err != nil {
			t.Fatalf("unparseable month label %q: %v", m.Month, err)
		}
		if !prev.IsZero() && !parsed.After(prev) {
			t.Errorf("months out of order: %s came after %s", m.Month, prev.Format("January 2006"))
		}
		prev = parsed
	}

	// Stable across repeats
	for i := 0; i < 200; i++ {
		again, err := d.Detect(sources, from, to)
		if err != nil {
			t.Fatalf("unexpected error on repeat %d: %v", i, err)
		}
		if len(again.SurplusMonths) != len(first.SurplusMonths) {
			t.Fatalf("repeat %d returned %d months, first returned %d",
				i, len(again.SurplusMonths), len(first.SurplusMonths))
		}
		for j := range again.SurplusMonths {
			if again.SurplusMonths[j] != first.SurplusMonths[j] {
				t.Fatalf("repeat %d differs at index %d: %+v vs %+v",
					i, j, again.SurplusMonths[j], first.SurplusMonths[j])
			}
		}
	}
}

func float64Ptr(f float64) *float64 { return &f }
