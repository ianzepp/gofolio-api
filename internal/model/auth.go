package model

import "time"

// AuthDevice stores a WebAuthn (FIDO2) credential registered by a user for
// passwordless hardware-key authentication. Multiple devices can be registered
// per user (e.g. a YubiKey and a Touch ID laptop).
//
// CredentialId is the raw bytes returned by the authenticator during registration
// and used as the lookup key during assertion. CredentialPublicKey is the COSE-
// encoded public key stored for signature verification.
//
// Counter is the monotonically increasing signature counter reported by the
// authenticator on each use. The server must reject assertions where the received
// counter is not greater than the stored value to detect cloned authenticators.
// SAFETY: never reset Counter to zero after a successful assertion.
type AuthDevice struct {
	ID                  string    `db:"id" json:"id"`
	CreatedAt           time.Time `db:"createdAt" json:"createdAt"`
	CredentialId        []byte    `db:"credentialId" json:"credentialId"`
	CredentialPublicKey []byte    `db:"credentialPublicKey" json:"credentialPublicKey"`
	Counter             int       `db:"counter" json:"counter"`
	UpdatedAt           time.Time `db:"updatedAt" json:"updatedAt"`
	UserId              string    `db:"userId" json:"userId"`
}

// ApiKey stores a bcrypt-hashed API key that grants programmatic access to
// the Ghostfolio API on behalf of a user. The plaintext key is shown to the
// user only once at creation time and never stored.
//
// SAFETY: HashedKey must never be returned in API responses. Validation is done
// by comparing a submitted key against the stored hash, not by decryption.
type ApiKey struct {
	ID        string    `db:"id" json:"id"`
	CreatedAt time.Time `db:"createdAt" json:"createdAt"`
	HashedKey string    `db:"hashedKey" json:"-"`
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
	UserId    string    `db:"userId" json:"userId"`
}
