package models

import "time"

type BillAssignment struct {
	ID              int       `json:"id"`
	BillID          int       `json:"bill_id"`
	PayPeriodID     int       `json:"pay_period_id"`
	PlannedAmount   *float64  `json:"planned_amount"`
	ForecastAmount  *float64  `json:"forecast_amount"`
	ActualAmount    *float64  `json:"actual_amount"`
	Status          string    `json:"status"` // pending, paid, deferred, uncertain, skipped
	DeferredToID    *int      `json:"deferred_to_id"`
	IsExtra         bool      `json:"is_extra"`
	ExtraName       string    `json:"extra_name,omitempty"`
	Notes           string    `json:"notes"`
	ManuallyMoved   bool      `json:"manually_moved"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Joined fields
	BillName        string `json:"bill_name,omitempty"`
}

type CreateAssignmentRequest struct {
	BillID         int      `json:"bill_id"`
	PayPeriodID    int      `json:"pay_period_id"`
	PlannedAmount  *float64 `json:"planned_amount"`
	ForecastAmount *float64 `json:"forecast_amount"`
	ActualAmount   *float64 `json:"actual_amount"`
	Status         string   `json:"status"`
	IsExtra        bool     `json:"is_extra"`
	ExtraName      string   `json:"extra_name"`
	Notes          string   `json:"notes"`
}

type UpdateAssignmentRequest struct {
	PlannedAmount  *float64 `json:"planned_amount,omitempty"`
	ForecastAmount *float64 `json:"forecast_amount,omitempty"`
	ActualAmount   *float64 `json:"actual_amount,omitempty"`
	Status         *string  `json:"status,omitempty"`
	DeferredToID   *int     `json:"deferred_to_id,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
}

type UpdateStatusRequest struct {
	Status         string `json:"status"`
	DeferredToID   *int   `json:"deferred_to_id,omitempty"`
}
