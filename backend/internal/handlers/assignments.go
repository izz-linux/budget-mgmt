package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type AssignmentHandler struct {
	db DBTX
}

func NewAssignmentHandler(db DBTX) *AssignmentHandler {
	return &AssignmentHandler{db: db}
}

func (h *AssignmentHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT ba.id, ba.bill_id, ba.pay_period_id, ba.planned_amount,
		       ba.forecast_amount, ba.actual_amount, ba.status, ba.deferred_to_id,
		       ba.is_extra, COALESCE(ba.extra_name, ''), COALESCE(ba.notes, ''), ba.created_at, ba.updated_at,
		       b.name
		FROM bill_assignments ba
		JOIN bills b ON b.id = ba.bill_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if periodID := r.URL.Query().Get("period_id"); periodID != "" {
		query += " AND ba.pay_period_id = $" + strconv.Itoa(argIdx)
		id, _ := strconv.Atoi(periodID)
		args = append(args, id)
		argIdx++
	}
	if billID := r.URL.Query().Get("bill_id"); billID != "" {
		query += " AND ba.bill_id = $" + strconv.Itoa(argIdx)
		id, _ := strconv.Atoi(billID)
		args = append(args, id)
		argIdx++
	}
	if status := r.URL.Query().Get("status"); status != "" {
		query += " AND ba.status = $" + strconv.Itoa(argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY b.sort_order, b.id"

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var assignments []models.BillAssignment
	for rows.Next() {
		var a models.BillAssignment
		err := rows.Scan(&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount,
			&a.ForecastAmount, &a.ActualAmount, &a.Status, &a.DeferredToID,
			&a.IsExtra, &a.ExtraName, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
			&a.BillName)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		assignments = append(assignments, a)
	}

	if assignments == nil {
		assignments = []models.BillAssignment{}
	}
	models.WriteJSON(w, http.StatusOK, assignments)
}

func (h *AssignmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.Status == "" {
		req.Status = "pending"
	}

	var a models.BillAssignment
	err := h.db.QueryRow(ctx, `
		INSERT INTO bill_assignments (bill_id, pay_period_id, planned_amount, forecast_amount,
		                              actual_amount, status, is_extra, extra_name, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
		          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''), created_at, updated_at
	`, req.BillID, req.PayPeriodID, req.PlannedAmount, req.ForecastAmount,
		req.ActualAmount, req.Status, req.IsExtra, req.ExtraName, req.Notes,
	).Scan(&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
		&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
		&a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	models.WriteJSON(w, http.StatusCreated, a)
}

func (h *AssignmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req models.UpdateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	var a models.BillAssignment
	err = h.db.QueryRow(ctx, `
		UPDATE bill_assignments SET
			planned_amount = COALESCE($2, planned_amount),
			forecast_amount = COALESCE($3, forecast_amount),
			actual_amount = COALESCE($4, actual_amount),
			status = COALESCE($5, status),
			deferred_to_id = COALESCE($6, deferred_to_id),
			notes = COALESCE($7, notes),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
		          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''), created_at, updated_at
	`, id, req.PlannedAmount, req.ForecastAmount, req.ActualAmount,
		req.Status, req.DeferredToID, req.Notes,
	).Scan(&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
		&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
		&a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "assignment not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, a)
}

func (h *AssignmentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "paid": true, "deferred": true, "uncertain": true, "skipped": true,
	}
	if !validStatuses[req.Status] {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid status")
		return
	}

	var a models.BillAssignment
	err = h.db.QueryRow(ctx, `
		UPDATE bill_assignments SET
			status = $2,
			deferred_to_id = $3,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
		          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''), created_at, updated_at
	`, id, req.Status, req.DeferredToID,
	).Scan(&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
		&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
		&a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "assignment not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, a)
}

func (h *AssignmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	tag, err := h.db.Exec(ctx, `DELETE FROM bill_assignments WHERE id = $1`, id)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "assignment not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AssignmentHandler) AutoAssign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	fromDate, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid from date")
		return
	}
	toDate, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid to date")
		return
	}

	// Get active bills with due_day set
	billRows, err := h.db.Query(ctx, `
		SELECT id, name, default_amount, due_day, recurrence
		FROM bills
		WHERE is_active = true AND due_day IS NOT NULL
		ORDER BY id
	`)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer billRows.Close()

	type billInfo struct {
		ID            int
		DefaultAmount *float64
		DueDay        int
		Recurrence    string
	}
	var bills []billInfo
	for billRows.Next() {
		var b billInfo
		var name string
		if err := billRows.Scan(&b.ID, &name, &b.DefaultAmount, &b.DueDay, &b.Recurrence); err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		bills = append(bills, b)
	}

	if len(bills) == 0 {
		models.WriteJSON(w, http.StatusOK, []models.BillAssignment{})
		return
	}

	// Get all periods in range
	periodRows, err := h.db.Query(ctx, `
		SELECT id, pay_date FROM pay_periods
		WHERE pay_date >= $1 AND pay_date <= $2
		ORDER BY pay_date
	`, req.From, req.To)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer periodRows.Close()

	type periodInfo struct {
		ID      int
		PayDate time.Time
	}
	var periods []periodInfo
	for periodRows.Next() {
		var p periodInfo
		if err := periodRows.Scan(&p.ID, &p.PayDate); err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		periods = append(periods, p)
	}

	if len(periods) == 0 {
		models.WriteJSON(w, http.StatusOK, []models.BillAssignment{})
		return
	}

	// Pre-fetch existing assignments in range so we know which bill+month combos
	// already have an assignment (user may have moved bills to different periods).
	type billMonth struct {
		BillID int
		Year   int
		Month  time.Month
	}
	existingBillMonths := make(map[billMonth]bool)

	existRows, err := h.db.Query(ctx, `
		SELECT ba.bill_id, pp.pay_date
		FROM bill_assignments ba
		JOIN pay_periods pp ON pp.id = ba.pay_period_id
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2
	`, req.From, req.To)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer existRows.Close()

	for existRows.Next() {
		var billID int
		var payDate time.Time
		if err := existRows.Scan(&billID, &payDate); err != nil {
			continue
		}
		existingBillMonths[billMonth{billID, payDate.Year(), payDate.Month()}] = true
	}

	// For each bill, for each month in range, find the best period and create assignment
	var created []models.BillAssignment
	current := time.Date(fromDate.Year(), fromDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(toDate.Year(), toDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !current.After(endMonth) {
		year, month := current.Year(), current.Month()

		for _, bill := range bills {
			// Skip if this bill already has an assignment for this month
			// (user may have manually placed or moved it)
			if existingBillMonths[billMonth{bill.ID, year, month}] {
				continue
			}

			// Calculate due date for this month
			dueDate := time.Date(year, month, bill.DueDay, 0, 0, 0, 0, time.UTC)
			// Clamp to last day of month
			lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
			if bill.DueDay > lastDay {
				dueDate = time.Date(year, month, lastDay, 0, 0, 0, 0, time.UTC)
			}

			if dueDate.Before(fromDate) || dueDate.After(toDate) {
				continue
			}

			// Find the last period on or before the due date
			bestPeriod := -1
			for i := len(periods) - 1; i >= 0; i-- {
				if !periods[i].PayDate.After(dueDate) {
					bestPeriod = i
					break
				}
			}
			// If no period before due date, use the first period in range
			if bestPeriod < 0 {
				// Find the first period in this month or after
				idx := sort.Search(len(periods), func(i int) bool {
					return periods[i].PayDate.Year() > year ||
						(periods[i].PayDate.Year() == year && periods[i].PayDate.Month() >= month)
				})
				if idx < len(periods) {
					bestPeriod = idx
				}
			}
			if bestPeriod < 0 {
				continue
			}

			periodID := periods[bestPeriod].ID

			// Insert assignment if it doesn't exist
			var a models.BillAssignment
			err := h.db.QueryRow(ctx, `
				INSERT INTO bill_assignments (bill_id, pay_period_id, planned_amount, status)
				VALUES ($1, $2, $3, 'pending')
				ON CONFLICT (bill_id, pay_period_id) DO NOTHING
				RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
				          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''), created_at, updated_at
			`, bill.ID, periodID, bill.DefaultAmount).Scan(
				&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
				&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
				&a.Notes, &a.CreatedAt, &a.UpdatedAt,
			)
			if err != nil {
				// ON CONFLICT DO NOTHING returns no rows - that's fine, skip
				continue
			}
			created = append(created, a)
		}

		current = current.AddDate(0, 1, 0)
	}

	if created == nil {
		created = []models.BillAssignment{}
	}
	models.WriteJSON(w, http.StatusCreated, created)
}
