package model

import "time"

// User represents a Ghostfolio user account.
type User struct {
	ID            string    `db:"id" json:"id"`
	AccessToken   *string   `db:"accessToken" json:"accessToken,omitempty"`
	AuthChallenge *string   `db:"authChallenge" json:"authChallenge,omitempty"`
	CreatedAt     time.Time `db:"createdAt" json:"createdAt"`
	Provider      Provider  `db:"provider" json:"provider"`
	Role          Role      `db:"role" json:"role"`
	ThirdPartyId  *string   `db:"thirdPartyId" json:"thirdPartyId,omitempty"`
	UpdatedAt     time.Time `db:"updatedAt" json:"updatedAt"`
}

// Settings stores per-user JSON settings.
// PK: userId
type Settings struct {
	UserId    string    `db:"userId" json:"userId"`
	Settings  *string   `db:"settings" json:"settings,omitempty"` // JSON string
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
}

// Analytics tracks per-user analytics.
// PK: userId
type Analytics struct {
	UserId                              string    `db:"userId" json:"userId"`
	ActivityCount                       int       `db:"activityCount" json:"activityCount"`
	Country                             *string   `db:"country" json:"country,omitempty"`
	DataProviderGhostfolioDailyRequests int       `db:"dataProviderGhostfolioDailyRequests" json:"dataProviderGhostfolioDailyRequests"`
	LastRequestAt                       time.Time `db:"lastRequestAt" json:"lastRequestAt"`
	UpdatedAt                           time.Time `db:"updatedAt" json:"updatedAt"`
}
