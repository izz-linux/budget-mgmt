package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/izz-linux/budget-mgmt/backend/internal/models"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

// ---------------------------------------------------------------------------
// Income: Create validation
// ---------------------------------------------------------------------------

func TestIncomeCreate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{invalid`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "INVALID_JSON")
}

func TestIncomeCreate_MissingName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"pay_schedule":"weekly","schedule_detail":{"weekday":5}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestIncomeCreate_InvalidSchedule(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"name":"Test","pay_schedule":"daily","schedule_detail":{"weekday":5}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestIncomeCreate_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "pay_schedule", "schedule_detail", "default_amount", "is_active", "effective_from", "created_at", "updated_at"}).
		AddRow(1, "My Job", "biweekly", json.RawMessage(`{"weekday":5,"anchor_date":"2025-01-10"}`), float64Ptr(2500.0), true, (*time.Time)(nil), now, now)

	mock.ExpectQuery("INSERT INTO income_sources").
		WithArgs("My Job", "biweekly", json.RawMessage(`{"weekday":5,"anchor_date":"2025-01-10"}`), float64Ptr(2500.0), (*time.Time)(nil)).
		WillReturnRows(rows)

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"name":"My Job","pay_schedule":"biweekly","schedule_detail":{"weekday":5,"anchor_date":"2025-01-10"},"default_amount":2500}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Income: Get with invalid ID
// ---------------------------------------------------------------------------

func TestIncomeGet_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/income-sources/abc", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "INVALID_ID")
}

// ---------------------------------------------------------------------------
// Income: Update with invalid ID
// ---------------------------------------------------------------------------

func TestIncomeUpdate_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"name":"updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/income-sources/xyz", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "xyz")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestIncomeUpdate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{invalid`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/income-sources/1", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Income: Delete with invalid ID
// ---------------------------------------------------------------------------

func TestIncomeDelete_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/income-sources/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Income: List returns empty array
// ---------------------------------------------------------------------------

func TestIncomeList_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "pay_schedule", "schedule_detail", "default_amount", "is_active", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT (.+) FROM income_sources").WillReturnRows(rows)

	h := NewIncomeHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/income-sources", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data array, got %d items", len(resp.Data))
	}
}

// ---------------------------------------------------------------------------
// Bills: Create validation
// ---------------------------------------------------------------------------

func TestBillCreate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bills", body)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBillCreate_MissingName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	body := bytes.NewBufferString(`{"default_amount":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bills", body)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestBillGet_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBillUpdate_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	body := bytes.NewBufferString(`{"name":"updated"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/bills/xyz", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "xyz")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBillDelete_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bills/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestBillReorder_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewBillHandler(mock)
	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/bills/reorder", body)
	rr := httptest.NewRecorder()
	h.Reorder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Periods: Generate validation
// ---------------------------------------------------------------------------

func TestPeriodGenerate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{invalid`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pay-periods/generate", body)
	rr := httptest.NewRecorder()
	h.Generate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPeriodGenerate_InvalidFromDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{"from":"not-a-date","to":"2025-03-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pay-periods/generate", body)
	rr := httptest.NewRecorder()
	h.Generate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestPeriodGenerate_InvalidToDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{"from":"2025-01-01","to":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pay-periods/generate", body)
	rr := httptest.NewRecorder()
	h.Generate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPeriodUpdate_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{"expected_amount":1500}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pay-periods/abc", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPeriodUpdate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{bad`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pay-periods/1", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Assignments: validation
// ---------------------------------------------------------------------------

func TestAssignmentCreate_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments", body)
	rr := httptest.NewRecorder()
	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAssignmentUpdate_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"status":"paid"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/assignments/abc", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAssignmentUpdateStatus_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"status":"paid"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/xyz/status", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "xyz")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.UpdateStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestAssignmentUpdateStatus_InvalidStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"status":"invalid_status"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/1/status", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.UpdateStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestAssignmentUpdateStatus_ValidStatuses(t *testing.T) {
	validStatuses := []string{"pending", "paid", "deferred", "uncertain", "skipped"}
	for _, status := range validStatuses {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}

		h := NewAssignmentHandler(mock)
		body := bytes.NewBufferString(`{"status":"` + status + `"}`)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/assignments/1/status", body)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(withChiContext(req.Context(), rctx))

		rr := httptest.NewRecorder()

		// Expect the DB call (will fail since mock not set up, but validation should pass)
		mock.ExpectQuery("UPDATE bill_assignments").WillReturnError(fmt.Errorf("mock error"))

		h.UpdateStatus(rr, req)

		// Should NOT be 400 (validation passed) - it will be 404 since DB fails
		if rr.Code == http.StatusBadRequest {
			t.Errorf("status %q should be valid but got 400", status)
		}
		mock.Close()
	}
}

