package services

// SinkingFundPeriod is a candidate period for a sinking fund installment.
// Handlers load these from the DB and pass them to the service.
type SinkingFundPeriod struct {
	ID       int
	PayDate  string  // YYYY-MM-DD
	Income   float64 // expected_amount for this period
	Assigned float64 // sum of existing planned_amounts in this period
}

// SinkingFundInstallment describes one period's reserved amount.
type SinkingFundInstallment struct {
	PeriodID int     `json:"period_id"`
	PayDate  string  `json:"pay_date"`
	Surplus  float64 `json:"surplus"` // available before reservation (income - assigned)
	Amount   float64 `json:"amount"`  // amount reserved by this installment
}

// SinkingFundPlan is the dry-run result returned before applying.
type SinkingFundPlan struct {
	Installments []SinkingFundInstallment `json:"installments"`
	TotalFunded  float64                  `json:"total_funded"`
	TotalNeeded  float64                  `json:"total_needed"`
	Shortfall    float64                  `json:"shortfall"` // 0 = fully covered
}

const sinkingFundBuffer = 50.0

// PlanSinkingFund computes a dry-run sinking fund plan given a bill amount and
// a slice of candidate periods (already ordered oldest-first and limited to N).
// Nothing is written to the database.
func PlanSinkingFund(billAmount float64, periods []SinkingFundPeriod) *SinkingFundPlan {
	if len(periods) == 0 {
		return &SinkingFundPlan{
			Installments: []SinkingFundInstallment{},
			TotalNeeded:  billAmount,
			Shortfall:    billAmount,
		}
	}

	idealPerPeriod := billAmount / float64(len(periods))

	var installments []SinkingFundInstallment
	var totalFunded float64

	for _, p := range periods {
		surplus := p.Income - p.Assigned
		available := surplus - sinkingFundBuffer
		if available < 0 {
			available = 0
		}
		installment := idealPerPeriod
		if installment > available {
			installment = available
		}
		// Truncate to cents
		installment = float64(int(installment*100)) / 100

		totalFunded += installment
		installments = append(installments, SinkingFundInstallment{
			PeriodID: p.ID,
			PayDate:  p.PayDate,
			Surplus:  surplus,
			Amount:   installment,
		})
	}

	shortfall := billAmount - totalFunded
	if shortfall < 0 {
		shortfall = 0
	}

	return &SinkingFundPlan{
		Installments: installments,
		TotalFunded:  float64(int(totalFunded*100)) / 100,
		TotalNeeded:  billAmount,
		Shortfall:    float64(int(shortfall*100)) / 100,
	}
}
