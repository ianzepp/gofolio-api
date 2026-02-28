package model

import "time"

// AuthDevice stores WebAuthn credentials for a user.
type AuthDevice struct {
	ID                  string    `db:"id" json:"id"`
	CreatedAt           time.Time `db:"createdAt" json:"createdAt"`
	CredentialId        []byte    `db:"credentialId" json:"credentialId"`
	CredentialPublicKey []byte    `db:"credentialPublicKey" json:"credentialPublicKey"`
	Counter             int       `db:"counter" json:"counter"`
	UpdatedAt           time.Time `db:"updatedAt" json:"updatedAt"`
	UserId              string    `db:"userId" json:"userId"`
}

// ApiKey stores a hashed API key for a user.
type ApiKey struct {
	ID        string    `db:"id" json:"id"`
	CreatedAt time.Time `db:"createdAt" json:"createdAt"`
	HashedKey string    `db:"hashedKey" json:"hashedKey"`
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
	UserId    string    `db:"userId" json:"userId"`
}
