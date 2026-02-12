package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
	"github.com/izz-linux/budget-mgmt/backend/internal/services"
)

type PeriodHandler struct {
	db        *pgxpool.Pool
	generator *services.PeriodGenerator
}

func NewPeriodHandler(db *pgxpool.Pool) *PeriodHandler {
	return &PeriodHandler{
		db:        db,
		generator: services.NewPeriodGenerator(),
	}
}

func (h *PeriodHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		// Default: show 3 months from today
		now := time.Now()
		from = now.Format("2006-01-02")
		to = now.AddDate(0, 3, 0).Format("2006-01-02")
	}

	rows, err := h.db.Query(ctx, `
		SELECT pp.id, pp.income_source_id, pp.pay_date, pp.expected_amount,
		       pp.actual_amount, pp.notes, pp.created_at, inc.name,
		       COALESCE(SUM(ba.planned_amount), 0) as total_bills
		FROM pay_periods pp
		JOIN income_sources inc ON inc.id = pp.income_source_id
		LEFT JOIN bill_assignments ba ON ba.pay_period_id = pp.id
		WHERE pp.pay_date >= $1 AND pp.pay_date <= $2
		GROUP BY pp.id, inc.name
		ORDER BY pp.pay_date
	`, from, to)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var periods []models.PayPeriod
	for rows.Next() {
		var p models.PayPeriod
		err := rows.Scan(&p.ID, &p.IncomeSourceID, &p.PayDate, &p.ExpectedAmount,
			&p.ActualAmount, &p.Notes, &p.CreatedAt, &p.SourceName, &p.TotalBills)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		if p.ExpectedAmount != nil {
			p.Remaining = *p.ExpectedAmount - p.TotalBills
		}
		periods = append(periods, p)
	}

	if periods == nil {
		periods = []models.PayPeriod{}
	}
	models.WriteJSON(w, http.StatusOK, periods)
}

func (h *PeriodHandler) Generate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.GeneratePeriodsRequest
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

	// Get income sources
	query := `SELECT id, name, pay_schedule, schedule_detail, default_amount, is_active, created_at, updated_at
	          FROM income_sources WHERE is_active = true`
	args := []interface{}{}
	if len(req.SourceIDs) > 0 {
		query += " AND id = ANY($1)"
		args = append(args, req.SourceIDs)
	}

	rows, err := h.db.Query(ctx, query, args...)
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
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		sources = append(sources, s)
	}

	// Generate and insert periods
	var created []models.PayPeriod
	for _, source := range sources {
		dates, err := h.generator.Generate(source, fromDate, toDate)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "GENERATION_ERROR", err.Error())
			return
		}

		for _, date := range dates {
			var p models.PayPeriod
			err := h.db.QueryRow(ctx, `
				INSERT INTO pay_periods (income_source_id, pay_date, expected_amount)
				VALUES ($1, $2, $3)
				ON CONFLICT (income_source_id, pay_date) DO UPDATE SET
					expected_amount = COALESCE(EXCLUDED.expected_amount, pay_periods.expected_amount)
				RETURNING id, income_source_id, pay_date, expected_amount, actual_amount, notes, created_at
			`, source.ID, date, source.DefaultAmount).Scan(
				&p.ID, &p.IncomeSourceID, &p.PayDate, &p.ExpectedAmount,
				&p.ActualAmount, &p.Notes, &p.CreatedAt,
			)
			if err != nil {
				models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
				return
			}
			p.SourceName = source.Name
			created = append(created, p)
		}
	}

	if created == nil {
		created = []models.PayPeriod{}
	}
	models.WriteJSON(w, http.StatusCreated, created)
}

func (h *PeriodHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var body struct {
		ExpectedAmount *float64 `json:"expected_amount"`
		ActualAmount   *float64 `json:"actual_amount"`
		Notes          *string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	var p models.PayPeriod
	err = h.db.QueryRow(ctx, `
		UPDATE pay_periods SET
			expected_amount = COALESCE($2, expected_amount),
			actual_amount = COALESCE($3, actual_amount),
			notes = COALESCE($4, notes)
		WHERE id = $1
		RETURNING id, income_source_id, pay_date, expected_amount, actual_amount, notes, created_at
	`, id, body.ExpectedAmount, body.ActualAmount, body.Notes).Scan(
		&p.ID, &p.IncomeSourceID, &p.PayDate, &p.ExpectedAmount,
		&p.ActualAmount, &p.Notes, &p.CreatedAt,
	)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "pay period not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, p)
}
