package models

import "time"

type PayPeriod struct {
	ID             int       `json:"id"`
	IncomeSourceID int       `json:"income_source_id"`
	PayDate        time.Time `json:"pay_date"`
	ExpectedAmount *float64  `json:"expected_amount"`
	ActualAmount   *float64  `json:"actual_amount"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`

	// Computed fields (not stored)
	SourceName     string  `json:"source_name,omitempty"`
	TotalBills     float64 `json:"total_bills"`
	Remaining      float64 `json:"remaining"`
}

type GeneratePeriodsRequest struct {
	From      string `json:"from"`       // YYYY-MM-DD
	To        string `json:"to"`         // YYYY-MM-DD
	SourceIDs []int  `json:"source_ids"` // empty = all active sources
}
