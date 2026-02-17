package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type IncomeHandler struct {
	db DBTX
}

func NewIncomeHandler(db DBTX) *IncomeHandler {
	return &IncomeHandler{db: db}
}

func (h *IncomeHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeOnly := r.URL.Query().Get("active") == "true"

	query := `
		SELECT id, name, pay_schedule, schedule_detail, default_amount,
		       is_active, created_at, updated_at
		FROM income_sources
	`
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY name"

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var sources []models.IncomeSource
	for rows.Next() {
		var s models.IncomeSource
		err := rows.Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
			&s.DefaultAmount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		sources = append(sources, s)
	}

	if sources == nil {
		sources = []models.IncomeSource{}
	}
	models.WriteJSON(w, http.StatusOK, sources)
}

func (h *IncomeHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var s models.IncomeSource
	err = h.db.QueryRow(ctx, `
		SELECT id, name, pay_schedule, schedule_detail, default_amount,
		       is_active, created_at, updated_at
		FROM income_sources WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "income source not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, s)
}

func (h *IncomeHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.CreateIncomeSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.Name == "" {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	validSchedules := map[string]bool{"weekly": true, "biweekly": true, "semimonthly": true, "one_time": true}
	if !validSchedules[req.PaySchedule] {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "pay_schedule must be weekly, biweekly, semimonthly, or one_time")
		return
	}

	var s models.IncomeSource
	err := h.db.QueryRow(ctx, `
		INSERT INTO income_sources (name, pay_schedule, schedule_detail, default_amount)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, pay_schedule, schedule_detail, default_amount,
		          is_active, created_at, updated_at
	`, req.Name, req.PaySchedule, req.ScheduleDetail, req.DefaultAmount,
	).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	models.WriteJSON(w, http.StatusCreated, s)
}

func (h *IncomeHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req models.UpdateIncomeSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	// Build dynamic update to avoid COALESCE issues with intentional NULLs
	setClauses := []string{}
	args := []interface{}{id}
	argIdx := 2

	if req.Name != nil {
		setClauses = append(setClauses, "name = $"+strconv.Itoa(argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.PaySchedule != nil {
		setClauses = append(setClauses, "pay_schedule = $"+strconv.Itoa(argIdx))
		args = append(args, *req.PaySchedule)
		argIdx++
	}
	if req.ScheduleDetail != nil {
		setClauses = append(setClauses, "schedule_detail = $"+strconv.Itoa(argIdx))
		args = append(args, req.ScheduleDetail)
		argIdx++
	}
	if req.DefaultAmount != nil {
		setClauses = append(setClauses, "default_amount = $"+strconv.Itoa(argIdx))
		args = append(args, *req.DefaultAmount)
		argIdx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, "is_active = $"+strconv.Itoa(argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "no fields to update")
		return
	}

	query := "UPDATE income_sources SET " + setClauses[0]
	for i := 1; i < len(setClauses); i++ {
		query += ", " + setClauses[i]
	}
	query += `, updated_at = NOW() WHERE id = $1
		RETURNING id, name, pay_schedule, schedule_detail, default_amount,
		          is_active, created_at, updated_at`

	var s models.IncomeSource
	err = h.db.QueryRow(ctx, query, args...).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "income source not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, s)
}

func (h *IncomeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	// Cascade: delete bill_assignments tied to this source's pay periods
	_, err = h.db.Exec(ctx, `
		DELETE FROM bill_assignments
		WHERE pay_period_id IN (SELECT id FROM pay_periods WHERE income_source_id = $1)
	`, id)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Cascade: delete pay periods for this source
	_, err = h.db.Exec(ctx, `DELETE FROM pay_periods WHERE income_source_id = $1`, id)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Soft-delete the income source
	tag, err := h.db.Exec(ctx, `UPDATE income_sources SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "income source not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
