package model

// Property is a key-value store for application settings.
// PK: key
type Property struct {
	Key   string `db:"key" json:"key"`
	Value string `db:"value" json:"value"`
}
