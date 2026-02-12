package models

import "time"

type CreditCard struct {
	ID           int       `json:"id"`
	BillID       int       `json:"bill_id"`
	CardLabel    string    `json:"card_label"`
	StatementDay int       `json:"statement_day"`
	DueDay       int       `json:"due_day"`
	Issuer       string    `json:"issuer"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateCreditCardRequest struct {
	CardLabel    string `json:"card_label"`
	StatementDay int    `json:"statement_day"`
	DueDay       int    `json:"due_day"`
	Issuer       string `json:"issuer"`
}