func TestAssignmentDelete_InvalidID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assignments/abc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Optimizer: Suggest validation
// ---------------------------------------------------------------------------

func TestOptimizerSuggest_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewOptimizerHandler(mock)
	body := bytes.NewBufferString(`{bad json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimizer/suggest", body)
	rr := httptest.NewRecorder()
	h.Suggest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Import: Confirm without upload
// ---------------------------------------------------------------------------

func TestImportConfirm_NoPreview(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewImportHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/xlsx/confirm", nil)
	rr := httptest.NewRecorder()
	h.Confirm(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "NO_PREVIEW")
}

// ---------------------------------------------------------------------------
// Import: Upload with no file
// ---------------------------------------------------------------------------

func TestImportUpload_NoFile(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewImportHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/xlsx", nil)
	rr := httptest.NewRecorder()
	h.Upload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// AutoAssign: validation
// ---------------------------------------------------------------------------

func TestAutoAssign_InvalidJSON(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{bad json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "INVALID_JSON")
}

func TestAutoAssign_InvalidFromDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"not-a-date","to":"2026-03-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestAutoAssign_InvalidToDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-01-01","to":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestAutoAssign_NoBills(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"})
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-01-01","to":"2026-03-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	// Should return empty array
	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(resp.Data))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_NoPeriods(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	periodRows := pgxmock.NewRows([]string{"id", "pay_date"})
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-01-01","to":"2026-03-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_MatchesBillsToPeriods(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// Two periods in a future month: the 7th and the 21st
	month := futureMonth(2)
	from, to := monthRange(month)
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(month, 7)).
		AddRow(11, dayIn(month, 21))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// Bill due on the 15th goes to period 10 (the 7th — last period on or before it)
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(1, 1, 10, 100.0))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 10, 100.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_UsesFirstPeriodWhenNoneBeforeDueDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Bill due on the 3rd
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Internet", float64Ptr(50.0), 3, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// Only period is on the 7th (after the due date)
	month := futureMonth(2)
	from, to := monthRange(month)
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(month, 7))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// Should still assign to period 10 (first available in that month)
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, float64Ptr(50.0)).
		WillReturnRows(oneAssignmentRow(1, 1, 10, 50.0))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 10, 50.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_SkipsExistingAssignments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	// Bill already has an assignment for Feb (on period 10) - pre-fetch returns it
	existingRows := pgxmock.NewRows([]string{"bill_id", "pay_period_id", "pay_date", "manually_moved"}).
		AddRow(1, 10, time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC), false)
	mock.ExpectQuery("SELECT ba.bill_id, ba.pay_period_id, pp.pay_date, ba.manually_moved").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(existingRows)

	// No deleted bill+period combos
	deletedRows := pgxmock.NewRows([]string{"bill_id", "pay_period_id"})
	mock.ExpectQuery("SELECT dbp.bill_id, dbp.pay_period_id FROM deleted_bill_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(deletedRows)

	// No INSERT expected - the bill/month combo is already covered

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-02-01","to":"2026-02-28"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	// Should return 201 with empty array (no new assignments created)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_SkipsWhenBillMovedToDifferentPeriod(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Bill due on the 15th
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// Two periods: Feb 7 and Feb 21
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)).
		AddRow(11, time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	// User moved bill from period 10 (Feb 7) to period 11 (Feb 21) — existing assignment on 21st
	existingRows := pgxmock.NewRows([]string{"bill_id", "pay_period_id", "pay_date", "manually_moved"}).
		AddRow(1, 11, time.Date(2026, 2, 21, 0, 0, 0, 0, time.UTC), false)
	mock.ExpectQuery("SELECT ba.bill_id, ba.pay_period_id, pp.pay_date, ba.manually_moved").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(existingRows)

	// No deleted bill+period combos
	deletedRows := pgxmock.NewRows([]string{"bill_id", "pay_period_id"})
	mock.ExpectQuery("SELECT dbp.bill_id, dbp.pay_period_id FROM deleted_bill_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(deletedRows)

	// No INSERT expected — bill already has an assignment for Feb, even though it's on a different period

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-02-01","to":"2026-02-28"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_BiweeklyBillWithAnchorDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Biweekly bill anchored to the 15th of a future month
	m1 := futureMonth(2)
	m2 := futureMonth(3)
	anchor := dayIn(m1, 15)
	anchorJSON := []byte(fmt.Sprintf(`{"anchor_date":%q}`, anchor.Format("2006-01-02")))
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Loan", float64Ptr(200.0), 15, "biweekly", anchorJSON)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// 4 semi-monthly periods across the two months
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(m1, 1)).
		AddRow(11, dayIn(m1, 15)).
		AddRow(12, dayIn(m2, 1)).
		AddRow(13, dayIn(m2, 15))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// The biweekly cycle back-computes to the 1st of m1, so occurrences fall on
	// all 4 periods (one of them absorbing two occurrences).
	// Map iteration order is non-deterministic, so accept the INSERTs in any order.
	for i := 0; i < 4; i++ {
		mock.ExpectQuery("INSERT INTO bill_assignments").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(oneAssignmentRow(i+1, 1, 11, 200.0))
	}

	h := NewAssignmentHandler(mock)
	from := dayIn(m1, 1).Format("2006-01-02")
	to := dayIn(m2, 1).AddDate(0, 1, -1).Format("2006-01-02")
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 4 {
		t.Errorf("expected 4 assignments, got %d; body: %s", got, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_BiweeklyFallsBackWithoutAnchor(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Biweekly bill WITHOUT anchor date — should fall back to monthly
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Loan", float64Ptr(200.0), 15, "biweekly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// One period, on the 7th of a future month
	month := futureMonth(2)
	from, to := monthRange(month)
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(month, 7))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// Falls back to monthly: assigns to period 10
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, float64Ptr(200.0)).
		WillReturnRows(oneAssignmentRow(1, 1, 10, 200.0))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 10, 200.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_QuarterlyBillWithAnchorDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Quarterly bill anchored to the 15th of a future month
	m1 := futureMonth(2)
	m2 := futureMonth(5) // one quarter later
	anchor := dayIn(m1, 15)
	anchorJSON := []byte(fmt.Sprintf(`{"anchor_date":%q}`, anchor.Format("2006-01-02")))
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Insurance", float64Ptr(300.0), 15, "quarterly", anchorJSON)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(m1, 1)).
		AddRow(11, dayIn(m1, 15)).
		AddRow(12, dayIn(m2, 1)).
		AddRow(13, dayIn(m2, 15))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// Two occurrences a quarter apart, each landing on its month's 15th period
	for i := 0; i < 2; i++ {
		mock.ExpectQuery("INSERT INTO bill_assignments").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(oneAssignmentRow(i+1, 1, 11, 300.0))
	}

	h := NewAssignmentHandler(mock)
	from := dayIn(m1, 1).Format("2006-01-02")
	to := dayIn(futureMonth(7), 1).Format("2006-01-02")
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 2 {
		t.Errorf("expected 2 assignments, got %d; body: %s", got, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_AnnualBillWithAnchorDate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Annual bill anchored to the 1st of a future month
	m0 := futureMonth(1)
	m1 := futureMonth(2)
	anchor := dayIn(m1, 1)
	anchorJSON := []byte(fmt.Sprintf(`{"anchor_date":%q}`, anchor.Format("2006-01-02")))
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Car Registration", float64Ptr(500.0), 1, "annual", anchorJSON)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(m0, 15)).
		AddRow(11, dayIn(m1, 1)).
		AddRow(12, dayIn(m1, 15))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// The single annual occurrence lands on period 11 (its own pay date)
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(oneAssignmentRow(1, 1, 11, 500.0))

	h := NewAssignmentHandler(mock)
	from := dayIn(m0, 1).Format("2006-01-02")
	to := dayIn(futureMonth(11), 1).Format("2006-01-02")
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 11, 500.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_QuarterlyFallsBackWithoutAnchor(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Quarterly bill WITHOUT anchor date — should fall back to monthly
	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Insurance", float64Ptr(300.0), 15, "quarterly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	// One period, on the 7th of a future month
	month := futureMonth(2)
	from, to := monthRange(month)
	periodRows := pgxmock.NewRows([]string{"id", "pay_date"}).
		AddRow(10, dayIn(month, 7))
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(periodRows)

	expectAutoAssignPrefetch(mock, nil)

	// Falls back to monthly: assigns to period 10
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, float64Ptr(300.0)).
		WillReturnRows(oneAssignmentRow(1, 1, 10, 300.0))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q}`, from, to))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 10, 300.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAutoAssign_BillQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnError(fmt.Errorf("db connection lost"))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-01-01","to":"2026-03-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "DB_ERROR")
}

