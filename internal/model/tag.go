package model

// Tag categorizes orders/activities.
// Unique: (name, userId)
type Tag struct {
	ID     string  `db:"id" json:"id"`
	Name   string  `db:"name" json:"name"`
	UserId *string `db:"userId" json:"userId,omitempty"`
}
