package model

// Platform represents a brokerage or financial platform.
type Platform struct {
	ID   string  `db:"id" json:"id"`
	Name *string `db:"name" json:"name,omitempty"`
	Url  string  `db:"url" json:"url"`
}
