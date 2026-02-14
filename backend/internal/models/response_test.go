package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// WriteJSON tests
// ---------------------------------------------------------------------------

func TestWriteJSON_SetsContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, "hello")

	got := w.Header().Get("Content-Type")
	if got != "application/json" {
		t.Errorf("Content-Type = %q; want %q", got, "application/json")
	}
}

func TestWriteJSON_WritesCorrectStatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"202 Accepted", http.StatusAccepted},
		{"204 No Content", http.StatusNoContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteJSON(w, tc.status, nil)

			if w.Code != tc.status {
				t.Errorf("status code = %d; want %d", w.Code, tc.status)
			}
		})
	}
}

func TestWriteJSON_WrapsDataInEnvelope(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, "test-value")

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	// Data should be present.
	if resp.Data == nil {
		t.Fatal("expected Data to be non-nil")
	}
	if resp.Data != "test-value" {
		t.Errorf("Data = %v; want %q", resp.Data, "test-value")
	}

	// Meta with a timestamp should be present.
	if resp.Meta == nil {
		t.Fatal("expected Meta to be non-nil")
	}
	if resp.Meta.Timestamp.IsZero() {
		t.Error("expected Meta.Timestamp to be non-zero")
	}
}

func TestWriteJSON_MetaTimestampIsRecentUTC(t *testing.T) {
	before := time.Now().UTC()
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, nil)

	after := time.Now().UTC()

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	ts := resp.Meta.Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Meta.Timestamp %v is not between %v and %v", ts, before, after)
	}

	// Verify the timestamp is in UTC.
	if ts.Location() != time.UTC {
		t.Errorf("Meta.Timestamp location = %v; want UTC", ts.Location())
	}
}

func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, nil)

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// The "data" key should be present and its value should be null.
	dataRaw, ok := raw["data"]
	if !ok {
		t.Fatal("expected 'data' key in response JSON")
	}
	if string(dataRaw) != "null" {
		t.Errorf("data = %s; want null", string(dataRaw))
	}

	// meta should still be present.
	if _, ok := raw["meta"]; !ok {
		t.Fatal("expected 'meta' key in response JSON")
	}
}

type testItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestWriteJSON_StructData(t *testing.T) {
	w := httptest.NewRecorder()
	input := testItem{ID: 42, Name: "widget"}

	WriteJSON(w, http.StatusOK, input)

	// Decode into a generic structure to inspect the envelope.
	var envelope struct {
		Data testItem `json:"data"`
		Meta *Meta    `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if envelope.Data.ID != 42 {
		t.Errorf("Data.ID = %d; want 42", envelope.Data.ID)
	}
	if envelope.Data.Name != "widget" {
		t.Errorf("Data.Name = %q; want %q", envelope.Data.Name, "widget")
	}
	if envelope.Meta == nil {
		t.Fatal("expected Meta to be non-nil")
	}
}

func TestWriteJSON_SliceData(t *testing.T) {
	w := httptest.NewRecorder()
	items := []testItem{
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "beta"},
		{ID: 3, Name: "gamma"},
	}

	WriteJSON(w, http.StatusCreated, items)

	var envelope struct {
		Data []testItem `json:"data"`
		Meta *Meta      `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(envelope.Data) != 3 {
		t.Fatalf("len(Data) = %d; want 3", len(envelope.Data))
	}
	if envelope.Data[0].Name != "alpha" {
		t.Errorf("Data[0].Name = %q; want %q", envelope.Data[0].Name, "alpha")
	}
	if envelope.Data[2].ID != 3 {
		t.Errorf("Data[2].ID = %d; want 3", envelope.Data[2].ID)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status code = %d; want %d", w.Code, http.StatusCreated)
	}
	if envelope.Meta == nil {
		t.Fatal("expected Meta to be non-nil")
	}
}

func TestWriteJSON_EmptySliceData(t *testing.T) {
	w := httptest.NewRecorder()
	items := []testItem{}

	WriteJSON(w, http.StatusOK, items)

	var envelope struct {
		Data []testItem `json:"data"`
		Meta *Meta      `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if envelope.Data == nil {
		t.Fatal("expected Data to be non-nil empty slice")
	}
	if len(envelope.Data) != 0 {
		t.Errorf("len(Data) = %d; want 0", len(envelope.Data))
	}
}

func TestWriteJSON_MapData(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]interface{}{
		"count":  3,
		"active": true,
	}

	WriteJSON(w, http.StatusOK, data)

	var envelope struct {
		Data map[string]interface{} `json:"data"`
		Meta *Meta                  `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// JSON numbers decode as float64.
	if envelope.Data["count"] != float64(3) {
		t.Errorf("Data[count] = %v; want 3", envelope.Data["count"])
	}
	if envelope.Data["active"] != true {
		t.Errorf("Data[active] = %v; want true", envelope.Data["active"])
	}
}

func TestWriteJSON_NestedStructData(t *testing.T) {
	type inner struct {
		Value string `json:"value"`
	}
	type outer struct {
		Inner inner `json:"inner"`
	}

	w := httptest.NewRecorder()
	data := outer{Inner: inner{Value: "deep"}}

	WriteJSON(w, http.StatusOK, data)

	var envelope struct {
		Data outer `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if envelope.Data.Inner.Value != "deep" {
		t.Errorf("Data.Inner.Value = %q; want %q", envelope.Data.Inner.Value, "deep")
	}
}

func TestWriteJSON_NumericData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, 12345)

	var envelope struct {
		Data float64 `json:"data"` // JSON numbers decode as float64
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if envelope.Data != 12345 {
		t.Errorf("Data = %v; want 12345", envelope.Data)
	}
}

func TestWriteJSON_BooleanData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, true)

	var envelope struct {
		Data bool `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !envelope.Data {
		t.Error("Data = false; want true")
	}
}

func TestWriteJSON_ResponseIsValidJSON(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, "anything")

	var raw json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestWriteJSON_ResponseContainsOnlyDataAndMeta(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, "value")

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := raw["data"]; !ok {
		t.Error("expected 'data' key in response")
	}
	if _, ok := raw["meta"]; !ok {
		t.Error("expected 'meta' key in response")
	}
	if len(raw) != 2 {
		t.Errorf("response has %d top-level keys; want 2 (data, meta)", len(raw))
	}
}

// ---------------------------------------------------------------------------
// WriteError tests
// ---------------------------------------------------------------------------

func TestWriteError_SetsContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid input")

	got := w.Header().Get("Content-Type")
	if got != "application/json" {
		t.Errorf("Content-Type = %q; want %q", got, "application/json")
	}
}

func TestWriteError_WritesCorrectStatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
		{"404 Not Found", http.StatusNotFound},
		{"409 Conflict", http.StatusConflict},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteError(w, tc.status, "CODE", "message")

			if w.Code != tc.status {
				t.Errorf("status code = %d; want %d", w.Code, tc.status)
			}
		})
	}
}

func TestWriteError_FormatsErrorEnvelope(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")

	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("Error.Code = %q; want %q", resp.Error.Code, "NOT_FOUND")
	}
	if resp.Error.Message != "resource not found" {
		t.Errorf("Error.Message = %q; want %q", resp.Error.Message, "resource not found")
	}
}

func TestWriteError_DetailsOmittedWhenNil(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "bad")

	// Decode into a raw map so we can inspect which keys are present.
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// The top-level should only have "error".
	if _, ok := raw["error"]; !ok {
		t.Fatal("expected 'error' key in response")
	}

	// Decode the error object itself.
	var errDetail map[string]json.RawMessage
	if err := json.Unmarshal(raw["error"], &errDetail); err != nil {
		t.Fatalf("failed to decode error detail: %v", err)
	}

	// "details" should be omitted (omitempty tag on ErrorDetail.Details).
	if _, ok := errDetail["details"]; ok {
		t.Error("expected 'details' key to be omitted when nil")
	}
}

func TestWriteError_ResponseContainsOnlyErrorKey(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusInternalServerError, "INTERNAL", "something broke")

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(raw) != 1 {
		t.Errorf("response has %d top-level keys; want 1 (error)", len(raw))
	}
	if _, ok := raw["error"]; !ok {
		t.Error("expected 'error' key in response")
	}
}

