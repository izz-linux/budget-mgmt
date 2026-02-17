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
		SELECT id, name, default_amount, due_day, recurrence, recurrence_detail
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
		ID               int
		DefaultAmount    *float64
		DueDay           int
		Recurrence       string
		RecurrenceDetail json.RawMessage
	}
	var bills []billInfo
	for billRows.Next() {
		var b billInfo
		var name string
		if err := billRows.Scan(&b.ID, &name, &b.DefaultAmount, &b.DueDay, &b.Recurrence, &b.RecurrenceDetail); err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		bills = append(bills, b)
	}

	if len(bills) == 0 {
		models.WriteJSON(w, http.StatusOK, []models.BillAssignment{})
		return
	}

	// Get all periods in range (only from active income sources)
	periodRows, err := h.db.Query(ctx, `
		SELECT pp.id, pp.pay_date FROM pay_periods pp
		JOIN income_sources inc ON inc.id = pp.income_source_id
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2 AND inc.is_active = true
		ORDER BY pp.pay_date
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

	// Pre-fetch existing assignments in range so we know which bill+period combos
	// already exist (user may have moved or placed bills manually).
	type billPeriod struct {
		BillID   int
		PeriodID int
	}
	existingPairs := make(map[billPeriod]bool)

	// Also track bill+month for monthly bills to avoid duplicates
	type billMonth struct {
		BillID int
		Year   int
		Month  time.Month
	}
	existingBillMonths := make(map[billMonth]bool)

	existRows, err := h.db.Query(ctx, `
		SELECT ba.bill_id, ba.pay_period_id, pp.pay_date
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
		var billID, periodID int
		var payDate time.Time
		if err := existRows.Scan(&billID, &periodID, &payDate); err != nil {
			continue
		}
		existingPairs[billPeriod{billID, periodID}] = true
		existingBillMonths[billMonth{billID, payDate.Year(), payDate.Month()}] = true
	}

	// Helper: find the best period for a due date (last period on or before it)
	findBestPeriod := func(dueDate time.Time) int {
		best := -1
		for i := len(periods) - 1; i >= 0; i-- {
			if !periods[i].PayDate.After(dueDate) {
				best = i
				break
			}
		}
		if best < 0 && len(periods) > 0 {
			// No period before due date; use earliest period in or after due date's month
			year, month := dueDate.Year(), dueDate.Month()
			idx := sort.Search(len(periods), func(i int) bool {
				return periods[i].PayDate.Year() > year ||
					(periods[i].PayDate.Year() == year && periods[i].PayDate.Month() >= month)
			})
			if idx < len(periods) {
				best = idx
			}
		}
		return best
	}

	// Helper: insert a single assignment
	insertAssignment := func(billID int, periodID int, amount *float64) *models.BillAssignment {
		var a models.BillAssignment
		err := h.db.QueryRow(ctx, `
			INSERT INTO bill_assignments (bill_id, pay_period_id, planned_amount, status)
			VALUES ($1, $2, $3, 'pending')
			ON CONFLICT (bill_id, pay_period_id) DO NOTHING
			RETURNING id, bill_id, pay_period_id, planned_amount, forecast_amount, actual_amount,
			          status, deferred_to_id, is_extra, COALESCE(extra_name, ''), COALESCE(notes, ''), created_at, updated_at
		`, billID, periodID, amount).Scan(
			&a.ID, &a.BillID, &a.PayPeriodID, &a.PlannedAmount, &a.ForecastAmount,
			&a.ActualAmount, &a.Status, &a.DeferredToID, &a.IsExtra, &a.ExtraName,
			&a.Notes, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil // ON CONFLICT DO NOTHING or other error
		}
		return &a
	}

	var created []models.BillAssignment

	// Process biweekly bills: compute due dates every 14 days from anchor
	assignBiweekly := func(bill billInfo) bool {
		var detail struct {
			AnchorDate string `json:"anchor_date"`
		}
		if len(bill.RecurrenceDetail) > 0 {
			json.Unmarshal(bill.RecurrenceDetail, &detail)
		}
		if detail.AnchorDate == "" {
			return false // no anchor, fall back to monthly
		}
		anchor, parseErr := time.Parse("2006-01-02", detail.AnchorDate)
		if parseErr != nil {
			return false
		}

		// Calculate start of biweekly cycle relative to range
		daysDiff := fromDate.Sub(anchor).Hours() / 24
		cycleOffset := int(daysDiff) / 14
		if daysDiff < 0 {
			cycleOffset--
		}
		cur := anchor.AddDate(0, 0, cycleOffset*14)
		for cur.Before(fromDate) {
			cur = cur.AddDate(0, 0, 14)
		}

		// Aggregate amounts per period (multiple occurrences may map to same period)
		periodAmounts := make(map[int]float64)

		for !cur.After(toDate) {
			idx := findBestPeriod(cur)
			if idx >= 0 {
				pid := periods[idx].ID
				if !existingPairs[billPeriod{bill.ID, pid}] {
					amt := 0.0
					if bill.DefaultAmount != nil {
						amt = *bill.DefaultAmount
					}
					periodAmounts[pid] += amt
				}
			}
			cur = cur.AddDate(0, 0, 14)
		}

		for pid, amount := range periodAmounts {
			a := amount
			if result := insertAssignment(bill.ID, pid, &a); result != nil {
				created = append(created, *result)
			}
		}
		return true
	}

	// Process monthly bills: one assignment per month
	assignMonthly := func(bill billInfo) {
		current := time.Date(fromDate.Year(), fromDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		endMonth := time.Date(toDate.Year(), toDate.Month(), 1, 0, 0, 0, 0, time.UTC)

		for !current.After(endMonth) {
			year, month := current.Year(), current.Month()

			// Skip if this bill already has an assignment in this month
			if existingBillMonths[billMonth{bill.ID, year, month}] {
				current = current.AddDate(0, 1, 0)
				continue
			}

			dueDate := time.Date(year, month, bill.DueDay, 0, 0, 0, 0, time.UTC)
			lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
			if bill.DueDay > lastDay {
				dueDate = time.Date(year, month, lastDay, 0, 0, 0, 0, time.UTC)
			}

			if dueDate.Before(fromDate) || dueDate.After(toDate) {
				current = current.AddDate(0, 1, 0)
				continue
			}

			idx := findBestPeriod(dueDate)
			if idx >= 0 {
				if a := insertAssignment(bill.ID, periods[idx].ID, bill.DefaultAmount); a != nil {
					created = append(created, *a)
				}
			}

			current = current.AddDate(0, 1, 0)
		}
	}

	for _, bill := range bills {
		if bill.Recurrence == "biweekly" {
			if assignBiweekly(bill) {
				continue
			}
		}
		// Monthly or fallback for biweekly without anchor
		assignMonthly(bill)
	}

	if created == nil {
		created = []models.BillAssignment{}
	}
	models.WriteJSON(w, http.StatusCreated, created)
}
