package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type IncomeHandler struct {
	db DBTX
}

func NewIncomeHandler(db DBTX) *IncomeHandler {
	return &IncomeHandler{db: db}
}

var validPaySchedules = map[string]bool{
	"weekly": true, "biweekly": true, "semimonthly": true, "one_time": true,
}

// validateSchedule checks that paySchedule is one we support and that detail is
// a well-formed payload for it. The period generator trusts these values, so a
// bad weekday or day-of-month stored here surfaces later as a generation error
// (or, before the generator was hardened, as a hung request).
func validateSchedule(paySchedule string, detail json.RawMessage) error {
	if !validPaySchedules[paySchedule] {
		return errors.New("pay_schedule must be weekly, biweekly, semimonthly, or one_time")
	}
	if len(detail) == 0 || string(detail) == "null" {
		return errors.New("schedule_detail is required")
	}

	switch paySchedule {
	case "weekly":
		var s models.WeeklySchedule
		if err := json.Unmarshal(detail, &s); err != nil {
			return errors.New("schedule_detail must be an object with a weekday field")
		}
		if s.Weekday < 0 || s.Weekday > 6 {
			return errors.New("schedule_detail.weekday must be 0-6 (0=Sunday)")
		}

	case "biweekly":
		var s models.BiweeklySchedule
		if err := json.Unmarshal(detail, &s); err != nil {
			return errors.New("schedule_detail must be an object with an anchor_date field")
		}
		if s.AnchorDate == "" {
			return errors.New("schedule_detail.anchor_date is required for biweekly")
		}
		if _, err := time.Parse("2006-01-02", s.AnchorDate); err != nil {
			return errors.New("schedule_detail.anchor_date must be in YYYY-MM-DD format")
		}

	case "semimonthly":
		var s models.SemiMonthlySchedule
		if err := json.Unmarshal(detail, &s); err != nil {
			return errors.New("schedule_detail must be an object with a days array")
		}
		if len(s.Days) != 2 {
			return errors.New("schedule_detail.days must contain exactly 2 days")
		}
		for _, d := range s.Days {
			if d < 1 || d > 31 {
				return errors.New("schedule_detail.days entries must be 1-31")
			}
		}

	case "one_time":
		var s models.OneTimeSchedule
		if err := json.Unmarshal(detail, &s); err != nil {
			return errors.New("schedule_detail must be an object with a date field")
		}
		if s.Date == "" {
			return errors.New("schedule_detail.date is required for one_time")
		}
		if _, err := time.Parse("2006-01-02", s.Date); err != nil {
			return errors.New("schedule_detail.date must be in YYYY-MM-DD format")
		}
	}

	return nil
}

func (h *IncomeHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeOnly := r.URL.Query().Get("active") == "true"

	query := `
		SELECT id, name, pay_schedule, schedule_detail, default_amount,
		       is_active, effective_from, created_at, updated_at
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
			&s.DefaultAmount, &s.IsActive, &s.EffectiveFrom, &s.CreatedAt, &s.UpdatedAt)
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
		       is_active, effective_from, created_at, updated_at
		FROM income_sources WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.EffectiveFrom, &s.CreatedAt, &s.UpdatedAt)
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
	if err := validateSchedule(req.PaySchedule, req.ScheduleDetail); err != nil {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Parse effective_from if provided
	var effectiveFrom *time.Time
	if req.EffectiveFrom != nil && *req.EffectiveFrom != "" {
		parsed, err := time.ParseInLocation("2006-01-02", *req.EffectiveFrom, time.Local)
		if err != nil {
			models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "effective_from must be in YYYY-MM-DD format")
			return
		}
		effectiveFrom = &parsed
	}

	var s models.IncomeSource
	err := h.db.QueryRow(ctx, `
		INSERT INTO income_sources (name, pay_schedule, schedule_detail, default_amount, effective_from)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, pay_schedule, schedule_detail, default_amount,
		          is_active, effective_from, created_at, updated_at
	`, req.Name, req.PaySchedule, req.ScheduleDetail, req.DefaultAmount, effectiveFrom,
	).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.EffectiveFrom, &s.CreatedAt, &s.UpdatedAt)
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

	// If either half of the schedule is changing, validate the combination that
	// will actually be stored — the unchanged half has to come from the DB.
	if req.PaySchedule != nil || len(req.ScheduleDetail) > 0 {
		var curSchedule string
		var curDetail json.RawMessage
		err = h.db.QueryRow(ctx,
			`SELECT pay_schedule, schedule_detail FROM income_sources WHERE id = $1`, id,
		).Scan(&curSchedule, &curDetail)
		if err != nil {
			models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "income source not found")
			return
		}

		schedule := curSchedule
		if req.PaySchedule != nil {
			schedule = *req.PaySchedule
		}
		detail := curDetail
		if len(req.ScheduleDetail) > 0 {
			detail = req.ScheduleDetail
		}
		if err := validateSchedule(schedule, detail); err != nil {
			models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
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
	if req.EffectiveFrom != nil {
		if *req.EffectiveFrom == "" {
			// Allow clearing effective_from by passing empty string
			setClauses = append(setClauses, "effective_from = NULL")
		} else {
			parsed, err := time.ParseInLocation("2006-01-02", *req.EffectiveFrom, time.Local)
			if err != nil {
				models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "effective_from must be in YYYY-MM-DD format")
				return
			}
			setClauses = append(setClauses, "effective_from = $"+strconv.Itoa(argIdx))
			args = append(args, parsed)
			argIdx++
		}
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
		          is_active, effective_from, created_at, updated_at`

	var s models.IncomeSource
	err = h.db.QueryRow(ctx, query, args...).Scan(&s.ID, &s.Name, &s.PaySchedule, &s.ScheduleDetail,
		&s.DefaultAmount, &s.IsActive, &s.EffectiveFrom, &s.CreatedAt, &s.UpdatedAt)
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
