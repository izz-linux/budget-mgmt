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
// Income: schedule_detail validation (Create)
// ---------------------------------------------------------------------------

// TestIncomeCreate_RejectsBadScheduleDetail covers the hole that let an
// out-of-range weekday reach the database, where the period generator would
// later spin on it forever. Create validated only the pay_schedule string.
func TestIncomeCreate_RejectsBadScheduleDetail(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"weekly weekday too high", `{"name":"J","pay_schedule":"weekly","schedule_detail":{"weekday":7}}`},
		{"weekly weekday negative", `{"name":"J","pay_schedule":"weekly","schedule_detail":{"weekday":-1}}`},
		{"weekly weekday absurd", `{"name":"J","pay_schedule":"weekly","schedule_detail":{"weekday":99}}`},
		{"weekly detail missing", `{"name":"J","pay_schedule":"weekly"}`},
		{"biweekly anchor missing", `{"name":"J","pay_schedule":"biweekly","schedule_detail":{"weekday":5}}`},
		{"biweekly anchor malformed", `{"name":"J","pay_schedule":"biweekly","schedule_detail":{"anchor_date":"01/10/2026"}}`},
		{"semimonthly one day", `{"name":"J","pay_schedule":"semimonthly","schedule_detail":{"days":[15]}}`},
		{"semimonthly three days", `{"name":"J","pay_schedule":"semimonthly","schedule_detail":{"days":[1,15,28]}}`},
		{"semimonthly day zero", `{"name":"J","pay_schedule":"semimonthly","schedule_detail":{"days":[0,15]}}`},
		{"semimonthly day 32", `{"name":"J","pay_schedule":"semimonthly","schedule_detail":{"days":[1,32]}}`},
		{"one_time date missing", `{"name":"J","pay_schedule":"one_time","schedule_detail":{}}`},
		{"one_time date malformed", `{"name":"J","pay_schedule":"one_time","schedule_detail":{"date":"nope"}}`},
		{"unknown schedule", `{"name":"J","pay_schedule":"daily","schedule_detail":{"weekday":1}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()
			// No DB expectations at all: validation must reject before any write.

			h := NewIncomeHandler(mock)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", bytes.NewBufferString(tc.body))
			rr := httptest.NewRecorder()
			h.Create(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
			}
			assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestIncomeCreate_AcceptsValidScheduleDetail(t *testing.T) {
	cases := []struct {
		name     string
		schedule string
		detail   string
	}{
		{"weekly sunday", "weekly", `{"weekday":0}`},
		{"weekly saturday", "weekly", `{"weekday":6}`},
		{"biweekly with anchor", "biweekly", `{"weekday":5,"anchor_date":"2026-01-09"}`},
		{"semimonthly", "semimonthly", `{"days":[1,16]}`},
		{"semimonthly end of month", "semimonthly", `{"days":[15,31]}`},
		{"one_time", "one_time", `{"date":"2026-12-24"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			now := time.Now()
			mock.ExpectQuery("INSERT INTO income_sources").
				WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
					pgxmock.AnyArg(), pgxmock.AnyArg()).
				WillReturnRows(pgxmock.NewRows([]string{
					"id", "name", "pay_schedule", "schedule_detail", "default_amount",
					"is_active", "effective_from", "created_at", "updated_at",
				}).AddRow(1, "J", tc.schedule, json.RawMessage(tc.detail),
					(*float64)(nil), true, (*time.Time)(nil), now, now))

			h := NewIncomeHandler(mock)
			body := fmt.Sprintf(`{"name":"J","pay_schedule":%q,"schedule_detail":%s}`, tc.schedule, tc.detail)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/income-sources", bytes.NewBufferString(body))
			rr := httptest.NewRecorder()
			h.Create(rr, req)

			if rr.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Income: schedule validation (Update)
// ---------------------------------------------------------------------------

func updateIncomeRequest(t *testing.T, mock pgxmock.PgxPoolIface, id int, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewIncomeHandler(mock)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/income-sources/1", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fmt.Sprint(id))
	req = req.WithContext(withChiContext(req.Context(), rctx))
	rr := httptest.NewRecorder()
	h.Update(rr, req)
	return rr
}

func expectCurrentSchedule(mock pgxmock.PgxPoolIface, schedule, detail string) {
	mock.ExpectQuery("SELECT pay_schedule, schedule_detail FROM income_sources").
		WithArgs(1).
		WillReturnRows(pgxmock.NewRows([]string{"pay_schedule", "schedule_detail"}).
			AddRow(schedule, json.RawMessage(detail)))
}

// TestIncomeUpdate_RejectsUnknownPaySchedule pins the asymmetry the fix closed:
// Update validated nothing, so it accepted a pay_schedule Create rejects.
func TestIncomeUpdate_RejectsUnknownPaySchedule(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectCurrentSchedule(mock, "weekly", `{"weekday":5}`)

	rr := updateIncomeRequest(t, mock, 1, `{"pay_schedule":"daily"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestIncomeUpdate_RejectsBadDetailAgainstStoredSchedule changes only the
// detail; the schedule it must be checked against has to come from the DB.
func TestIncomeUpdate_RejectsBadDetailAgainstStoredSchedule(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectCurrentSchedule(mock, "weekly", `{"weekday":5}`)

	rr := updateIncomeRequest(t, mock, 1, `{"schedule_detail":{"weekday":9}}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestIncomeUpdate_RejectsScheduleChangeIncompatibleWithStoredDetail changes
// only the schedule; the stored detail must be re-checked against the new one.
func TestIncomeUpdate_RejectsScheduleChangeIncompatibleWithStoredDetail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Stored detail is a weekly one; switching to biweekly needs an anchor_date.
	expectCurrentSchedule(mock, "weekly", `{"weekday":5}`)

	rr := updateIncomeRequest(t, mock, 1, `{"pay_schedule":"biweekly"}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr.Body.Bytes(), "VALIDATION_ERROR")
}

func TestIncomeUpdate_AcceptsValidCombination(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectCurrentSchedule(mock, "weekly", `{"weekday":5}`)

	now := time.Now()
	// id, pay_schedule, schedule_detail
	mock.ExpectQuery("UPDATE income_sources SET").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "name", "pay_schedule", "schedule_detail", "default_amount",
			"is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(1, "J", "biweekly", json.RawMessage(`{"anchor_date":"2026-01-09"}`),
			(*float64)(nil), true, (*time.Time)(nil), now, now))

	rr := updateIncomeRequest(t, mock, 1,
		`{"pay_schedule":"biweekly","schedule_detail":{"anchor_date":"2026-01-09"}}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestIncomeUpdate_SkipsScheduleLookupWhenScheduleUntouched keeps the extra
// SELECT off the path of updates that cannot affect the schedule.
func TestIncomeUpdate_SkipsScheduleLookupWhenScheduleUntouched(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	now := time.Now()
	// id, name — and no preceding SELECT, because the schedule is untouched
	mock.ExpectQuery("UPDATE income_sources SET").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{
			"id", "name", "pay_schedule", "schedule_detail", "default_amount",
			"is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(1, "Renamed", "weekly", json.RawMessage(`{"weekday":5}`),
			(*float64)(nil), true, (*time.Time)(nil), now, now))

	rr := updateIncomeRequest(t, mock, 1, `{"name":"Renamed"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AutoAssign: force semantics
// ---------------------------------------------------------------------------

// autoAssignForceFixture wires up a bill due on the 15th with two periods (the
// 7th and the 21st) and one existing manually-moved assignment on the period
// given by manualPeriodID.
func autoAssignForceFixture(t *testing.T, mock pgxmock.PgxPoolIface, manualPeriodID int, manuallyMoved bool) (string, string) {
	t.Helper()

	billRows := pgxmock.NewRows([]string{"id", "name", "default_amount", "due_day", "recurrence", "recurrence_detail"}).
		AddRow(1, "Electric", float64Ptr(100.0), 15, "monthly", nil)
	mock.ExpectQuery("SELECT (.+) FROM bills").WillReturnRows(billRows)

	month := futureMonth(2)
	from, to := monthRange(month)
	mock.ExpectQuery("SELECT pp.id, pp.pay_date FROM pay_periods").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "pay_date"}).
			AddRow(10, dayIn(month, 7)).
			AddRow(11, dayIn(month, 21)))

	manualDay := 7
	if manualPeriodID == 11 {
		manualDay = 21
	}
	existing := pgxmock.NewRows([]string{"bill_id", "pay_period_id", "pay_date", "manually_moved"}).
		AddRow(1, manualPeriodID, dayIn(month, manualDay), manuallyMoved)
	expectAutoAssignPrefetch(mock, existing)

	return from, to
}

func runAutoAssign(t *testing.T, mock pgxmock.PgxPoolIface, from, to string, force bool) *httptest.ResponseRecorder {
	t.Helper()
	h := NewAssignmentHandler(mock)
	body := bytes.NewBufferString(fmt.Sprintf(`{"from":%q,"to":%q,"force":%t}`, from, to, force))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assignments/auto-assign", body)
	rr := httptest.NewRecorder()
	h.AutoAssign(rr, req)
	return rr
}

// TestAutoAssign_WithoutForceLeavesManualPlacement is the baseline: a manually
// moved bill stays where the user put it.
func TestAutoAssign_WithoutForceLeavesManualPlacement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Manual placement on period 11 (the 21st); the computed period is 10.
	from, to := autoAssignForceFixture(t, mock, 11, true)
	// No DELETE and no INSERT expected.

	rr := runAutoAssign(t, mock, from, to, false)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 0 {
		t.Errorf("expected no assignments created, got %d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAutoAssign_ForceRelocatesManualPlacement is the bug: the manually-moved
// branch was unreachable (manuallyMovedBills is a strict subset of
// existingBillMonths), so force did nothing at all. It must now delete the
// manual placement and insert on the computed period — not merely insert, which
// would leave the bill on two periods in the same month.
func TestAutoAssign_ForceRelocatesManualPlacement(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Manual placement on period 11; the computed period for a bill due on the
	// 15th is period 10 (the 7th).
	from, to := autoAssignForceFixture(t, mock, 11, true)

	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(1, 11).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(1, 1, 10, 100.0))

	rr := runAutoAssign(t, mock, from, to, true)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertSingleAssignment(t, rr.Body.Bytes(), 1, 10, 100.0)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAutoAssign_ForceIsNoOpWhenPlacementAlreadyCorrect guards against churn:
// deleting and reinserting an identical row would reset the manual flag and
// bump timestamps for no reason.
func TestAutoAssign_ForceIsNoOpWhenPlacementAlreadyCorrect(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Manual placement already on period 10, which is also the computed period.
	from, to := autoAssignForceFixture(t, mock, 10, true)
	// No DELETE and no INSERT expected.

	rr := runAutoAssign(t, mock, from, to, true)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 0 {
		t.Errorf("expected no churn, but %d assignments were created", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAutoAssign_ForceIgnoresNonManualAssignments confirms force only touches
// placements the user made by hand — an ordinary auto-assigned row that happens
// to sit on a different period must be left alone.
func TestAutoAssign_ForceIgnoresNonManualAssignments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// Existing assignment on period 11 but NOT manually moved.
	from, to := autoAssignForceFixture(t, mock, 11, false)
	// No DELETE and no INSERT expected.

	rr := runAutoAssign(t, mock, from, to, true)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 0 {
		t.Errorf("force should not touch non-manual assignments, but created %d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Optimizer Apply
// ---------------------------------------------------------------------------

func applyMoves(t *testing.T, mock pgxmock.PgxPoolIface, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewOptimizerHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/optimizer/apply", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.Apply(rr, req)
	return rr
}

func expectPeriodExists(mock pgxmock.PgxPoolIface, id int) {
	mock.ExpectQuery("SELECT id FROM pay_periods").
		WithArgs(id).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(id))
}

// TestOptimizerApply_UsesTransaction pins the core fix: the batch was a series
// of bare DELETE/INSERT statements on the pool, so a mid-batch failure left
// assignments destroyed with nothing put back.
func TestOptimizerApply_UsesTransaction(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	expectPeriodExists(mock, 20)
	mock.ExpectQuery("SELECT bill_id, planned_amount FROM bill_assignments").
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "planned_amount"}).AddRow(1, float64Ptr(100.0)))
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(5).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 20, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(50, 1, 20, 100.0))
	mock.ExpectCommit()

	rr := applyMoves(t, mock, `{"moves":[{"assignment_id":5,"to_period_id":20}]}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestOptimizerApply_RejectsUnknownTargetPeriodBeforeAnyDelete is the ordering
// guarantee: validation happens up front, so a bad target cannot destroy rows
// belonging to earlier moves in the same batch.
func TestOptimizerApply_RejectsUnknownTargetPeriodBeforeAnyDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	expectPeriodExists(mock, 20)
	// Second move targets a period that does not exist.
	mock.ExpectQuery("SELECT id FROM pay_periods").
		WithArgs(999).
		WillReturnRows(pgxmock.NewRows([]string{"id"}))
	// Crucially: NO delete, NO insert — just a rollback.
	mock.ExpectRollback()

	rr := applyMoves(t, mock, `{"moves":[
		{"assignment_id":5,"to_period_id":20},
		{"assignment_id":6,"to_period_id":999}
	]}`)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr.Body.Bytes(), "NOT_FOUND")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestOptimizerApply_FailedLaterMoveRollsBackEarlierOnes is the data-loss case
// the transaction exists for.
func TestOptimizerApply_FailedLaterMoveRollsBackEarlierOnes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	expectPeriodExists(mock, 20)
	expectPeriodExists(mock, 21)

	// First move succeeds end to end.
	mock.ExpectQuery("SELECT bill_id, planned_amount FROM bill_assignments").
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "planned_amount"}).AddRow(1, float64Ptr(100.0)))
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(5).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 20, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(50, 1, 20, 100.0))

	// Second move blows up after the first has already deleted a row.
	mock.ExpectQuery("SELECT bill_id, planned_amount FROM bill_assignments").
		WithArgs(6).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "planned_amount"}).AddRow(2, float64Ptr(75.0)))
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(6).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(2, 21, float64Ptr(75.0)).
		WillReturnError(fmt.Errorf("constraint violation"))

	// The whole batch must be undone, including the first move.
	mock.ExpectRollback()

	rr := applyMoves(t, mock, `{"moves":[
		{"assignment_id":5,"to_period_id":20},
		{"assignment_id":6,"to_period_id":21}
	]}`)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body: %s", rr.Code, rr.Body.String())
	}
	assertErrorCode(t, rr.Body.Bytes(), "DB_ERROR")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestOptimizerApply_ChainedMovesFollowNewRowID covers the second optimizer
