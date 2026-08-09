package domain

import "time"

type Link struct {
	Code      string     `json:"code"`
	TargetURL string     `json:"targetUrl"`
	ShortURL  string     `json:"shortUrl"`
	Clicks    int64      `json:"clicks"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	Expired   bool       `json:"expired"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (l Link) IsExpired(now time.Time) bool {
	return l.DeletedAt != nil || (l.ExpiresAt != nil && !l.ExpiresAt.After(now))
}

type ClickEvent struct {
	Code      string    `json:"code"`
	TargetURL string    `json:"targetUrl"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Referer   string    `json:"referer"`
	ClickedAt time.Time `json:"clickedAt"`
}
