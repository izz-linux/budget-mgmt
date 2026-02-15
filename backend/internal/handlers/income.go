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

	var s models.IncomeSource
	err = h.db.QueryRow(ctx, `
		UPDATE income_sources SET
			name = COALESCE($2, name),
			pay_schedule = COALESCE($3, pay_schedule),
			schedule_detail = COALESCE($4, schedule_detail),
			default_amount = COALESCE($5, default_amount),
			is_active = COALESCE($6, is_active),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, pay_schedule, schedule_detail, default_amount,
		          is_active, created_at, updated_at
	`, id, req.Name, req.PaySchedule, req.ScheduleDetail,
		req.DefaultAmount, req.IsActive,
	).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
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