// bug: the optimizer can suggest moving one assignment twice (A->B then B->C),
// so the same assignment_id appears twice in a batch. The first move replaces
// the row, giving it a new id, and the second move used to 404 on the old one.
func TestOptimizerApply_ChainedMovesFollowNewRowID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	expectPeriodExists(mock, 20)
	expectPeriodExists(mock, 30)

	// Move 1: assignment 5 -> period 20, producing new row id 50.
	mock.ExpectQuery("SELECT bill_id, planned_amount FROM bill_assignments").
		WithArgs(5).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "planned_amount"}).AddRow(1, float64Ptr(100.0)))
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(5).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 20, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(50, 1, 20, 100.0))

	// Move 2 still references assignment_id 5, but the live row is now 50 —
	// the lookup and the delete must both use 50, not 5.
	mock.ExpectQuery("SELECT bill_id, planned_amount FROM bill_assignments").
		WithArgs(50).
		WillReturnRows(pgxmock.NewRows([]string{"bill_id", "planned_amount"}).AddRow(1, float64Ptr(100.0)))
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(50).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 30, float64Ptr(100.0)).
		WillReturnRows(oneAssignmentRow(60, 1, 30, 100.0))

	mock.ExpectCommit()

	rr := applyMoves(t, mock, `{"moves":[
		{"assignment_id":5,"to_period_id":20},
		{"assignment_id":5,"to_period_id":30}
	]}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	applied := decodeAssignments(t, rr.Body.Bytes())
	if len(applied) != 2 {
		t.Fatalf("expected 2 applied moves, got %d", len(applied))
	}
	if applied[1].PayPeriodID != 30 {
		t.Errorf("expected the chained move to land on period 30, got %d", applied[1].PayPeriodID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestOptimizerApply_RejectsEmptyBatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	// No transaction should even be opened.

	rr := applyMoves(t, mock, `{"moves":[]}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Sinking fund Apply
// ---------------------------------------------------------------------------

func expectSinkingFundPlanQueries(mock pgxmock.PgxPoolIface, billAmount float64, periods [][]any) {
	mock.ExpectQuery("SELECT COALESCE\\(default_amount, 0\\) FROM bills").
		WithArgs(1).
		WillReturnRows(pgxmock.NewRows([]string{"default_amount"}).AddRow(billAmount))
	mock.ExpectQuery("SELECT pay_date FROM pay_periods").
		WithArgs(99).
		WillReturnRows(pgxmock.NewRows([]string{"pay_date"}).
			AddRow(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)))

	rows := pgxmock.NewRows([]string{"id", "pay_date", "income", "assigned"})
	for _, p := range periods {
		rows = rows.AddRow(p...)
	}
	mock.ExpectQuery("SELECT pp.id").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(rows)
}

