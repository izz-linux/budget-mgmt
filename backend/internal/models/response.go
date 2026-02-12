package models

import (
	"encoding/json"
	"net/http"
	"time"
)

type APIResponse struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Timestamp time.Time `json:"timestamp"`
}

type APIError struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Data: data,
		Meta: &Meta{Timestamp: time.Now().UTC()},
	})
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIError{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
