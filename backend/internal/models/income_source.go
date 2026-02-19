package models

import (
	"encoding/json"
	"time"
)

type IncomeSource struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	PaySchedule    string          `json:"pay_schedule"`
	ScheduleDetail json.RawMessage `json:"schedule_detail"`
	DefaultAmount  *float64        `json:"default_amount"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// WeeklySchedule is used when PaySchedule == "weekly"
type WeeklySchedule struct {
	Weekday int `json:"weekday"` // 0=Sunday, 5=Friday, etc.
}

// BiweeklySchedule is used when PaySchedule == "biweekly"
type BiweeklySchedule struct {
	Weekday    int    `json:"weekday"`
	AnchorDate string `json:"anchor_date"` // a known pay date to anchor the biweekly cycle
}

// SemiMonthlySchedule is used when PaySchedule == "semimonthly"
type SemiMonthlySchedule struct {
	Days              []int `json:"days"`                // e.g. [1, 16]
	AdjustForWeekends bool  `json:"adjust_for_weekends"` // if true, move weekend dates to preceding Friday
}

// OneTimeSchedule is used when PaySchedule == "one_time" (e.g. bonus)
type OneTimeSchedule struct {
	Date string `json:"date"` // YYYY-MM-DD
}

type CreateIncomeSourceRequest struct {
	Name           string          `json:"name"`
	PaySchedule    string          `json:"pay_schedule"`
	ScheduleDetail json.RawMessage `json:"schedule_detail"`
	DefaultAmount  *float64        `json:"default_amount"`
}

type UpdateIncomeSourceRequest struct {
	Name           *string          `json:"name,omitempty"`
	PaySchedule    *string          `json:"pay_schedule,omitempty"`
	ScheduleDetail json.RawMessage  `json:"schedule_detail,omitempty"`
	DefaultAmount  *float64         `json:"default_amount,omitempty"`
	IsActive       *bool            `json:"is_active,omitempty"`
}
