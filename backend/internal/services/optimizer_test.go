package services

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// NewOptimizer
// ---------------------------------------------------------------------------

func TestNewOptimizer(t *testing.T) {
	o := NewOptimizer()
	if o == nil {
		t.Fatal("NewOptimizer returned nil")
	}
}

// ---------------------------------------------------------------------------
// canPayFrom helper
// ---------------------------------------------------------------------------

func TestCanPayFrom(t *testing.T) {
	tests := []struct {
		name   string
		payDay int
		dueDay int
		want   bool
	}{
		{"pay before due", 1, 15, true},
		{"pay equals due", 15, 15, true},
		{"pay after due", 20, 15, false},
		{"due day is zero (always payable)", 20, 0, true},
		{"pay day 1 due day 0", 1, 0, true},
		{"pay day 31 due day 31", 31, 31, true},
		{"pay day 31 due day 1", 31, 1, false},
		{"pay day 1 due day 1", 1, 1, true},
		{"pay day 15 due day 14", 15, 14, false},
		{"pay day 0 due day 0", 0, 0, true},
		{"pay day 0 due day 15", 0, 15, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canPayFrom(tt.payDay, tt.dueDay)
			if got != tt.want {
				t.Errorf("canPayFrom(%d, %d) = %v, want %v", tt.payDay, tt.dueDay, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// findBill helper
// ---------------------------------------------------------------------------

func TestFindBill(t *testing.T) {
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 1, Amount: 1200},
		{ID: 2, Name: "Electric", DueDay: 15, Amount: 100},
		{ID: 3, Name: "Internet", DueDay: 20, Amount: 60},
	}

	t.Run("found", func(t *testing.T) {
		b := findBill(bills, 2)
		if b == nil {
			t.Fatal("expected bill, got nil")
		}
		if b.Name != "Electric" {
			t.Errorf("expected Electric, got %s", b.Name)
		}
	})

	t.Run("first element", func(t *testing.T) {
		b := findBill(bills, 1)
		if b == nil {
			t.Fatal("expected bill, got nil")
		}
		if b.Name != "Rent" {
			t.Errorf("expected Rent, got %s", b.Name)
		}
	})

	t.Run("last element", func(t *testing.T) {
		b := findBill(bills, 3)
		if b == nil {
			t.Fatal("expected bill, got nil")
		}
		if b.Name != "Internet" {
			t.Errorf("expected Internet, got %s", b.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		b := findBill(bills, 99)
		if b != nil {
			t.Errorf("expected nil, got %v", b)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		b := findBill([]OptBill{}, 1)
		if b != nil {
			t.Errorf("expected nil for empty slice, got %v", b)
		}
	})
}

// ---------------------------------------------------------------------------
// findPeriod helper
// ---------------------------------------------------------------------------

func TestFindPeriod(t *testing.T) {
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}

	t.Run("found", func(t *testing.T) {
		p := findPeriod(periods, 20)
		if p == nil {
			t.Fatal("expected period, got nil")
		}
		if p.PayDate != "2025-01-15" {
			t.Errorf("expected 2025-01-15, got %s", p.PayDate)
		}
	})

	t.Run("first element", func(t *testing.T) {
		p := findPeriod(periods, 10)
		if p == nil {
			t.Fatal("expected period, got nil")
		}
		if p.PayDay != 1 {
			t.Errorf("expected PayDay 1, got %d", p.PayDay)
		}
	})

	t.Run("not found", func(t *testing.T) {
		p := findPeriod(periods, 99)
		if p != nil {
			t.Errorf("expected nil, got %v", p)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		p := findPeriod([]OptPeriod{}, 10)
		if p != nil {
			t.Errorf("expected nil for empty slice, got %v", p)
		}
	})
}

// ---------------------------------------------------------------------------
// minBalance helper
// ---------------------------------------------------------------------------

func TestMinBalance(t *testing.T) {
	t.Run("empty map returns zero", func(t *testing.T) {
		got := minBalance(map[int]float64{})
		if got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})

	t.Run("single entry", func(t *testing.T) {
		got := minBalance(map[int]float64{1: 500})
		if got != 500 {
			t.Errorf("expected 500, got %f", got)
		}
	})

	t.Run("multiple entries picks minimum", func(t *testing.T) {
		got := minBalance(map[int]float64{1: 500, 2: 200, 3: 800})
		if got != 200 {
			t.Errorf("expected 200, got %f", got)
		}
	})

	t.Run("negative balances", func(t *testing.T) {
		got := minBalance(map[int]float64{1: -100, 2: 200})
		if got != -100 {
			t.Errorf("expected -100, got %f", got)
		}
	})

	t.Run("all negative", func(t *testing.T) {
		got := minBalance(map[int]float64{1: -100, 2: -300, 3: -50})
		if got != -300 {
			t.Errorf("expected -300, got %f", got)
		}
	})

	t.Run("all same value", func(t *testing.T) {
		got := minBalance(map[int]float64{1: 250, 2: 250, 3: 250})
		if got != 250 {
			t.Errorf("expected 250, got %f", got)
		}
	})

	t.Run("zero balance is valid minimum", func(t *testing.T) {
		got := minBalance(map[int]float64{1: 0, 2: 100})
		if got != 0 {
			t.Errorf("expected 0, got %f", got)
		}
	})
}

// ---------------------------------------------------------------------------
// hasBillInPeriod helper
// ---------------------------------------------------------------------------

func TestHasBillInPeriod(t *testing.T) {
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10},
		{BillID: 2, PeriodID: 20},
		{BillID: 1, PeriodID: 30},
	}

	t.Run("found", func(t *testing.T) {
		if !hasBillInPeriod(assignments, 1, 10) {
			t.Error("expected true for bill 1 in period 10")
		}
	})

	t.Run("same bill different period", func(t *testing.T) {
		if !hasBillInPeriod(assignments, 1, 30) {
			t.Error("expected true for bill 1 in period 30")
		}
	})

	t.Run("not found", func(t *testing.T) {
		if hasBillInPeriod(assignments, 2, 10) {
			t.Error("expected false for bill 2 in period 10")
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		if hasBillInPeriod(nil, 1, 10) {
			t.Error("expected false for empty slice")
		}
	})
}

// ---------------------------------------------------------------------------
// Optimize: empty / nil inputs
// ---------------------------------------------------------------------------

func TestOptimize_EmptyBills(t *testing.T) {
	o := NewOptimizer()
	periods := []OptPeriod{{ID: 1, PayDate: "2025-01-01", PayDay: 1, Income: 2000}}
	result := o.Optimize([]OptBill{}, periods, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestOptimize_EmptyPeriods(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{{ID: 1, Name: "Rent", DueDay: 1, Amount: 1200}}
	result := o.Optimize(bills, []OptPeriod{}, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestOptimize_BothEmpty(t *testing.T) {
	o := NewOptimizer()
	result := o.Optimize([]OptBill{}, []OptPeriod{}, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestOptimize_NilBills(t *testing.T) {
	o := NewOptimizer()
	periods := []OptPeriod{{ID: 1, PayDate: "2025-01-01", PayDay: 1, Income: 2000}}
	result := o.Optimize(nil, periods, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestOptimize_NilPeriods(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{{ID: 1, Name: "Rent", DueDay: 1, Amount: 1200}}
	result := o.Optimize(bills, nil, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

// ---------------------------------------------------------------------------
// Optimize: no assignments => no suggestions
// ---------------------------------------------------------------------------

func TestOptimize_NoAssignments(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 1, Amount: 1200},
		{ID: 2, Name: "Electric", DueDay: 15, Amount: 100},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	result := o.Optimize(bills, periods, nil)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions with no assignments, got %d", len(result.Suggestions))
	}
	// Both periods have full income, so min balance should be 2000
	if result.CurrentMinBalance != 2000 {
		t.Errorf("expected CurrentMinBalance 2000, got %f", result.CurrentMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: already balanced periods produce no suggestions
// ---------------------------------------------------------------------------

func TestOptimize_BalancedPeriods_NoSuggestions(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1000},
		{ID: 2, Name: "Electric", DueDay: 20, Amount: 1000},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// Each period has one $1000 bill => balance is 1000 each, difference < 50
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 20}}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions for balanced periods, got %d", len(result.Suggestions))
	}
	if result.CurrentMinBalance != 1000 {
		t.Errorf("expected CurrentMinBalance 1000, got %f", result.CurrentMinBalance)
	}
	if result.Improvement != 0 {
		t.Errorf("expected 0 improvement, got %f", result.Improvement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: difference under threshold (< 50) produces no suggestions
// ---------------------------------------------------------------------------

func TestOptimize_DifferenceBelowThreshold(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Bill A", DueDay: 5, Amount: 1000},
		{ID: 2, Name: "Bill B", DueDay: 20, Amount: 1020}, // only 20 difference
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 20}}
	// Period 10: 2000-1000=1000, Period 20: 2000-1020=980, diff=20 < 50
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions when difference < 50, got %d", len(result.Suggestions))
	}
}

// ---------------------------------------------------------------------------
// Optimize: unbalanced periods produce suggestions
// ---------------------------------------------------------------------------

func TestOptimize_UnbalancedPeriods_ProducesSuggestions(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 3, Amount: 1200},
		{ID: 2, Name: "Electric", DueDay: 20, Amount: 150},
		{ID: 3, Name: "Internet", DueDay: 22, Amount: 60},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// All bills on period 10: balance 10 = 2000-1200-150-60 = 590, balance 20 = 2000
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for unbalanced periods, got 0")
	}
	if result.Improvement <= 0 {
		t.Errorf("expected positive improvement, got %f", result.Improvement)
	}
	if result.OptimizedMinBalance <= result.CurrentMinBalance {
		t.Errorf("expected optimized min balance (%f) > current (%f)",
			result.OptimizedMinBalance, result.CurrentMinBalance)
	}
	if result.CurrentMinBalance != 590 {
		t.Errorf("expected CurrentMinBalance 590, got %f", result.CurrentMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: suggestion fields are correctly populated
// ---------------------------------------------------------------------------

func TestOptimize_SuggestionFields(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1500},
		{ID: 2, Name: "Electric", DueDay: 20, Amount: 200},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	s := result.Suggestions[0]
	if s.BillName == "" {
		t.Error("suggestion BillName should not be empty")
	}
	if s.FromPeriod == "" {
		t.Error("suggestion FromPeriod should not be empty")
	}
	if s.ToPeriod == "" {
		t.Error("suggestion ToPeriod should not be empty")
	}
	if s.Amount <= 0 {
		t.Errorf("suggestion Amount should be positive, got %f", s.Amount)
	}
	if s.Reason == "" {
		t.Error("suggestion Reason should not be empty")
	}
	if s.FromPeriod != "2025-01-01" {
		t.Errorf("expected FromPeriod 2025-01-01, got %s", s.FromPeriod)
	}
	if s.ToPeriod != "2025-01-15" {
		t.Errorf("expected ToPeriod 2025-01-15, got %s", s.ToPeriod)
	}
}

// ---------------------------------------------------------------------------
// Optimize: all bills on one period (edge case)
// ---------------------------------------------------------------------------

func TestOptimize_AllBillsOnOnePeriod(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 800},
		{ID: 2, Name: "Electric", DueDay: 10, Amount: 200},
		{ID: 3, Name: "Internet", DueDay: 12, Amount: 100},
		{ID: 4, Name: "Phone", DueDay: 20, Amount: 80},
		{ID: 5, Name: "Water", DueDay: 25, Amount: 50},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
		{BillID: 4, PeriodID: 10}, {BillID: 5, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 770 {
		t.Errorf("expected CurrentMinBalance 770, got %f", result.CurrentMinBalance)
	}
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions when all bills on one period")
	}
	if result.OptimizedMinBalance <= 770 {
		t.Errorf("expected optimized balance > 770, got %f", result.OptimizedMinBalance)
	}
	for _, s := range result.Suggestions {
		if s.BillName != "Phone" && s.BillName != "Water" {
			t.Errorf("only Phone and Water should be movable to period with PayDay 15, got %s", s.BillName)
		}
	}
}

// ---------------------------------------------------------------------------
// Optimize: canPayFrom constraint prevents invalid moves
// ---------------------------------------------------------------------------

func TestOptimize_CanPayFromBlocksMove(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "EarlyBill", DueDay: 5, Amount: 1500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions because canPayFrom blocks move, got %d", len(result.Suggestions))
	}
}

// ---------------------------------------------------------------------------
// Optimize: bill with DueDay 0 is always movable
// ---------------------------------------------------------------------------

func TestOptimize_DueDayZeroAlwaysMovable(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "FlexBill", DueDay: 0, Amount: 500},
		{ID: 2, Name: "BigBill", DueDay: 5, Amount: 1000},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for DueDay 0 bill")
	}
	if result.Suggestions[0].BillName != "FlexBill" {
		t.Errorf("expected FlexBill to be moved, got %s", result.Suggestions[0].BillName)
	}
}

// ---------------------------------------------------------------------------
// Optimize: multiple iterations of optimization
// ---------------------------------------------------------------------------

func TestOptimize_MultipleIterations(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Bill A", DueDay: 25, Amount: 500},
		{ID: 2, Name: "Bill B", DueDay: 28, Amount: 400},
		{ID: 3, Name: "Bill C", DueDay: 22, Amount: 300},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) < 1 {
		t.Fatalf("expected at least 1 suggestion for multi-iteration case, got %d", len(result.Suggestions))
	}
	if result.Improvement <= 0 {
		t.Errorf("expected positive improvement, got %f", result.Improvement)
	}
	if result.OptimizedMinBalance <= 800 {
		t.Errorf("expected OptimizedMinBalance > 800, got %f", result.OptimizedMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: three periods with multiple iterations
// ---------------------------------------------------------------------------

func TestOptimize_ThreePeriodsMultipleIterations(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1200},
		{ID: 2, Name: "Car", DueDay: 10, Amount: 400},
		{ID: 3, Name: "Electric", DueDay: 12, Amount: 150},
		{ID: 4, Name: "Phone", DueDay: 20, Amount: 80},
		{ID: 5, Name: "Internet", DueDay: 25, Amount: 60},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 1500},
		{ID: 20, PayDate: "2025-01-10", PayDay: 10, Income: 1500},
		{ID: 30, PayDate: "2025-01-20", PayDay: 20, Income: 1500},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
		{BillID: 4, PeriodID: 10}, {BillID: 5, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != -390 {
		t.Errorf("expected CurrentMinBalance -390, got %f", result.CurrentMinBalance)
	}
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for severely unbalanced periods")
	}
	if result.OptimizedMinBalance <= result.CurrentMinBalance {
		t.Errorf("expected optimization to improve min balance")
	}
}

// ---------------------------------------------------------------------------
// Optimize: single period means no moves possible
// ---------------------------------------------------------------------------

func TestOptimize_SinglePeriod(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1200},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions with single period, got %d", len(result.Suggestions))
	}
	if result.CurrentMinBalance != 800 {
		t.Errorf("expected CurrentMinBalance 800, got %f", result.CurrentMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: assignment references nonexistent bill (graceful handling)
// ---------------------------------------------------------------------------

func TestOptimize_AssignmentForNonexistentBill(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 999, PeriodID: 20},
	}
	result := o.Optimize(bills, periods, assignments)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// Optimize: does not mutate original assignments slice
// ---------------------------------------------------------------------------

func TestOptimize_DoesNotMutateOriginalAssignments(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Big", DueDay: 25, Amount: 1500},
		{ID: 2, Name: "Small", DueDay: 20, Amount: 100},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}}
	original := make([]OptAssignment, len(assignments))
	copy(original, assignments)

	o.Optimize(bills, periods, assignments)

	for i, a := range original {
		if assignments[i] != a {
			t.Errorf("original assignments mutated at index %d: was %v, now %v", i, a, assignments[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Optimize: suggestions always have non-nil slice (never nil)
// ---------------------------------------------------------------------------

func TestOptimize_SuggestionsNeverNil(t *testing.T) {
	o := NewOptimizer()

	t.Run("empty inputs", func(t *testing.T) {
		result := o.Optimize([]OptBill{}, []OptPeriod{}, nil)
		if result.Suggestions == nil {
			t.Error("Suggestions should be empty slice, not nil")
		}
	})

	t.Run("no improvement possible", func(t *testing.T) {
		bills := []OptBill{{ID: 1, Name: "A", DueDay: 5, Amount: 100}}
		periods := []OptPeriod{
			{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
			{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
		}
		result := o.Optimize(bills, periods, []OptAssignment{{BillID: 1, PeriodID: 10}})
		if result.Suggestions == nil {
			t.Error("Suggestions should be empty slice, not nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Optimize: improvement equals optimizedMin minus currentMin
// ---------------------------------------------------------------------------

func TestOptimize_ImprovementCalculation(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Bill A", DueDay: 20, Amount: 600},
		{ID: 2, Name: "Bill B", DueDay: 25, Amount: 400},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)

	expectedImprovement := result.OptimizedMinBalance - result.CurrentMinBalance
	if math.Abs(result.Improvement-expectedImprovement) > 0.001 {
		t.Errorf("Improvement (%f) should equal OptimizedMinBalance (%f) - CurrentMinBalance (%f) = %f",
			result.Improvement, result.OptimizedMinBalance, result.CurrentMinBalance, expectedImprovement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: periods are sorted by pay date internally
// ---------------------------------------------------------------------------

func TestOptimize_PeriodsAreSortedByPayDate(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Bill", DueDay: 25, Amount: 1500},
	}
	periods := []OptPeriod{
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions even with reversed period order")
	}
}

// ---------------------------------------------------------------------------
// Optimize: convergence within iteration limit
// ---------------------------------------------------------------------------

func TestOptimize_ConvergesWithManyBills(t *testing.T) {
	o := NewOptimizer()
	var bills []OptBill
	var assignments []OptAssignment
	for i := 1; i <= 20; i++ {
		bills = append(bills, OptBill{
			ID:     i,
			Name:   "Bill",
			DueDay: 0,
			Amount: 50,
		})
		assignments = append(assignments, OptAssignment{BillID: i, PeriodID: 10})
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	result := o.Optimize(bills, periods, assignments)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.OptimizedMinBalance < 1400 {
		t.Errorf("expected optimized balance near 1500, got %f", result.OptimizedMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: optimizer picks largest bill to move first
// ---------------------------------------------------------------------------

func TestOptimize_PicksLargestBillFirst(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Small", DueDay: 20, Amount: 50},
		{ID: 2, Name: "Medium", DueDay: 20, Amount: 200},
		{ID: 3, Name: "Large", DueDay: 20, Amount: 500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	if result.Suggestions[0].BillName != "Large" {
		t.Errorf("expected first suggestion to move Large bill, got %s", result.Suggestions[0].BillName)
	}
	if result.Suggestions[0].Amount != 500 {
		t.Errorf("expected first suggestion amount 500, got %f", result.Suggestions[0].Amount)
	}
}

// ---------------------------------------------------------------------------
// Optimize: no valid moves when canPayFrom blocks all bills
// ---------------------------------------------------------------------------

func TestOptimize_NoValidMovesAllBlocked(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "A", DueDay: 3, Amount: 500},
		{ID: 2, Name: "B", DueDay: 5, Amount: 500},
		{ID: 3, Name: "C", DueDay: 10, Amount: 500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10}, {BillID: 3, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions when all moves blocked by canPayFrom, got %d", len(result.Suggestions))
	}
	if result.Improvement != 0 {
		t.Errorf("expected 0 improvement, got %f", result.Improvement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: empty assignments
// ---------------------------------------------------------------------------

func TestOptimize_EmptyAssignmentsSlice(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1200},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	result := o.Optimize(bills, periods, []OptAssignment{})
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions with empty assignments, got %d", len(result.Suggestions))
	}
	if result.CurrentMinBalance != 2000 {
		t.Errorf("expected current min 2000, got %f", result.CurrentMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: oscillation prevention (move doesn't make things worse)
// ---------------------------------------------------------------------------

func TestOptimize_DoesNotOscillate(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Big", DueDay: 0, Amount: 1800},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) > 100 {
		t.Errorf("too many suggestions, possible infinite loop: %d", len(result.Suggestions))
	}
}

// ---------------------------------------------------------------------------
// Optimize: realistic biweekly paycheck scenario
// ---------------------------------------------------------------------------

func TestOptimize_RealisticBiweeklyScenario(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Mortgage", DueDay: 1, Amount: 1500},
		{ID: 2, Name: "Car Payment", DueDay: 15, Amount: 450},
		{ID: 3, Name: "Electric", DueDay: 20, Amount: 120},
		{ID: 4, Name: "Internet", DueDay: 22, Amount: 80},
		{ID: 5, Name: "Phone", DueDay: 25, Amount: 60},
		{ID: 6, Name: "Insurance", DueDay: 28, Amount: 200},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2500},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2500},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10},
		{BillID: 3, PeriodID: 20}, {BillID: 4, PeriodID: 20},
		{BillID: 5, PeriodID: 20}, {BillID: 6, PeriodID: 20},
	}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 550 {
		t.Errorf("expected CurrentMinBalance 550, got %f", result.CurrentMinBalance)
	}
	if result.Improvement <= 0 {
		t.Errorf("expected improvement in realistic scenario, got %f", result.Improvement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: large number of periods
// ---------------------------------------------------------------------------

func TestOptimize_ManyPeriods(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 0, Amount: 2000},
		{ID: 2, Name: "Utils", DueDay: 0, Amount: 500},
	}
	periods := []OptPeriod{
		{ID: 1, PayDate: "2025-01-01", PayDay: 1, Income: 1000},
		{ID: 2, PayDate: "2025-01-08", PayDay: 8, Income: 1000},
		{ID: 3, PayDate: "2025-01-15", PayDay: 15, Income: 1000},
		{ID: 4, PayDate: "2025-01-22", PayDay: 22, Income: 1000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 1}, {BillID: 2, PeriodID: 1}}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != -1500 {
		t.Errorf("expected CurrentMinBalance -1500, got %f", result.CurrentMinBalance)
	}
	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for heavily unbalanced 4-period scenario")
	}
	if result.OptimizedMinBalance <= result.CurrentMinBalance {
		t.Error("expected optimization to improve min balance")
	}
}

// ---------------------------------------------------------------------------
// Optimize: verify Improvement is zero when no suggestions are made
// ---------------------------------------------------------------------------

func TestOptimize_ZeroImprovementWhenNoSuggestions(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "A", DueDay: 5, Amount: 1000},
		{ID: 2, Name: "B", DueDay: 20, Amount: 1000},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 20}}
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) != 0 {
		t.Fatalf("expected no suggestions, got %d", len(result.Suggestions))
	}
	if result.Improvement != 0 {
		t.Errorf("expected 0 improvement when balanced, got %f", result.Improvement)
	}
	if result.CurrentMinBalance != result.OptimizedMinBalance {
		t.Errorf("expected current (%f) == optimized (%f) when balanced",
			result.CurrentMinBalance, result.OptimizedMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: single bill, two periods, bill is movable
// ---------------------------------------------------------------------------

func TestOptimize_SingleBillMovable(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "OnlyBill", DueDay: 20, Amount: 800},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 1000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 1000},
	}
	assignments := []OptAssignment{{BillID: 1, PeriodID: 10}}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 200 {
		t.Errorf("expected CurrentMinBalance 200, got %f", result.CurrentMinBalance)
	}
	if result.OptimizedMinBalance != 200 {
		t.Errorf("expected OptimizedMinBalance 200 (moving single bill just swaps), got %f",
			result.OptimizedMinBalance)
	}
}

// ---------------------------------------------------------------------------
// Optimize: verifies the optimizer handles equal income periods correctly
// ---------------------------------------------------------------------------

func TestOptimize_EqualIncomePeriods(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "A", DueDay: 0, Amount: 300},
		{ID: 2, Name: "B", DueDay: 0, Amount: 300},
		{ID: 3, Name: "C", DueDay: 0, Amount: 300},
		{ID: 4, Name: "D", DueDay: 0, Amount: 300},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 1500},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 1500},
	}
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, {BillID: 2, PeriodID: 10},
		{BillID: 3, PeriodID: 10}, {BillID: 4, PeriodID: 10},
	}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 300 {
		t.Errorf("expected CurrentMinBalance 300, got %f", result.CurrentMinBalance)
	}
	if result.OptimizedMinBalance != 900 {
		t.Errorf("expected OptimizedMinBalance 900, got %f", result.OptimizedMinBalance)
	}
	if len(result.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions (move 2 of 4 bills), got %d", len(result.Suggestions))
	}
	if result.Improvement != 600 {
		t.Errorf("expected improvement of 600, got %f", result.Improvement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: multi-month scenario (the bug that map[int]int caused)
// ---------------------------------------------------------------------------

func TestOptimize_MultiMonthNoDuplication(t *testing.T) {
	o := NewOptimizer()
	// A bill assigned to multiple periods (once per month) should all be tracked
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 1, Amount: 1200},
		{ID: 2, Name: "Electric", DueDay: 20, Amount: 150},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
		{ID: 30, PayDate: "2025-02-01", PayDay: 1, Income: 2000},
		{ID: 40, PayDate: "2025-02-15", PayDay: 15, Income: 2000},
	}
	// Both bills assigned each month
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10}, // Rent Jan
		{BillID: 2, PeriodID: 20}, // Electric Jan
		{BillID: 1, PeriodID: 30}, // Rent Feb
		{BillID: 2, PeriodID: 40}, // Electric Feb
	}
	result := o.Optimize(bills, periods, assignments)

	// All 4 assignments should be accounted for in balances:
	// P10: 2000-1200=800, P20: 2000-150=1850, P30: 2000-1200=800, P40: 2000-150=1850
	// Min = 800
	if result.CurrentMinBalance != 800 {
		t.Errorf("expected CurrentMinBalance 800 (all assignments counted), got %f", result.CurrentMinBalance)
	}

	// Suggestions should never move a bill to a period that already has it
	for _, s := range result.Suggestions {
		if s.BillName == "Rent" && s.ToPeriod == "2025-01-01" {
			t.Error("should not suggest moving Rent to period that already has it")
		}
		if s.BillName == "Rent" && s.ToPeriod == "2025-02-01" {
			t.Error("should not suggest moving Rent to period that already has it")
		}
		if s.BillName == "Electric" && s.ToPeriod == "2025-01-15" {
			t.Error("should not suggest moving Electric to period that already has it")
		}
		if s.BillName == "Electric" && s.ToPeriod == "2025-02-15" {
			t.Error("should not suggest moving Electric to period that already has it")
		}
	}
}

// ---------------------------------------------------------------------------
// Optimize: manually moved bill should not produce duplicate suggestions
// ---------------------------------------------------------------------------

func TestOptimize_ManuallyMovedBill(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1200},
		{ID: 2, Name: "Electric", DueDay: 20, Amount: 150},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// Electric already on period 20 (user manually moved it there)
	assignments := []OptAssignment{
		{BillID: 1, PeriodID: 10},
		{BillID: 2, PeriodID: 20},
	}
	// P10: 2000-1200=800, P20: 2000-150=1850, diff=1050 > 50
	// But Rent (dueDay 5) can't move to period 20 (payDay 15 > 5)
	// And Electric is already on the surplus period
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions for manually moved bill scenario, got %d", len(result.Suggestions))
	}
}
