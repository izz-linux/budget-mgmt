package models

import (
	"encoding/json"
	"time"
)

type Bill struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	DefaultAmount    *float64         `json:"default_amount"`
	DueDay           *int             `json:"due_day"`
	Recurrence       string           `json:"recurrence"`
	RecurrenceDetail json.RawMessage  `json:"recurrence_detail,omitempty"`
	IsAutopay        bool             `json:"is_autopay"`
	Category         string           `json:"category"`
	Notes            string           `json:"notes"`
	IsActive         bool             `json:"is_active"`
	SortOrder        int              `json:"sort_order"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	CreditCard       *CreditCard      `json:"credit_card,omitempty"`
}

type CreateBillRequest struct {
	Name             string           `json:"name"`
	DefaultAmount    *float64         `json:"default_amount"`
	DueDay           *int             `json:"due_day"`
	Recurrence       string           `json:"recurrence"`
	RecurrenceDetail json.RawMessage  `json:"recurrence_detail,omitempty"`
	IsAutopay        bool             `json:"is_autopay"`
	Category         string           `json:"category"`
	Notes            string           `json:"notes"`
	SortOrder        int              `json:"sort_order"`
	CreditCard       *CreateCreditCardRequest `json:"credit_card,omitempty"`
}

type UpdateBillRequest struct {
	Name             *string          `json:"name,omitempty"`
	DefaultAmount    *float64         `json:"default_amount,omitempty"`
	DueDay           *int             `json:"due_day,omitempty"`
	Recurrence       *string          `json:"recurrence,omitempty"`
	RecurrenceDetail json.RawMessage  `json:"recurrence_detail,omitempty"`
	IsAutopay        *bool            `json:"is_autopay,omitempty"`
	Category         *string          `json:"category,omitempty"`
	Notes            *string          `json:"notes,omitempty"`
	IsActive         *bool            `json:"is_active,omitempty"`
	SortOrder        *int             `json:"sort_order,omitempty"`
}

type ReorderBillsRequest struct {
	Orders []BillOrder `json:"orders"`
}

type BillOrder struct {
	ID        int `json:"id"`
	SortOrder int `json:"sort_order"`
}
