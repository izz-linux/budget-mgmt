package handlers

import (
	"net/http"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type DashboardHandler struct {
	db DBTX
}

func NewDashboardHandler(db DBTX) *DashboardHandler {
	return &DashboardHandler{db: db}
}

type DashboardSummary struct {
	TotalIncome    float64             `json:"total_income"`
	TotalBills     float64             `json:"total_bills"`
	Remaining      float64             `json:"remaining"`
	PaidCount      int                 `json:"paid_count"`
	PendingCount   int                 `json:"pending_count"`
	UpcomingBills  []UpcomingBill      `json:"upcoming_bills"`
	PeriodSummaries []PeriodSummaryItem `json:"period_summaries"`
}

type UpcomingBill struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	DueDay      int     `json:"due_day"`
	Amount      float64 `json:"amount"`
	IsAutopay   bool    `json:"is_autopay"`
}

type PeriodSummaryItem struct {
	ID             int     `json:"id"`
	PayDate        string  `json:"pay_date"`
	SourceName     string  `json:"source_name"`
	ExpectedAmount float64 `json:"expected_amount"`
	TotalBills     float64 `json:"total_bills"`
	Remaining      float64 `json:"remaining"`
}

// upcomingDueDays returns the distinct day-of-month numbers covered by a window
// of `days` days starting at `from` (inclusive on both ends), so a window that
// crosses a month boundary still matches next month's early days.
func upcomingDueDays(from time.Time, days int) []int {
	seen := make(map[int]bool, days+1)
	var out []int
	for i := 0; i <= days; i++ {
		d := from.AddDate(0, 0, i).Day()
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	now := time.Now()
	from := now.Format("2006-01-02")
	to := now.AddDate(0, 2, 0).Format("2006-01-02")

	// Periods
	periodRows, err := h.db.Query(ctx, `
		SELECT pp.id, pp.pay_date, COALESCE(pp.expected_amount, 0), inc.name,
		       COALESCE(SUM(ba.planned_amount) FILTER (WHERE ab.id IS NOT NULL), 0) as total_bills
		FROM pay_periods pp
		JOIN income_sources inc ON inc.id = pp.income_source_id
		LEFT JOIN bill_assignments ba ON ba.pay_period_id = pp.id
		LEFT JOIN bills ab ON ab.id = ba.bill_id AND ab.is_active = true
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2 AND inc.is_active = true
		GROUP BY pp.id, inc.name
		ORDER BY pp.pay_date
	`, from, to)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer periodRows.Close()

	summary := DashboardSummary{
		UpcomingBills:   []UpcomingBill{},
		PeriodSummaries: []PeriodSummaryItem{},
	}

	for periodRows.Next() {
		var item PeriodSummaryItem
		var payDate time.Time
		if err := periodRows.Scan(&item.ID, &payDate, &item.ExpectedAmount, &item.SourceName, &item.TotalBills); err != nil {
			continue
		}
		item.PayDate = payDate.Format("2006-01-02")
		item.Remaining = item.ExpectedAmount - item.TotalBills
		summary.TotalIncome += item.ExpectedAmount
		summary.TotalBills += item.TotalBills
		summary.PeriodSummaries = append(summary.PeriodSummaries, item)
	}
	summary.Remaining = summary.TotalIncome - summary.TotalBills

	// Assignment counts
	h.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0)
		FROM bill_assignments ba
		JOIN pay_periods pp ON pp.id = ba.pay_period_id
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2
	`, from, to).Scan(&summary.PaidCount, &summary.PendingCount)

	// Upcoming bills (next 7 days).
	// A day-number BETWEEN range breaks at month end — on the 28th it searches
	// days 28-35 and never matches a bill due on the 2nd of next month. Use the
	// actual calendar days the window covers instead.
	upcomingDays := upcomingDueDays(now, 7)
	billRows, err := h.db.Query(ctx, `
		SELECT id, name, due_day, COALESCE(default_amount, 0), is_autopay
		FROM bills
		WHERE is_active = true AND due_day IS NOT NULL
		AND due_day = ANY($1)
		ORDER BY due_day
	`, upcomingDays)
	if err == nil {
		defer billRows.Close()
		for billRows.Next() {
			var b UpcomingBill
			if err := billRows.Scan(&b.ID, &b.Name, &b.DueDay, &b.Amount, &b.IsAutopay); err != nil {
				continue
			}
			summary.UpcomingBills = append(summary.UpcomingBills, b)
		}
	}

	models.WriteJSON(w, http.StatusOK, summary)
}
