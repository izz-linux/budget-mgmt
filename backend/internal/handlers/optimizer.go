package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/izz-linux/budget-mgmt/backend/internal/models"
	"github.com/izz-linux/budget-mgmt/backend/internal/services"
)

type OptimizerHandler struct {
	db              DBTX
	optimizer       *services.Optimizer
	surplusDetector *services.SurplusDetector
}

func NewOptimizerHandler(db DBTX) *OptimizerHandler {
	return &OptimizerHandler{
		db:              db,
		optimizer:       services.NewOptimizer(),
		surplusDetector: services.NewSurplusDetector(),
	}
}

func (h *OptimizerHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Strategy string `json:"strategy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	// Fetch bills
	billRows, err := h.db.Query(ctx, `
		SELECT id, name, due_day, COALESCE(default_amount, 0)
		FROM bills WHERE is_active = true AND due_day IS NOT NULL
	`)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer billRows.Close()

	var bills []services.OptBill
	for billRows.Next() {
		var b services.OptBill
		if err := billRows.Scan(&b.ID, &b.Name, &b.DueDay, &b.Amount); err != nil {
			continue
		}
		bills = append(bills, b)
	}

	// Fetch periods
	periodRows, err := h.db.Query(ctx, `
		SELECT pp.id, pp.pay_date, EXTRACT(DAY FROM pp.pay_date)::int,
		       COALESCE(pp.expected_amount, 0)
		FROM pay_periods pp
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2
		ORDER BY pp.pay_date
	`, req.From, req.To)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer periodRows.Close()

	var periods []services.OptPeriod
	for periodRows.Next() {
		var p services.OptPeriod
		var payDate time.Time
		if err := periodRows.Scan(&p.ID, &payDate, &p.PayDay, &p.Income); err != nil {
			continue
		}
		p.PayDate = payDate.Format("2006-01-02")
		periods = append(periods, p)
	}

	// Fetch current assignments (include assignment ID for apply)
	assignRows, err := h.db.Query(ctx, `
		SELECT ba.id, ba.bill_id, ba.pay_period_id FROM bill_assignments ba
		WHERE ba.pay_period_id IN (SELECT id FROM pay_periods WHERE pay_date >= $1 AND pay_date <= $2)
	`, req.From, req.To)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer assignRows.Close()

	var currentAssignments []services.OptAssignment
	for assignRows.Next() {
		var a services.OptAssignment
		if err := assignRows.Scan(&a.AssignmentID, &a.BillID, &a.PeriodID); err != nil {
			continue
		}
		currentAssignments = append(currentAssignments, a)
	}

	result := h.optimizer.Optimize(bills, periods, currentAssignments)
	models.WriteJSON(w, http.StatusOK, result)
}

// Apply executes selected optimizer suggestions by moving assignments to new periods.
// Each move deletes the old assignment and creates a new one in the target period,
// marked as manually_moved since the optimizer is an explicit user action.
func (h *OptimizerHandler) Apply(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Moves []struct {
			AssignmentID int `json:"assignment_id"`
			ToPeriodID   int `json:"to_period_id"`
		} `json:"moves"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if len(req.Moves) == 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "no moves specified")
		return
	}

	// A batch is all-or-nothing: each move is a DELETE followed by an INSERT, so
	// a failure partway through would otherwise leave assignments destroyed with
	// no replacement.
	tx, err := h.db.Begin(ctx)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback(ctx)

	// Reject unknown target periods up front, before anything is deleted.
	for _, move := range req.Moves {
		var exists int
		err := tx.QueryRow(ctx, `SELECT id FROM pay_periods WHERE id = $1`, move.ToPeriodID).Scan(&exists)
		if err != nil {
			models.WriteError(w, http.StatusNotFound, "NOT_FOUND",
				fmt.Sprintf("pay period %d not found", move.ToPeriodID))
			return
		}
	}

	// The optimizer can chain moves for one bill (A->B then B->C). Each move
	// replaces the row, so the second move's assignment_id refers to a row that
	// no longer exists — track where each original assignment currently lives.
	liveID := make(map[int]int, len(req.Moves))

	var applied []models.BillAssignment

	for _, move := range req.Moves {
		currentID := move.AssignmentID
		if mapped, ok := liveID[move.AssignmentID]; ok {
			currentID = mapped
		}

		// Look up the existing assignment
		var billID int
		var plannedAmount *float64
		err := tx.QueryRow(ctx, `
			SELECT bill_id, planned_amount FROM bill_assignments WHERE id = $1
		`, currentID).Scan(&billID, &plannedAmount)
		if err != nil {
			models.WriteError(w, http.StatusNotFound, "NOT_FOUND",
				fmt.Sprintf("assignment %d not found", move.AssignmentID))
			return
		}

		// Delete the old assignment
		_, err = tx.Exec(ctx, `DELETE FROM bill_assignments WHERE id = $1`, currentID)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		// Create new assignment in the target period, marked as manually_moved
		var a models.BillAssignment
		err = tx.QueryRow(ctx, `
			INSERT INTO bill_assignments (bill_id, pay_period_id, planned_amount, status, manually_moved)
			VALUES ($1, $2, $3, 'pending', true)
			ON CONFLICT (bill_id, pay_period_id) DO UPDATE SET
				planned_amount = EXCLUDED.planned_amount,
				manually_moved = true,
				updated_at = NOW()
			RETURNING `+assignmentReturnCols+`
		`, billID, move.ToPeriodID, plannedAmount).Scan(
			&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
			&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
			&a.Notes, &a.ManuallyMoved, &a.IsSinkingFund, &a.SinkingFundForPeriodID,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		// Remember the new row id under the original id so a later chained move
		// that still references the original resolves to the live row.
		liveID[move.AssignmentID] = a.ID
		if currentID != move.AssignmentID {
			liveID[currentID] = a.ID
		}

		applied = append(applied, a)
	}

	if err := tx.Commit(ctx); err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	models.WriteJSON(w, http.StatusOK, applied)
}

func (h *OptimizerHandler) Surplus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		year := time.Now().Year()
		fromStr = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		toStr = time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}

	from, _ := time.Parse("2006-01-02", fromStr)
	to, _ := time.Parse("2006-01-02", toStr)

	// Fetch income sources
	rows, err := h.db.Query(ctx, `
		SELECT id, name, pay_schedule, schedule_detail, default_amount, is_active, created_at, updated_at
		FROM income_sources WHERE is_active = true
	`)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var sources []models.IncomeSource
	for rows.Next() {
		var s models.IncomeSource
		if err := rows.Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
			&s.DefaultAmount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		sources = append(sources, s)
	}

	result, err := h.surplusDetector.Detect(sources, from, to)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DETECTION_ERROR", err.Error())
		return
	}

	models.WriteJSON(w, http.StatusOK, result)
}