func TestWriteError_VariousErrorCodes(t *testing.T) {
	tests := []struct {
		code    string
		message string
	}{
		{"VALIDATION_ERROR", "field 'email' is required"},
		{"NOT_FOUND", "user with id 123 not found"},
		{"UNAUTHORIZED", "invalid or expired token"},
		{"FORBIDDEN", "insufficient permissions"},
		{"CONFLICT", "resource already exists"},
		{"INTERNAL_ERROR", "an unexpected error occurred"},
		{"RATE_LIMITED", "too many requests, please try again later"},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			w := httptest.NewRecorder()

			WriteError(w, http.StatusBadRequest, tc.code, tc.message)

			var resp APIError
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Error.Code != tc.code {
				t.Errorf("Error.Code = %q; want %q", resp.Error.Code, tc.code)
			}
			if resp.Error.Message != tc.message {
				t.Errorf("Error.Message = %q; want %q", resp.Error.Message, tc.message)
			}
		})
	}
}

func TestWriteError_EmptyCodeAndMessage(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusInternalServerError, "", "")

	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Code != "" {
		t.Errorf("Error.Code = %q; want empty string", resp.Error.Code)
	}
	if resp.Error.Message != "" {
		t.Errorf("Error.Message = %q; want empty string", resp.Error.Message)
	}
}

func TestWriteError_SpecialCharactersInMessage(t *testing.T) {
	w := httptest.NewRecorder()
	msg := `field "name" contains <invalid> chars & symbols`

	WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)

	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error.Message != msg {
		t.Errorf("Error.Message = %q; want %q", resp.Error.Message, msg)
	}
}

func TestWriteError_IsValidJSON(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "CODE", "msg")

	var raw json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestWriteError_ErrorDetailHasCodeAndMessage(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "TEST_CODE", "test message")

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var errDetail map[string]json.RawMessage
	if err := json.Unmarshal(raw["error"], &errDetail); err != nil {
		t.Fatalf("failed to decode error object: %v", err)
	}

	if _, ok := errDetail["code"]; !ok {
		t.Error("expected 'code' key in error object")
	}
	if _, ok := errDetail["message"]; !ok {
		t.Error("expected 'message' key in error object")
	}
}

// ---------------------------------------------------------------------------
// Struct serialization tests (APIResponse, APIError, Meta, ErrorDetail)
// ---------------------------------------------------------------------------

func TestAPIResponse_JSONSerialization(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	resp := APIResponse{
		Data: "hello",
		Meta: &Meta{Timestamp: ts},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal APIResponse: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := decoded["data"]; !ok {
		t.Error("expected 'data' key")
	}
	if _, ok := decoded["meta"]; !ok {
		t.Error("expected 'meta' key")
	}
}

func TestAPIResponse_MetaOmittedWhenNil(t *testing.T) {
	resp := APIResponse{
		Data: "hello",
		Meta: nil,
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := decoded["meta"]; ok {
		t.Error("expected 'meta' key to be omitted when Meta is nil")
	}
}

func TestAPIError_JSONSerialization(t *testing.T) {
	apiErr := APIError{
		Error: ErrorDetail{
			Code:    "NOT_FOUND",
			Message: "not found",
		},
	}

	b, err := json.Marshal(apiErr)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded APIError
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Error.Code != "NOT_FOUND" {
		t.Errorf("Code = %q; want %q", decoded.Error.Code, "NOT_FOUND")
	}
	if decoded.Error.Message != "not found" {
		t.Errorf("Message = %q; want %q", decoded.Error.Message, "not found")
	}
}

func TestErrorDetail_DetailsOmittedWhenNil(t *testing.T) {
	detail := ErrorDetail{
		Code:    "ERR",
		Message: "msg",
		Details: nil,
	}

	b, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := decoded["details"]; ok {
		t.Error("expected 'details' to be omitted when nil")
	}
}

func TestErrorDetail_DetailsIncludedWhenPresent(t *testing.T) {
	detail := ErrorDetail{
		Code:    "VALIDATION_ERROR",
		Message: "validation failed",
		Details: map[string]string{"field": "email"},
	}

	b, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := decoded["details"]; !ok {
		t.Error("expected 'details' key to be present when Details is non-nil")
	}
}
