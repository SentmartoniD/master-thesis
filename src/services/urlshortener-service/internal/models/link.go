package models

import (
	"strings"
	"time"
)

type Link struct {
	ID int64

	CreatedAt time.Time

	TargetURL string
	// Unique
	Code      string
	IsActive  bool
	ExpiresAt *time.Time
}

type CreateLinkRequest struct {
	URL       string     `json:"url" binding:"required,url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type LinkResponse struct {
	Code      string    `json:"code"`
	ShortURL  string    `json:"short_url"`
	TargetURL string    `json:"target_url"`
	CreatedAt time.Time `json:"created_at"`
}

func (link *Link) ToLinkResponse(baseURL string) *LinkResponse {
	return &LinkResponse{
		Code:      link.Code,
		ShortURL:  strings.TrimSuffix(baseURL, "/") + "/" + link.Code,
		TargetURL: link.TargetURL,
		CreatedAt: link.CreatedAt,
	}
}
