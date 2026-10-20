package models

import "time"

type ClickEvent struct {
	Code       string    `json:"code"`
	OccurredAt time.Time `json:"occurred_at"`
	Referrer   string    `json:"referrer,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
}
