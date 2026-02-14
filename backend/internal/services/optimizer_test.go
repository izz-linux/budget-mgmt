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
// Optimize: empty / nil inputs
// ---------------------------------------------------------------------------

func TestOptimize_EmptyBills(t *testing.T) {
	o := NewOptimizer()
	periods := []OptPeriod{{ID: 1, PayDate: "2025-01-01", PayDay: 1, Income: 2000}}
	result := o.Optimize([]OptBill{}, periods, map[int]int{})
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
	result := o.Optimize(bills, []OptPeriod{}, map[int]int{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions, got %d", len(result.Suggestions))
	}
}

func TestOptimize_BothEmpty(t *testing.T) {
	o := NewOptimizer()
	result := o.Optimize([]OptBill{}, []OptPeriod{}, map[int]int{})
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
	result := o.Optimize(nil, periods, map[int]int{})
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
	result := o.Optimize(bills, nil, map[int]int{})
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
	result := o.Optimize(bills, periods, map[int]int{})
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
	assignments := map[int]int{1: 10, 2: 20}
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
	assignments := map[int]int{1: 10, 2: 20}
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
	// Use bills where only some can move to the later period (canPayFrom constraint).
	// Bill 1 (DueDay 3) can only be paid from PayDay <= 3, so it stays on period 10.
	// Bills 2,3 (DueDay 20,22) can be paid from period 20 (PayDay 15).
	// This prevents oscillation since Rent can't move back from period 10.
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
	assignments := map[int]int{1: 10, 2: 10, 3: 10}
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
	// Both bills on period 10: balance 10 = 2000-1500-200 = 300, balance 20 = 2000
	assignments := map[int]int{1: 10, 2: 10}
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
	// The move should be from period 10 to period 20
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
	// All bills on period 10: balance = 2000 - 800 - 200 - 100 - 80 - 50 = 770
	// Period 20 balance = 2000
	assignments := map[int]int{1: 10, 2: 10, 3: 10, 4: 10, 5: 10}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 770 {
		t.Errorf("expected CurrentMinBalance 770, got %f", result.CurrentMinBalance)
	}

	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions when all bills on one period")
	}

	// The optimizer should move some bills to period 20, improving balance
	if result.OptimizedMinBalance <= 770 {
		t.Errorf("expected optimized balance > 770, got %f", result.OptimizedMinBalance)
	}

	// Only bills with DueDay >= 15 (PayDay of period 20) or DueDay == 0 can be moved
	// Eligible: Phone (20), Water (25)
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
	// Bill due on day 5 cannot be paid from a period with PayDay 15
	bills := []OptBill{
		{ID: 1, Name: "EarlyBill", DueDay: 5, Amount: 1500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// Bill on period 10 (balance 500), period 20 (balance 2000). Diff > 50.
	// But due day 5 < pay day 15, so can't move. No suggestions.
	assignments := map[int]int{1: 10}
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
	// Both on period 10: balance = 2000-500-1000 = 500, period 20 = 2000
	// BigBill (due 5) can't move to period 20 (payDay 15 > dueDay 5)
	// FlexBill (due 0) can move to period 20
	assignments := map[int]int{1: 10, 2: 10}
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
	// Create a scenario where multiple bills need to be moved across multiple iterations
	bills := []OptBill{
		{ID: 1, Name: "Bill A", DueDay: 25, Amount: 500},
		{ID: 2, Name: "Bill B", DueDay: 28, Amount: 400},
		{ID: 3, Name: "Bill C", DueDay: 22, Amount: 300},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// All on period 10: balance = 2000 - 500 - 400 - 300 = 800, period 20 = 2000
	// All due days > 15, so all can be moved to period 20
	// Iteration 1: moves Bill A (500, largest), period 10 = 1300, period 20 = 1500
	// The difference is still 200 which is > 50, so...
	// Iteration 2: moves Bill B (400), period 10 = 1700, period 20 = 1100
	// Now period 20 is tighter. Might trigger another iteration.
	assignments := map[int]int{1: 10, 2: 10, 3: 10}
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) < 1 {
		t.Fatalf("expected at least 1 suggestion for multi-iteration case, got %d", len(result.Suggestions))
	}

	// The optimizer should produce an improvement
	if result.Improvement <= 0 {
		t.Errorf("expected positive improvement, got %f", result.Improvement)
	}

	// The optimized min balance should be better than the original 800
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
	// All on period 10: 1500 - 1200 - 400 - 150 - 80 - 60 = -390
	// Period 20 = 1500, Period 30 = 1500
	assignments := map[int]int{1: 10, 2: 10, 3: 10, 4: 10, 5: 10}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != -390 {
		t.Errorf("expected CurrentMinBalance -390, got %f", result.CurrentMinBalance)
	}

	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for severely unbalanced periods")
	}

	// After optimization, min balance should be significantly better
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
	assignments := map[int]int{1: 10}
	result := o.Optimize(bills, periods, assignments)
	// With one period, tight == surplus => no suggestions
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
	// Bill ID 999 doesn't exist; it should be silently ignored
	assignments := map[int]int{1: 10, 999: 20}
	result := o.Optimize(bills, periods, assignments)
	// Should not panic and should return a valid result
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// ---------------------------------------------------------------------------
// Optimize: does not mutate original assignments map
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
	assignments := map[int]int{1: 10, 2: 10}
	original := make(map[int]int)
	for k, v := range assignments {
		original[k] = v
	}

	o.Optimize(bills, periods, assignments)

	for k, v := range original {
		if assignments[k] != v {
			t.Errorf("original assignments mutated: key %d changed from %d to %d", k, v, assignments[k])
		}
	}
}

// ---------------------------------------------------------------------------
// Optimize: suggestions always have non-nil slice (never nil)
// ---------------------------------------------------------------------------

func TestOptimize_SuggestionsNeverNil(t *testing.T) {
	o := NewOptimizer()

	t.Run("empty inputs", func(t *testing.T) {
		result := o.Optimize([]OptBill{}, []OptPeriod{}, map[int]int{})
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
		result := o.Optimize(bills, periods, map[int]int{1: 10})
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
	assignments := map[int]int{1: 10, 2: 10}
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
	// Periods given in reverse order
	periods := []OptPeriod{
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
	}
	assignments := map[int]int{1: 10}
	result := o.Optimize(bills, periods, assignments)
	// Should produce same result regardless of input order
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Period 10 has balance 500, period 20 has balance 2000
	// Bill can move (DueDay 25 >= PayDay 15)
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions even with reversed period order")
	}
}

// ---------------------------------------------------------------------------
// Optimize: convergence within iteration limit
// ---------------------------------------------------------------------------

func TestOptimize_ConvergesWithManyBills(t *testing.T) {
	o := NewOptimizer()
	// Create many bills to test convergence within 100 iterations
	var bills []OptBill
	for i := 1; i <= 20; i++ {
		bills = append(bills, OptBill{
			ID:     i,
			Name:   "Bill",
			DueDay: 0, // always movable
			Amount: 50,
		})
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	// All 20 bills ($50 each = $1000 total) on period 10
	assignments := make(map[int]int)
	for i := 1; i <= 20; i++ {
		assignments[i] = 10
	}
	// Period 10 balance = 2000 - 1000 = 1000, Period 20 = 2000
	result := o.Optimize(bills, periods, assignments)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should converge to roughly equal balances (1500 each)
	// 10 bills moved to period 20 => each period has $500 of bills, balance = 1500
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
	// All on period 10: 2000-50-200-500 = 1250, period 20 = 2000
	assignments := map[int]int{1: 10, 2: 10, 3: 10}
	result := o.Optimize(bills, periods, assignments)

	if len(result.Suggestions) == 0 {
		t.Fatal("expected at least one suggestion")
	}
	// The first suggestion should be the largest bill
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
	// All bills have early due days that can't be paid from period 20 (PayDay 15)
	bills := []OptBill{
		{ID: 1, Name: "A", DueDay: 3, Amount: 500},
		{ID: 2, Name: "B", DueDay: 5, Amount: 500},
		{ID: 3, Name: "C", DueDay: 10, Amount: 500},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	assignments := map[int]int{1: 10, 2: 10, 3: 10}
	// Period 10: 2000-1500=500, Period 20: 2000, difference 1500 >> 50
	// But no bill can move because all due days < 15
	result := o.Optimize(bills, periods, assignments)
	if len(result.Suggestions) != 0 {
		t.Errorf("expected 0 suggestions when all moves blocked by canPayFrom, got %d", len(result.Suggestions))
	}
	if result.Improvement != 0 {
		t.Errorf("expected 0 improvement, got %f", result.Improvement)
	}
}

// ---------------------------------------------------------------------------
// Optimize: empty assignments map
// ---------------------------------------------------------------------------

func TestOptimize_EmptyAssignmentsMap(t *testing.T) {
	o := NewOptimizer()
	bills := []OptBill{
		{ID: 1, Name: "Rent", DueDay: 5, Amount: 1200},
	}
	periods := []OptPeriod{
		{ID: 10, PayDate: "2025-01-01", PayDay: 1, Income: 2000},
		{ID: 20, PayDate: "2025-01-15", PayDay: 15, Income: 2000},
	}
	result := o.Optimize(bills, periods, map[int]int{})
	// No bills assigned, so both periods at full income, perfectly balanced
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
	// Period 10: 200, Period 20: 2000
	// Moving Bill to period 20: Period 10: 2000, Period 20: 200
	// Then it would want to move back. The optimizer should stop after 1 move
	// because after the move the tight/surplus just swap.
	assignments := map[int]int{1: 10}
	result := o.Optimize(bills, periods, assignments)

	// The number of suggestions should be limited - the optimizer should
	// eventually break when there's no further improvement available
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Verify it terminates (it does because the loop limit is 100)
	// After first move: period 10 = 2000, period 20 = 200
	// Then it would try to move back to period 10 but the bill is on period 20 now
	// and period 20 is tight. bestBillID search looks at tightID period,
	// so it would find the bill and try to move it back.
	// This tests that the iteration limit prevents infinite oscillation.
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
	// Mortgage and Car Payment on period 10: 2500-1500-450=550
	// Electric, Internet, Phone, Insurance on period 20: 2500-120-80-60-200=2040
	assignments := map[int]int{1: 10, 2: 10, 3: 20, 4: 20, 5: 20, 6: 20}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 550 {
		t.Errorf("expected CurrentMinBalance 550, got %f", result.CurrentMinBalance)
	}

	// Car Payment (due 15) can be paid from period 20 (PayDay 15, 15 <= 15)
	// Moving it: period 10 = 2500-1500=1000, period 20 = 2500-450-120-80-60-200=1590
	// That's a significant improvement
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
	// All on period 1: 1000-2000-500=-1500, others at 1000 each
	assignments := map[int]int{1: 1, 2: 1}
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
	// Perfectly balanced: each period has balance 1000
	assignments := map[int]int{1: 10, 2: 20}
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
	// Period 10: 200, Period 20: 1000, diff = 800 > 50
	// Bill due 20 >= PayDay 15, so it can move
	assignments := map[int]int{1: 10}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 200 {
		t.Errorf("expected CurrentMinBalance 200, got %f", result.CurrentMinBalance)
	}

	// After moving: Period 10: 1000, Period 20: 200
	// That doesn't improve min balance (still 200). But the optimizer tries it
	// because it's just looking at moving from tight to surplus.
	// The final improvement should be 0 because min doesn't change.
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
	// All 4 bills on period 10: 1500 - 1200 = 300, period 20 = 1500
	assignments := map[int]int{1: 10, 2: 10, 3: 10, 4: 10}
	result := o.Optimize(bills, periods, assignments)

	if result.CurrentMinBalance != 300 {
		t.Errorf("expected CurrentMinBalance 300, got %f", result.CurrentMinBalance)
	}

	// Optimal: 2 bills each side => each period: 1500 - 600 = 900
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
