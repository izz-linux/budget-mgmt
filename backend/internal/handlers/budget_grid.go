package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type GridHandler struct {
	db DBTX
}

func NewGridHandler(db DBTX) *GridHandler {
	return &GridHandler{db: db}
}

type BudgetGridResponse struct {
	Bills       []models.Bill                `json:"bills"`
	Periods     []models.PayPeriod           `json:"periods"`
	Assignments map[string]models.BillAssignment `json:"assignments"` // key: "billId-periodId"
}

func (h *GridHandler) GetGrid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		now := time.Now()
		from = now.Format("2006-01-02")
		to = now.AddDate(0, 3, 0).Format("2006-01-02")
	}

	// Fetch bills
	billRows, err := h.db.Query(ctx, `
		SELECT b.id, b.name, b.default_amount, b.due_day, b.recurrence,
		       b.recurrence_detail, b.is_autopay, COALESCE(b.category, ''), COALESCE(b.notes, ''),
		       b.is_active, b.sort_order, b.created_at, b.updated_at,
		       cc.id, cc.card_label, cc.statement_day, cc.due_day, cc.issuer
		FROM bills b
		LEFT JOIN credit_cards cc ON cc.bill_id = b.id
		WHERE b.is_active = true
		ORDER BY b.sort_order, b.id
	`)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer billRows.Close()

	var bills []models.Bill
	for billRows.Next() {
		var b models.Bill
		var ccID *int
		var ccLabel, ccIssuer *string
		var ccStatementDay, ccDueDay *int

		err := billRows.Scan(
			&b.ID, &b.Name, &b.DefaultAmount, &b.DueDay, &b.Recurrence,
			&b.RecurrenceDetail, &b.IsAutopay, &b.Category, &b.Notes,
			&b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt,
			&ccID, &ccLabel, &ccStatementDay, &ccDueDay, &ccIssuer,
		)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		if ccID != nil {
			b.CreditCard = &models.CreditCard{
				ID:           *ccID,
				BillID:       b.ID,
				StatementDay: *ccStatementDay,
				DueDay:       *ccDueDay,
			}
			if ccLabel != nil {
				b.CreditCard.CardLabel = *ccLabel
			}
			if ccIssuer != nil {
				b.CreditCard.Issuer = *ccIssuer
			}
		}
		bills = append(bills, b)
	}

	// Fetch periods with totals
	periodRows, err := h.db.Query(ctx, `
		SELECT pp.id, pp.income_source_id, pp.pay_date, pp.expected_amount,
		       pp.actual_amount, COALESCE(pp.notes, ''), pp.created_at, inc.name,
		       COALESCE(SUM(ba.planned_amount), 0) as total_bills
		FROM pay_periods pp
		JOIN income_sources inc ON inc.id = pp.income_source_id
		LEFT JOIN bill_assignments ba ON ba.pay_period_id = pp.id
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2
		GROUP BY pp.id, pp.income_source_id, pp.pay_date, pp.expected_amount,
		         pp.actual_amount, pp.notes, pp.created_at, inc.name
		ORDER BY pp.pay_date
	`, from, to)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer periodRows.Close()

	var periods []models.PayPeriod
	periodIDs := []int{}
	for periodRows.Next() {
		var p models.PayPeriod
		err := periodRows.Scan(&p.ID, &p.IncomeSourceID, &p.PayDate, &p.ExpectedAmount,
			&p.ActualAmount, &p.Notes, &p.CreatedAt, &p.SourceName, &p.TotalBills)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		if p.ExpectedAmount != nil {
			p.Remaining = *p.ExpectedAmount - p.TotalBills
		}
		periods = append(periods, p)
		periodIDs = append(periodIDs, p.ID)
	}

	// Fetch assignments for these periods
	assignments := make(map[string]models.BillAssignment)
	if len(periodIDs) > 0 {
		assignRows, err := h.db.Query(ctx, `
			SELECT ba.id, ba.bill_id, ba.pay_period_id, ba.planned_amount,
			       ba.forecast_amount, ba.actual_amount, ba.status, ba.deferred_to_id,
			       ba.is_extra, COALESCE(ba.extra_name, ''), COALESCE(ba.notes, ''), ba.created_at, ba.updated_at,
			       b.name
			FROM bill_assignments ba
			JOIN bills b ON b.id = ba.bill_id
			WHERE ba.pay_period_id = ANY($1)
			ORDER BY b.sort_order, b.id
		`, periodIDs)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		defer assignRows.Close()

		for assignRows.Next() {
			var a models.BillAssignment
			err := assignRows.Scan(&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount,
				&a.ForecastAmount, &a.ActualAmount, &a.Status, &a.DeferredToID,
				&a.IsExtra, &a.ExtraName, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
				&a.BillName)
			if err != nil {
				models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
				return
			}
			key := strconv.Itoa(a.BillID) + "-" + strconv.Itoa(a.PayPeriodID)
			assignments[key] = a
		}
	}

	if bills == nil {
		bills = []models.Bill{}
	}
	if periods == nil {
		periods = []models.PayPeriod{}
	}

	models.WriteJSON(w, http.StatusOK, BudgetGridResponse{
		Bills:       bills,
		Periods:     periods,
		Assignments: assignments,
	})
}
