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

// toCents rounds a dollar amount to whole cents.
func toCents(v float64) int64 {
	if v >= 0 {
		return int64(v*100 + 0.5)
	}
	return -int64(-v*100 + 0.5)
}

func fromCents(c int64) float64 { return float64(c) / 100 }

// PlanSinkingFund computes a dry-run sinking fund plan given a bill amount and
// a slice of candidate periods (already ordered oldest-first and limited to N).
// Nothing is written to the database.
//
// The arithmetic runs in integer cents so that repeated division never loses a
// fraction of a cent per period. Allocation is two passes: an even split capped
// by each period's headroom, then a redistribution of whatever the capped
// periods could not absorb onto the periods that still have room. Without the
// second pass a single tight period turns into a reported shortfall even when
// later paychecks have ample surplus.
func PlanSinkingFund(billAmount float64, periods []SinkingFundPeriod) *SinkingFundPlan {
	if len(periods) == 0 {
		return &SinkingFundPlan{
			Installments: []SinkingFundInstallment{},
			TotalNeeded:  billAmount,
			Shortfall:    billAmount,
		}
	}

	needed := toCents(billAmount)
	if needed < 0 {
		needed = 0
	}

	n := len(periods)
	surplus := make([]float64, n)  // reported as-is (income - assigned)
	headroom := make([]int64, n)   // what may actually be reserved, in cents
	allocated := make([]int64, n)

	for i, p := range periods {
		surplus[i] = p.Income - p.Assigned
		avail := toCents(surplus[i]) - toCents(sinkingFundBuffer)
		if avail < 0 {
			avail = 0
		}
		headroom[i] = avail
	}

	// Pass 1: even split, capped by headroom. Remainder cents go to the
	// earliest periods so the plan front-loads rather than trailing off.
	base := needed / int64(n)
	remainder := needed % int64(n)
	for i := 0; i < n; i++ {
		want := base
		if int64(i) < remainder {
			want++
		}
		if want > headroom[i] {
			want = headroom[i]
		}
		allocated[i] = want
	}

	// Pass 2: push whatever is still unfunded onto periods with room left.
	var funded int64
	for _, a := range allocated {
		funded += a
	}
	for i := 0; i < n && funded < needed; i++ {
		room := headroom[i] - allocated[i]
		if room <= 0 {
			continue
		}
		take := needed - funded
		if take > room {
			take = room
		}
		allocated[i] += take
		funded += take
	}

	installments := make([]SinkingFundInstallment, 0, n)
	for i, p := range periods {
		installments = append(installments, SinkingFundInstallment{
			PeriodID: p.ID,
			PayDate:  p.PayDate,
			Surplus:  surplus[i],
			Amount:   fromCents(allocated[i]),
		})
	}

	shortfall := needed - funded
	if shortfall < 0 {
		shortfall = 0
	}

	return &SinkingFundPlan{
		Installments: installments,
		TotalFunded:  fromCents(funded),
		TotalNeeded:  billAmount,
		Shortfall:    fromCents(shortfall),
	}
}
