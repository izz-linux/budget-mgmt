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
	BillID   int
	PeriodID int
}

type Suggestion struct {
	BillName   string  `json:"bill_name"`
	FromPeriod string  `json:"from_period"`
	ToPeriod   string  `json:"to_period"`
	Amount     float64 `json:"amount"`
	Reason     string  `json:"reason"`
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
func (o *Optimizer) Optimize(bills []OptBill, periods []OptPeriod, currentAssignments map[int]int) *OptimizationResult {
	if len(bills) == 0 || len(periods) == 0 {
		return &OptimizationResult{Suggestions: []Suggestion{}}
	}

	// Sort periods by pay date
	sort.Slice(periods, func(i, j int) bool { return periods[i].PayDate < periods[j].PayDate })

	// Calculate current balances
	periodBalances := make(map[int]float64)
	for i := range periods {
		periodBalances[periods[i].ID] = periods[i].Income
	}
	for billID, periodID := range currentAssignments {
		bill := findBill(bills, billID)
		if bill != nil {
			periodBalances[periodID] -= bill.Amount
		}
	}

	currentMin := minBalance(periodBalances)

	// Try to improve by moving bills from tight periods to surplus periods
	optimized := make(map[int]int)
	for k, v := range currentAssignments {
		optimized[k] = v
	}

	var suggestions []Suggestion

	for iterations := 0; iterations < 100; iterations++ {
		// Recalculate balances
		optBalances := make(map[int]float64)
		for i := range periods {
			optBalances[periods[i].ID] = periods[i].Income
		}
		for billID, periodID := range optimized {
			bill := findBill(bills, billID)
			if bill != nil {
				optBalances[periodID] -= bill.Amount
			}
		}

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

		// Find a bill currently assigned to tight period that can move to surplus
		bestImprovement := 0.0
		bestBillID := 0
		for billID, periodID := range optimized {
			if periodID != tightID {
				continue
			}
			bill := findBill(bills, billID)
			if bill == nil {
				continue
			}
			// Can this bill be paid from the surplus period? (pay date before due)
			surplusPeriod := findPeriod(periods, surplusID)
			if surplusPeriod == nil {
				continue
			}
			if canPayFrom(surplusPeriod.PayDay, bill.DueDay) {
				improvement := bill.Amount
				if improvement > bestImprovement {
					bestImprovement = improvement
					bestBillID = billID
				}
			}
		}

		if bestBillID == 0 {
			break // No valid moves
		}

		// Apply the move
		fromPeriod := findPeriod(periods, optimized[bestBillID])
		toPeriod := findPeriod(periods, surplusID)
		bill := findBill(bills, bestBillID)

		suggestions = append(suggestions, Suggestion{
			BillName:   bill.Name,
			FromPeriod: fromPeriod.PayDate,
			ToPeriod:   toPeriod.PayDate,
			Amount:     bill.Amount,
			Reason:     "Rebalance: move from overloaded to surplus period",
		})

		optimized[bestBillID] = surplusID
	}

	// Calculate optimized minimum balance
	optBalances := make(map[int]float64)
	for i := range periods {
		optBalances[periods[i].ID] = periods[i].Income
	}
	for billID, periodID := range optimized {
		bill := findBill(bills, billID)
		if bill != nil {
			optBalances[periodID] -= bill.Amount
		}
	}
	optimizedMin := minBalance(optBalances)

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
