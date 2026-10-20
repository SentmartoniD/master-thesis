package models

import "time"

type Click struct {
	ID         int64
	Code       string
	OccurredAt time.Time
	Referrer   string
	UserAgent  string
}

func (event ClickEvent) ToClick() Click {
	return Click{
		Code:       event.Code,
		OccurredAt: event.OccurredAt,
		Referrer:   event.Referrer,
		UserAgent:  event.UserAgent,
	}
}
