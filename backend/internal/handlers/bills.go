package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
)

type BillHandler struct {
	db DBTX
}

func NewBillHandler(db DBTX) *BillHandler {
	return &BillHandler{db: db}
}

func (h *BillHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeOnly := r.URL.Query().Get("active") == "true"

	query := `
		SELECT b.id, b.name, b.default_amount, b.due_day, b.recurrence,
		       b.recurrence_detail, b.is_autopay, COALESCE(b.category, ''), COALESCE(b.notes, ''),
		       b.is_active, b.sort_order, b.created_at, b.updated_at,
		       cc.id, cc.card_label, cc.statement_day, cc.due_day, cc.issuer, cc.created_at
		FROM bills b
		LEFT JOIN credit_cards cc ON cc.bill_id = b.id
	`
	if activeOnly {
		query += " WHERE b.is_active = true"
	}
	query += " ORDER BY b.sort_order, b.id"

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var bills []models.Bill
	for rows.Next() {
		var b models.Bill
		var ccID *int
		var ccLabel, ccIssuer *string
		var ccStatementDay, ccDueDay *int
		var ccCreatedAt *interface{}

		err := rows.Scan(
			&b.ID, &b.Name, &b.DefaultAmount, &b.DueDay, &b.Recurrence,
			&b.RecurrenceDetail, &b.IsAutopay, &b.Category, &b.Notes,
			&b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt,
			&ccID, &ccLabel, &ccStatementDay, &ccDueDay, &ccIssuer, &ccCreatedAt,
		)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		if ccID != nil {
			b.CreditCard = &models.CreditCard{
				ID:           *ccID,
				BillID:       b.ID,
				StatementDay: *ccStatementDay,
				DueDay:       *ccDueDay,
			}
			if ccLabel != nil {
				b.CreditCard.CardLabel = *ccLabel
			}
			if ccIssuer != nil {
				b.CreditCard.Issuer = *ccIssuer
			}
		}
		bills = append(bills, b)
	}

	if bills == nil {
		bills = []models.Bill{}
	}
	models.WriteJSON(w, http.StatusOK, bills)
}

func (h *BillHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var b models.Bill
	err = h.db.QueryRow(ctx, `
		SELECT id, name, default_amount, due_day, recurrence, recurrence_detail,
		       is_autopay, COALESCE(category, ''), COALESCE(notes, ''), is_active, sort_order, created_at, updated_at
		FROM bills WHERE id = $1
	`, id).Scan(
		&b.ID, &b.Name, &b.DefaultAmount, &b.DueDay, &b.Recurrence,
		&b.RecurrenceDetail, &b.IsAutopay, &b.Category, &b.Notes,
		&b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "bill not found")
		return
	}

	// Check for credit card
	var cc models.CreditCard
	err = h.db.QueryRow(ctx, `
		SELECT id, bill_id, card_label, statement_day, due_day, issuer, created_at
		FROM credit_cards WHERE bill_id = $1
	`, id).Scan(&cc.ID, &cc.BillID, &cc.CardLabel, &cc.StatementDay, &cc.DueDay, &cc.Issuer, &cc.CreatedAt)
	if err == nil {
		b.CreditCard = &cc
	}

	models.WriteJSON(w, http.StatusOK, b)
}

func (h *BillHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.CreateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.Name == "" {
		models.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if req.Recurrence == "" {
		req.Recurrence = "monthly"
	}

	var b models.Bill
	err := h.db.QueryRow(ctx, `
		INSERT INTO bills (name, default_amount, due_day, recurrence, recurrence_detail,
		                   is_autopay, category, notes, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, default_amount, due_day, recurrence, recurrence_detail,
		          is_autopay, COALESCE(category, ''), COALESCE(notes, ''), is_active, sort_order, created_at, updated_at
	`, req.Name, req.DefaultAmount, req.DueDay, req.Recurrence, req.RecurrenceDetail,
		req.IsAutopay, req.Category, req.Notes, req.SortOrder,
	).Scan(
		&b.ID, &b.Name, &b.DefaultAmount, &b.DueDay, &b.Recurrence,
		&b.RecurrenceDetail, &b.IsAutopay, &b.Category, &b.Notes,
		&b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	// Create credit card if provided
	if req.CreditCard != nil {
		var cc models.CreditCard
		err := h.db.QueryRow(ctx, `
			INSERT INTO credit_cards (bill_id, card_label, statement_day, due_day, issuer)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, bill_id, card_label, statement_day, due_day, issuer, created_at
		`, b.ID, req.CreditCard.CardLabel, req.CreditCard.StatementDay,
			req.CreditCard.DueDay, req.CreditCard.Issuer,
		).Scan(&cc.ID, &cc.BillID, &cc.CardLabel, &cc.StatementDay, &cc.DueDay, &cc.Issuer, &cc.CreatedAt)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		b.CreditCard = &cc
	}

	models.WriteJSON(w, http.StatusCreated, b)
}

func (h *BillHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	var req models.UpdateBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	var b models.Bill
	err = h.db.QueryRow(ctx, `
		UPDATE bills SET
			name = COALESCE($2, name),
			default_amount = COALESCE($3, default_amount),
			due_day = COALESCE($4, due_day),
			recurrence = COALESCE($5, recurrence),
			recurrence_detail = COALESCE($6, recurrence_detail),
			is_autopay = COALESCE($7, is_autopay),
			category = COALESCE($8, category),
			notes = COALESCE($9, notes),
			is_active = COALESCE($10, is_active),
			sort_order = COALESCE($11, sort_order),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, default_amount, due_day, recurrence, recurrence_detail,
		          is_autopay, COALESCE(category, ''), COALESCE(notes, ''), is_active, sort_order, created_at, updated_at
	`, id, req.Name, req.DefaultAmount, req.DueDay, req.Recurrence,
		req.RecurrenceDetail, req.IsAutopay, req.Category, req.Notes,
		req.IsActive, req.SortOrder,
	).Scan(
		&b.ID, &b.Name, &b.DefaultAmount, &b.DueDay, &b.Recurrence,
		&b.RecurrenceDetail, &b.IsAutopay, &b.Category, &b.Notes,
		&b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "bill not found")
		return
	}

	models.WriteJSON(w, http.StatusOK, b)
}

func (h *BillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_ID", "id must be an integer")
		return
	}

	tag, err := h.db.Exec(ctx, `UPDATE bills SET is_active = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		models.WriteError(w, http.StatusNotFound, "NOT_FOUND", "bill not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BillHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req models.ReorderBillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		models.WriteError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback(ctx)

	for _, order := range req.Orders {
		_, err := tx.Exec(ctx, `UPDATE bills SET sort_order = $2, updated_at = NOW() WHERE id = $1`, order.ID, order.SortOrder)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
