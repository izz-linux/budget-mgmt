package handlers

import (
	"encoding/json"
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

	// Fetch current assignments
	assignRows, err := h.db.Query(ctx, `
		SELECT bill_id, pay_period_id FROM bill_assignments
		WHERE pay_period_id IN (SELECT id FROM pay_periods WHERE pay_date >= $1 AND pay_date <= $2)
	`, req.From, req.To)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer assignRows.Close()

	var currentAssignments []services.OptAssignment
	for assignRows.Next() {
		var a services.OptAssignment
		if err := assignRows.Scan(&a.BillID, &a.PeriodID); err != nil {
			continue
		}
		currentAssignments = append(currentAssignments, a)
	}

	result := h.optimizer.Optimize(bills, periods, currentAssignments)
	models.WriteJSON(w, http.StatusOK, result)
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
