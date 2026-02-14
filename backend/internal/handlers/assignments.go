package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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
