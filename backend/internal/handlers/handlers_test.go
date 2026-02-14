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
	rows := pgxmock.NewRows([]string{"id", "name", "pay_schedule", "schedule_detail", "default_amount", "is_active", "created_at", "updated_at"}).
		AddRow(1, "My Job", "biweekly", json.RawMessage(`{"weekday":5,"anchor_date":"2025-01-10"}`), float64Ptr(2500.0), true, now, now)

	mock.ExpectQuery("INSERT INTO income_sources").
		WithArgs("My Job", "biweekly", json.RawMessage(`{"weekday":5,"anchor_date":"2025-01-10"}`), float64Ptr(2500.0)).
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