func TestAutoAssign_PeriodQueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(`{"from":"2026-01-01","to":"2026-03-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "DB_ERROR")
}

// ---------------------------------------------------------------------------
// Bill Delete: Success cases
// ---------------------------------------------------------------------------

func TestBillDelete_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("UPDATE bills SET is_active = false").
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bills/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBillDelete_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("UPDATE bills SET is_active = false").
		WithArgs(999).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bills/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "NOT_FOUND")
}

func TestBillDelete_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec("UPDATE bills SET is_active = false").
		WithArgs(1).
		WillReturnError(fmt.Errorf("connection lost"))

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bills/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "DB_ERROR")
}

// ---------------------------------------------------------------------------
// Assignment Delete: Success cases
// ---------------------------------------------------------------------------

func TestAssignmentDelete_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// First query to get bill_id and pay_period_id
	mock.ExpectQuery("SELECT bill_id, pay_period_id FROM bill_assignments").
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "pay_period_id"}).AddRow(1, 10))

	// Then delete
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(5).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// Then track the deletion
	mock.ExpectExec("INSERT INTO deleted_bill_periods").
		WithArgs(1, 10).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	h := NewAssignmentHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assignments/5", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "5")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAssignmentDelete_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Query returns no rows (assignment not found)
	mock.ExpectQuery("SELECT bill_id, pay_period_id FROM bill_assignments").
		WithArgs(999).
		WillReturnError(fmt.Errorf("no rows in result set"))

	h := NewAssignmentHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assignments/999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "NOT_FOUND")
}

func TestAssignmentDelete_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// First query succeeds
	mock.ExpectQuery("SELECT bill_id, pay_period_id FROM bill_assignments").
		WithArgs(1).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "pay_period_id"}).AddRow(1, 10))

	// Delete fails
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(1).
		WillReturnError(fmt.Errorf("connection lost"))

	h := NewAssignmentHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/assignments/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Delete(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "DB_ERROR")
}

// ---------------------------------------------------------------------------
// Period Update: Success and error cases
// ---------------------------------------------------------------------------

func TestPeriodUpdate_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Now()
	payDate := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{
		"id", "income_source_id", "pay_date", "expected_amount", "actual_amount", "notes", "created_at",
	}).AddRow(1, 1, payDate, float64Ptr(2000.0), (*float64)(nil), "", now)

	mock.ExpectQuery("UPDATE pay_periods SET").
		WithArgs(1, float64Ptr(2000.0), (*float64)(nil), (*string)(nil)).
		WillReturnRows(rows)

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{"expected_amount":2000}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pay-periods/1", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPeriodUpdate_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("UPDATE pay_periods SET").
		WithArgs(999, float64Ptr(1500.0), (*float64)(nil), (*string)(nil)).
		WillReturnError(fmt.Errorf("no rows in result set"))

	h := NewPeriodHandler(mock)
	body := bytes.NewBufferString(`{"expected_amount":1500}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/pay-periods/999", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(withChiContext(req.Context(), rctx))

	rr := httptest.NewRecorder()
	h.Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Income: Create one_time schedule
// ---------------------------------------------------------------------------

func TestIncomeCreate_OneTime_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "pay_schedule", "schedule_detail", "default_amount", "is_active", "effective_from", "created_at", "updated_at"}).
		AddRow(1, "Year-End Bonus", "one_time", json.RawMessage(`{"date":"2026-03-15"}`), float64Ptr(5000.0), true, (*time.Time)(nil), now, now)

	mock.ExpectQuery("INSERT INTO income_sources").
		WithArgs("Year-End Bonus", "one_time", json.RawMessage(`{"date":"2026-03-15"}`), float64Ptr(5000.0), (*time.Time)(nil)).
		WillReturnRows(rows)

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"name":"Year-End Bonus","pay_schedule":"one_time","schedule_detail":{"date":"2026-03-15"},"default_amount":5000}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestIncomeCreate_InvalidSchedule_StillRejects(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := NewIncomeHandler(mock)
	body := bytes.NewBufferString(`{"name":"Test","pay_schedule":"daily","schedule_detail":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", body)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

// ---------------------------------------------------------------------------
// Bill List: active filter
// ---------------------------------------------------------------------------

func TestBillList_ActiveFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{
		"id", "name", "default_amount", "due_day", "recurrence",
		"recurrence_detail", "is_autopay", "category", "notes",
		"is_active", "sort_order", "created_at", "updated_at",
		"cc_id", "cc_label", "cc_statement_day", "cc_due_day", "cc_issuer",
	})
	mock.ExpectQuery("SELECT (.+) FROM bills (.+) WHERE b.is_active = true").WillReturnRows(rows)

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills?active=true", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBillList_NoFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{
		"id", "name", "default_amount", "due_day", "recurrence",
		"recurrence_detail", "is_autopay", "category", "notes",
		"is_active", "sort_order", "created_at", "updated_at",
		"cc_id", "cc_label", "cc_statement_day", "cc_due_day", "cc_issuer",
	})
	// Should NOT have WHERE is_active in query
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(rows)

	h := NewBillHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills", nil)
	rr := httptest.NewRecorder()
	h.List(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func withChiContext(ctx interface{ Value(any) any }, rctx *chi.Context) interface{ Deadline() (time.Time, bool); Done() <-chan struct{}; Err() error; Value(any) any } {
	return chiContextWrapper{ctx, rctx}
}

type chiContextWrapper struct {
	parent interface{ Value(any) any }
	rctx   *chi.Context
}

func (c chiContextWrapper) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c chiContextWrapper) Done() <-chan struct{}        { return nil }
func (c chiContextWrapper) Err() error                   { return nil }
func (c chiContextWrapper) Value(key any) any {
	if key == chi.RouteCtxKey {
		return c.rctx
	}
	return c.parent.Value(key)
}

func assertErrorCode(t *testing.T, body []byte, expectedCode string) {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if resp.Error.Code != expectedCode {
		t.Errorf("expected error code %q, got %q", expectedCode, resp.Error.Code)
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

// ---------------------------------------------------------------------------
// AutoAssign test helpers
//
// AutoAssign deliberately ignores pay periods before today, so any test that
// hard-codes calendar dates stops exercising the insert path the moment those
// dates pass — the handler quietly does nothing and the test's only real
// assertion (mock expectations) starts failing. Build the dates relative to
// time.Now() instead.
// ---------------------------------------------------------------------------

// futureMonth returns the first day of the month `ahead` months from now (UTC).
func futureMonth(ahead int) time.Time {
	n := time.Now()
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, ahead, 0)
}

// dayIn returns the given day-of-month within base's month (UTC).
func dayIn(base time.Time, day int) time.Time {
	return time.Date(base.Year(), base.Month(), day, 0, 0, 0, 0, time.UTC)
}

// monthRange returns from/to strings spanning the whole of base's month.
func monthRange(base time.Time) (string, string) {
	first := dayIn(base, 1)
	last := first.AddDate(0, 1, -1)
	return first.Format("2006-01-02"), last.Format("2006-01-02")
}

// assignmentRows builds a row set with exactly the columns the handler scans.
// The old fixtures returned 14 columns while the handler scanned 16, which only
// went unnoticed because insertAssignment swallowed the resulting scan error.
func assignmentRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "bill_id", "pay_period_id", "planned_amount", "forecast_amount",
		"actual_amount", "status", "deferred_to_id", "is_extra", "extra_name",
		"notes", "manually_moved", "is_sinking_fund", "sinking_fund_for_period_id",
		"created_at", "updated_at",
	})
}

func addAssignmentRow(rows *pgxmock.Rows, id, billID, periodID int, amount float64) *pgxmock.Rows {
	now := time.Now()
	return rows.AddRow(
		id, billID, periodID, float64Ptr(amount), (*float64)(nil), (*float64)(nil),
		"pending", (*int)(nil), false, "", "", false, false, (*int)(nil), now, now,
	)
}

func oneAssignmentRow(id, billID, periodID int, amount float64) *pgxmock.Rows {
	return addAssignmentRow(assignmentRows(), id, billID, periodID, amount)
}

// decodeAssignments unwraps the handler's {"data": [...]} envelope so tests can
// assert on what was actually created, not merely that the mock was called.
func decodeAssignments(t *testing.T, body []byte) []models.BillAssignment {
	t.Helper()
	var resp struct {
		Data []models.BillAssignment `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to parse assignments response: %v; body: %s", err, body)
	}
	return resp.Data
}

