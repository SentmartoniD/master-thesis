package models

import "time"

type CodeAnalytics struct {
	Code        string     `json:"code"`
	TotalClicks int64      `json:"total_clicks"`
	Last24h     int64      `json:"last_24h"`
	LastClickAt *time.Time `json:"last_click_at,omitempty"`
}
