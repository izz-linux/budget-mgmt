package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
	"github.com/izz-linux/budget-mgmt/backend/internal/services"
)

type ImportHandler struct {
	db       *pgxpool.Pool
	importer *services.XLSXImporter
	// Store the last preview for confirmation
	lastPreview *services.ImportPreview
	lastFile    string
}

func NewImportHandler(db *pgxpool.Pool) *ImportHandler {
	return &ImportHandler{
		db:       db,
		importer: services.NewXLSXImporter(),
	}
}

func (h *ImportHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Max 10MB file
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		models.WriteError(w, http.StatusBadRequest, "NO_FILE", "no file uploaded")
		return
	}
	defer file.Close()

	// Save to temp file
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, "budget-import-"+header.Filename)
	dst, err := os.Create(tmpPath)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "FILE_ERROR", err.Error())
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		models.WriteError(w, http.StatusInternalServerError, "FILE_ERROR", err.Error())
		return
	}
	dst.Close()

	// Parse the file
	preview, err := h.importer.ParseFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		models.WriteError(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
		return
	}

	h.lastPreview = preview
	h.lastFile = tmpPath

	models.WriteJSON(w, http.StatusOK, preview)
}

func (h *ImportHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.lastPreview == nil {
		models.WriteError(w, http.StatusBadRequest, "NO_PREVIEW", "no pending import to confirm. Upload a file first.")
		return
	}

	defer func() {
		if h.lastFile != "" {
			os.Remove(h.lastFile)
		}
		h.lastPreview = nil
		h.lastFile = ""
	}()

	tx, err := h.db.Begin(ctx)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer tx.Rollback(ctx)

	imported := 0
	for i, pb := range h.lastPreview.Bills {
		var billID int
		recurrence := "monthly"

		err := tx.QueryRow(ctx, `
			INSERT INTO bills (name, default_amount, due_day, recurrence, is_autopay, category, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, pb.Name, pb.DefaultAmt, pb.DueDay, recurrence, pb.IsAutopay, pb.Category, i).Scan(&billID)
		if err != nil {
			models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		if pb.CreditCard != nil {
			_, err := tx.Exec(ctx, `
				INSERT INTO credit_cards (bill_id, card_label, statement_day, due_day, issuer)
				VALUES ($1, $2, $3, $4, $5)
			`, billID, pb.CreditCard.CardLabel, pb.CreditCard.StatementDay, pb.CreditCard.DueDay, pb.CreditCard.Issuer)
			if err != nil {
				models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
				return
			}
		}
		imported++
	}

	// Record import
	_, err = tx.Exec(ctx, `
		INSERT INTO import_history (filename, row_count, period_count, status)
		VALUES ($1, $2, $3, 'completed')
	`, filepath.Base(h.lastFile), imported, h.lastPreview.PeriodCount)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	models.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"imported_bills":   imported,
		"period_count":     h.lastPreview.PeriodCount,
		"status":           "completed",
	})
}

func (h *ImportHandler) History(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.db.Query(ctx, `
		SELECT id, filename, imported_at, row_count, period_count, status, error_log
		FROM import_history ORDER BY imported_at DESC
	`)
	if err != nil {
		models.WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var id, rowCount, periodCount int
		var filename, status string
		var importedAt interface{}
		var errorLog *string
		if err := rows.Scan(&id, &filename, &importedAt, &rowCount, &periodCount, &status, &errorLog); err != nil {
			continue
		}
		history = append(history, map[string]interface{}{
			"id":           id,
			"filename":     filename,
			"imported_at":  importedAt,
			"row_count":    rowCount,
			"period_count": periodCount,
			"status":       status,
		})
	}

	if history == nil {
		history = []map[string]interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"data": history})
}