// assertSingleAssignment checks the handler actually returned one assignment
// landing on the expected period with the expected amount.
func assertSingleAssignment(t *testing.T, body []byte, wantBillID, wantPeriodID int, wantAmount float64) {
	t.Helper()
	created := decodeAssignments(t, body)
	if len(created) != 1 {
		t.Fatalf("expected exactly 1 assignment, got %d; body: %s", len(created), body)
	}
	a := created[0]
	if a.BillID != wantBillID {
		t.Errorf("expected bill_id %d, got %d", wantBillID, a.BillID)
	}
	if a.PayPeriodID != wantPeriodID {
		t.Errorf("expected pay_period_id %d, got %d", wantPeriodID, a.PayPeriodID)
	}
	if a.PlannedAmount == nil || *a.PlannedAmount != wantAmount {
		t.Errorf("expected planned_amount %v, got %v", wantAmount, a.PlannedAmount)
	}
}

// expectAutoAssignPrefetch sets up the two lookups AutoAssign always performs
// after loading bills and periods: existing assignments and deletion tombstones.
func expectAutoAssignPrefetch(mock pgxmock.PgxPoolIface, existing *pgxmock.Rows) {
	if existing == nil {
		existing = pgxmock.NewRows([]string{"bill_id", "pay_period_id", "pay_date", "manually_moved"})
	}
	mock.ExpectQuery("SELECT ba.bill_id, ba.pay_period_id, pp.pay_date, ba.manually_moved").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(existing)
	mock.ExpectQuery("SELECT dbp.bill_id, dbp.pay_period_id FROM deleted_bill_periods").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "pay_period_id"}))
}
