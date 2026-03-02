package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
	"github.com/izz-linux/budget-mgmt/backend/internal/services"
)

type SinkingFundHandler struct {
	db DBTX
}

func NewSinkingFundHandler(db DBTX) *SinkingFundHandler {
	return &SinkingFundHandler{db: db}
}

// Plan is a dry-run that returns a SinkingFundPlan without writing to the DB.
// POST /api/v1/bills/{id}/sinking-fund/plan
func (h *SinkingFundHandler) Plan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	billID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req struct {
		TargetPeriodID int `json:"target_period_id"`
		NumPeriods     int `json:"num_periods"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.TargetPeriodID == 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_period_id is required")
		return
	}
	if req.NumPeriods <= 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "num_periods must be positive")
		return
	}

	// Load bill amount
	var billAmount float64
	err = h.db.QueryRow(ctx,
		`SELECT COALESCE(default_amount, 0) FROM bills WHERE id = $1 AND is_active = true`,
		billID,
	).Scan(&billAmount)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "bill not found")
		return
	}

	// Load target period's pay_date
	var targetPayDate time.Time
	err = h.db.QueryRow(ctx,
		`SELECT pay_date FROM pay_periods WHERE id = $1`, req.TargetPeriodID,
	).Scan(&targetPayDate)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "target period not found")
		return
	}

	// Fetch N periods preceding the target period, ordered DESC then reversed to oldest-first.
	// Sum existing assignments per period (excluding any existing sinking fund installments
	// for this bill+target pair so replanning starts fresh logically).
	rows, err := h.db.Query(ctx, `
		SELECT pp.id,
		       pp.pay_date::text,
		       COALESCE(pp.expected_amount, 0) AS income,
		       COALESCE(SUM(ba.planned_amount) FILTER (
		           WHERE ba.bill_id IS NOT NULL
		             AND NOT (ba.is_sinking_fund = true AND ba.bill_id = $3 AND ba.sinking_fund_for_period_id = $2)
		       ), 0) AS assigned
		FROM pay_periods pp
		LEFT JOIN bill_assignments ba ON ba.pay_period_id = pp.id
		WHERE pp.pay_date < $1
		GROUP BY pp.id, pp.pay_date
		ORDER BY pp.pay_date DESC
		LIMIT $4
	`, targetPayDate, req.TargetPeriodID, billID, req.NumPeriods)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var candidates []services.SinkingFundPeriod
	for rows.Next() {
		var p services.SinkingFundPeriod
		if err := rows.Scan(&p.ID, &p.PayDate, &p.Income, &p.Assigned); err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		candidates = append(candidates, p)
	}

	// Reverse DESC→ASC (oldest first)
	for i, j := 0, len(candidates)-1; i < j; i, j = i+1, j-1 {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	plan := services.PlanSinkingFund(billAmount, candidates)
	models.WriteJSON(w, http.StatusOK, plan)
}

// Apply writes sinking fund installments for a bill+target period.
// POST /api/v1/bills/{id}/sinking-fund/apply
func (h *SinkingFundHandler) Apply(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	billID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req struct {
		TargetPeriodID int `json:"target_period_id"`
		NumPeriods     int `json:"num_periods"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.TargetPeriodID == 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_period_id is required")
		return
	}
	if req.NumPeriods <= 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "num_periods must be positive")
		return
	}

	// Run the plan computation the same way Plan() does
	var billAmount float64
	err = h.db.QueryRow(ctx,
		`SELECT COALESCE(default_amount, 0) FROM bills WHERE id = $1 AND is_active = true`,
		billID,
	).Scan(&billAmount)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "bill not found")
		return
	}

	var targetPayDate time.Time
	err = h.db.QueryRow(ctx,
		`SELECT pay_date FROM pay_periods WHERE id = $1`, req.TargetPeriodID,
	).Scan(&targetPayDate)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "target period not found")
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT pp.id,
		       pp.pay_date::text,
		       COALESCE(pp.expected_amount, 0) AS income,
		       COALESCE(SUM(ba.planned_amount) FILTER (
		           WHERE ba.bill_id IS NOT NULL
		             AND NOT (ba.is_sinking_fund = true AND ba.bill_id = $3 AND ba.sinking_fund_for_period_id = $2)
		       ), 0) AS assigned
		FROM pay_periods pp
		LEFT JOIN bill_assignments ba ON ba.pay_period_id = pp.id
		WHERE pp.pay_date < $1
		GROUP BY pp.id, pp.pay_date
		ORDER BY pp.pay_date DESC
		LIMIT $4
	`, targetPayDate, req.TargetPeriodID, billID, req.NumPeriods)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var candidates []services.SinkingFundPeriod
	for rows.Next() {
		var p services.SinkingFundPeriod
		if err := rows.Scan(&p.ID, &p.PayDate, &p.Income, &p.Assigned); err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		candidates = append(candidates, p)
	}

	for i, j := 0, len(candidates)-1; i < j; i, j = i+1, j-1 {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}

	plan := services.PlanSinkingFund(billAmount, candidates)

	tx, err := h.db.Begin(ctx)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback(ctx)

	// Clear existing installments for this bill+target pair first (idempotent)
	_, err = tx.Exec(ctx, `
		DELETE FROM bill_assignments
		WHERE bill_id = $1 AND is_sinking_fund = true AND sinking_fund_for_period_id = $2
	`, billID, req.TargetPeriodID)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var created []models.BillAssignment
	for _, inst := range plan.Installments {
		if inst.Amount <= 0 {
			continue // skip zero-contribution periods
		}
		amount := inst.Amount
		var a models.BillAssignment
		err = tx.QueryRow(ctx, `
			INSERT INTO bill_assignments
				(bill_id, pay_period_id, planned_amount, status, manually_moved, is_sinking_fund, sinking_fund_for_period_id)
			VALUES ($1, $2, $3, 'pending', true, true, $4)
			ON CONFLICT (bill_id, pay_period_id) DO UPDATE SET
				planned_amount = EXCLUDED.planned_amount,
				manually_moved = true,
				is_sinking_fund = true,
				sinking_fund_for_period_id = EXCLUDED.sinking_fund_for_period_id,
				updated_at = NOW()
			RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
			          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''),
			          manually_moved, is_sinking_fund, sinking_fund_for_period_id, created_at, updated_at
		`, billID, inst.PeriodID, amount, req.TargetPeriodID).Scan(
			&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
			&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
			&a.Notes, &a.ManuallyMoved, &a.IsSinkingFund, &a.SinkingFundForPeriodID,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		created = append(created, a)
	}

	if err := tx.Commit(ctx); err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	if created == nil {
		created = []models.BillAssignment{}
	}
	models.WriteJSON(w, http.StatusCreated, created)
}

// Clear removes all sinking fund installments for a bill+target period pair.
// DELETE /api/v1/bills/{id}/sinking-fund?target_period_id=N
func (h *SinkingFundHandler) Clear(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	billID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	targetPeriodIDStr := r.URL.Query().Get("target_period_id")
	targetPeriodID, err := strconv.Atoi(targetPeriodIDStr)
	if err != nil || targetPeriodID == 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "target_period_id query param required")
		return
	}

	_, err = h.db.Exec(ctx, `
		DELETE FROM bill_assignments
		WHERE bill_id = $1 AND is_sinking_fund = true AND sinking_fund_for_period_id = $2
	`, billID, targetPeriodID)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