func sinkingFundApply(t *testing.T, mock pgxmock.PgxPoolIface) *httptest.ResponseRecorder {
	t.Helper()
	h := NewSinkingFundHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bills/1/sinking-fund/apply",
		bytes.NewBufferString(`{"target_period_id":99,"num_periods":2}`))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(withChiContext(req.Context(), rctx))
	rr := httptest.NewRecorder()
	h.Apply(rr, req)
	return rr
}

// TestSinkingFundApply_SkipsPeriodHoldingOrdinaryAssignment is the data-loss
// fix: the upsert's DO UPDATE used to rewrite the bill's ordinary assignment in
// that period into a sinking-fund installment, which Clear() (which deletes
// WHERE is_sinking_fund = true) then removed outright. The narrowed DO UPDATE
// matches no row in that case, surfacing as pgx.ErrNoRows, which the handler
// must treat as "skip this period" rather than an error.
func TestSinkingFundApply_SkipsPeriodHoldingOrdinaryAssignment(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectSinkingFundPlanQueries(mock, 200, [][]any{
		{11, "2026-05-15", 1000.0, 0.0},
		{10, "2026-05-01", 1000.0, 0.0},
	})

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(1, 99).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	// Period 10 already holds an ordinary assignment for this bill: the guarded
	// upsert matches nothing and returns no rows.
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, 100.0, 99).
		WillReturnRows(pgxmock.NewRows([]string{}))

	// Period 11 is free and gets its installment.
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 11, 100.0, 99).
		WillReturnRows(oneAssignmentRow(70, 1, 11, 100.0))

	mock.ExpectCommit()

	rr := sinkingFundApply(t, mock)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	created := decodeAssignments(t, rr.Body.Bytes())
	if len(created) != 1 {
		t.Fatalf("expected 1 installment (the other period was skipped), got %d", len(created))
	}
	if created[0].PayPeriodID != 11 {
		t.Errorf("expected the installment on period 11, got %d", created[0].PayPeriodID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestSinkingFundApply_WritesInstallmentsForFreePeriods(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	expectSinkingFundPlanQueries(mock, 200, [][]any{
		{11, "2026-05-15", 1000.0, 0.0},
		{10, "2026-05-01", 1000.0, 0.0},
	})

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM bill_assignments").
		WithArgs(1, 99).
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 10, 100.0, 99).
		WillReturnRows(oneAssignmentRow(70, 1, 10, 100.0))
	mock.ExpectQuery("INSERT INTO bill_assignments").
		WithArgs(1, 11, 100.0, 99).
		WillReturnRows(oneAssignmentRow(71, 1, 11, 100.0))
	mock.ExpectCommit()

	rr := sinkingFundApply(t, mock)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if got := len(decodeAssignments(t, rr.Body.Bytes())); got != 2 {
		t.Errorf("expected 2 installments, got %d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Dashboard: 7-day upcoming window
// ---------------------------------------------------------------------------

// TestUpcomingDueDays_CrossesMonthBoundary pins the month-end bug: the handler
// computed `due_day BETWEEN day AND day+7`, so on the 28th it searched days
// 28-35 and matched nothing due early next month, emptying the panel.
func TestUpcomingDueDays_CrossesMonthBoundary(t *testing.T) {
	// 2026-01-28 + 7 days spans Jan 28-31 and Feb 1-4.
	got := upcomingDueDays(time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC), 7)

	want := map[int]bool{28: true, 29: true, 30: true, 31: true, 1: true, 2: true, 3: true, 4: true}
	if len(got) != len(want) {
		t.Fatalf("expected %d days, got %d: %v", len(want), len(got), got)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected day %d in %v", d, got)
		}
	}
}

func TestUpcomingDueDays_MidMonthIsContiguous(t *testing.T) {
	got := upcomingDueDays(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), 7)

	if len(got) != 8 {
		t.Fatalf("expected 8 days, got %d: %v", len(got), got)
	}
	for i, d := range got {
		if d != 10+i {
			t.Errorf("expected day %d at index %d, got %d", 10+i, i, d)
		}
	}
}

// TestUpcomingDueDays_ShortFebruaryDeduplicates confirms no day is repeated
// when a window wraps a short month.
func TestUpcomingDueDays_ShortFebruaryDeduplicates(t *testing.T) {
	got := upcomingDueDays(time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC), 7)

	seen := map[int]bool{}
	for _, d := range got {
		if seen[d] {
			t.Errorf("day %d appears more than once in %v", d, got)
		}
		seen[d] = true
	}
}
