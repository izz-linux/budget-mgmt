package services

import (
	"sort"
)

type OptBill struct {
	ID     int
	Name   string
	DueDay int
	Amount float64
}

type OptPeriod struct {
	ID       int
	PayDate  string // YYYY-MM-DD
	PayDay   int    // day of month
	Income   float64
	Assigned float64
}

type OptAssignment struct {
	BillID       int
	PeriodID     int
	AssignmentID int // DB ID of the bill_assignment row
}

type Suggestion struct {
	AssignmentID int     `json:"assignment_id"` // DB ID of the assignment to move
	BillID       int     `json:"bill_id"`
	BillName     string  `json:"bill_name"`
	FromPeriodID int     `json:"from_period_id"`
	ToPeriodID   int     `json:"to_period_id"`
	FromPeriod   string  `json:"from_period"` // YYYY-MM-DD
	ToPeriod     string  `json:"to_period"`   // YYYY-MM-DD
	Amount       float64 `json:"amount"`
	Reason       string  `json:"reason"`
}

type OptimizationResult struct {
	Suggestions         []Suggestion `json:"suggestions"`
	CurrentMinBalance   float64      `json:"current_min_balance"`
	OptimizedMinBalance float64      `json:"optimized_min_balance"`
	Improvement         float64      `json:"improvement"`
}

type Optimizer struct{}

func NewOptimizer() *Optimizer {
	return &Optimizer{}
}

// Optimize takes bills and periods and suggests reassignments to maximize minimum balance.
// currentAssignments is a slice of all bill-to-period assignments (a bill may appear multiple
// times across different periods, e.g. once per month).
func (o *Optimizer) Optimize(bills []OptBill, periods []OptPeriod, currentAssignments []OptAssignment) *OptimizationResult {
	if len(bills) == 0 || len(periods) == 0 {
		return &OptimizationResult{Suggestions: []Suggestion{}}
	}

	// Sort periods by pay date
	sort.Slice(periods, func(i, j int) bool { return periods[i].PayDate < periods[j].PayDate })

	// Calculate current balances
	currentMin := calcMinBalance(bills, periods, currentAssignments)

	// Working copy of assignments
	optimized := make([]OptAssignment, len(currentAssignments))
	copy(optimized, currentAssignments)

	var suggestions []Suggestion

	for iterations := 0; iterations < 100; iterations++ {
		// Recalculate balances
		optBalances := calcBalances(bills, periods, optimized)

		// Find tightest and most surplus periods
		tightID, surplusID := 0, 0
		tightBal := 1e18
		surplusBal := -1e18
		for _, p := range periods {
			if optBalances[p.ID] < tightBal {
				tightBal = optBalances[p.ID]
				tightID = p.ID
			}
			if optBalances[p.ID] > surplusBal {
				surplusBal = optBalances[p.ID]
				surplusID = p.ID
			}
		}

		if tightID == surplusID || surplusBal-tightBal < 50 {
			break // Already balanced enough
		}

		// Find the best assignment in the tight period that can move to surplus
		bestImprovement := 0.0
		bestIdx := -1
		surplusPeriod := findPeriod(periods, surplusID)
		if surplusPeriod == nil {
			break
		}
		for i, a := range optimized {
			if a.PeriodID != tightID {
				continue
			}
			bill := findBill(bills, a.BillID)
			if bill == nil {
				continue
			}
			if !canPayFrom(surplusPeriod.PayDay, bill.DueDay) {
				continue
			}
			// Don't move if the bill is already assigned to the target period
			if hasBillInPeriod(optimized, a.BillID, surplusID) {
				continue
			}
			if bill.Amount > bestImprovement {
				bestImprovement = bill.Amount
				bestIdx = i
			}
		}

		if bestIdx < 0 {
			break // No valid moves
		}

		// Apply the move
		fromPeriod := findPeriod(periods, optimized[bestIdx].PeriodID)
		toPeriod := surplusPeriod
		bill := findBill(bills, optimized[bestIdx].BillID)

		suggestions = append(suggestions, Suggestion{
			AssignmentID: optimized[bestIdx].AssignmentID,
			BillID:       bill.ID,
			BillName:     bill.Name,
			FromPeriodID: fromPeriod.ID,
			ToPeriodID:   toPeriod.ID,
			FromPeriod:   fromPeriod.PayDate,
			ToPeriod:     toPeriod.PayDate,
			Amount:       bill.Amount,
			Reason:       "Rebalance: move from overloaded to surplus period",
		})

		optimized[bestIdx].PeriodID = surplusID
	}

	// Calculate optimized minimum balance
	optimizedMin := calcMinBalance(bills, periods, optimized)

	if suggestions == nil {
		suggestions = []Suggestion{}
	}

	return &OptimizationResult{
		Suggestions:         suggestions,
		CurrentMinBalance:   currentMin,
		OptimizedMinBalance: optimizedMin,
		Improvement:         optimizedMin - currentMin,
	}
}

func calcBalances(bills []OptBill, periods []OptPeriod, assignments []OptAssignment) map[int]float64 {
	balances := make(map[int]float64)
	for i := range periods {
		balances[periods[i].ID] = periods[i].Income
	}
	for _, a := range assignments {
		bill := findBill(bills, a.BillID)
		if bill != nil {
			balances[a.PeriodID] -= bill.Amount
		}
	}
	return balances
}

func calcMinBalance(bills []OptBill, periods []OptPeriod, assignments []OptAssignment) float64 {
	return minBalance(calcBalances(bills, periods, assignments))
}

func hasBillInPeriod(assignments []OptAssignment, billID, periodID int) bool {
	for _, a := range assignments {
		if a.BillID == billID && a.PeriodID == periodID {
			return true
		}
	}
	return false
}

func findBill(bills []OptBill, id int) *OptBill {
	for i := range bills {
		if bills[i].ID == id {
			return &bills[i]
		}
	}
	return nil
}

func findPeriod(periods []OptPeriod, id int) *OptPeriod {
	for i := range periods {
		if periods[i].ID == id {
			return &periods[i]
		}
	}
	return nil
}

func canPayFrom(payDay, dueDay int) bool {
	// Simple check: can pay from this paycheck if pay date is before or on due date
	// In practice this would need month awareness, but for within-month this works
	return payDay <= dueDay || dueDay == 0
}

func minBalance(balances map[int]float64) float64 {
	min := 1e18
	for _, b := range balances {
		if b < min {
			min = b
		}
	}
	if min == 1e18 {
		return 0
	}
	return min
}
