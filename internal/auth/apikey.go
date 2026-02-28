// apikey.go handles API key extraction and hashing. API keys are a long-lived
// credential alternative to JWT tokens, intended for programmatic access where
// storing a session token is impractical (e.g., scripts, CI pipelines).
//
// Keys are never stored in plaintext. The database holds only the PBKDF2 hash,
// so a database leak does not expose usable keys. The hash parameters (100k
// iterations, SHA-256, empty salt) are fixed to match the NestJS ApiKeyService;
// changing them would invalidate all existing keys stored in the database.
//
// TRADE-OFF: An empty PBKDF2 salt means two users with the same key would
// produce the same hash. In practice, API keys are long random strings, so
// collision probability is negligible. The NestJS implementation made this
// choice and this port preserves it for compatibility.
package auth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ExtractApiKey strips the "Api-Key " prefix from an Authorization header value.
// Returns the raw key string and true on success. The prefix comparison is
// case-insensitive to tolerate client variations (e.g., "api-key " vs "Api-Key ").
func ExtractApiKey(authHeader string) (string, bool) {
	if len(authHeader) > 8 && strings.EqualFold(authHeader[:8], "Api-Key ") {
		return authHeader[8:], true
	}
	return "", false
}

// HashApiKey hashes an API key using PBKDF2 with SHA-256, matching the NestJS
// ApiKeyService implementation (100k iterations, 64-byte key, empty salt).
//
// The 100k iteration count provides brute-force resistance at the cost of ~10ms
// CPU per authentication. This overhead is acceptable for API key auth, which
// is less frequent than JWT validation. The resulting hex string is used as the
// lookup key in the ApiKey table.
//
// EDGE: Returns an error only if the underlying pbkdf2.Key call fails, which
// should not occur with SHA-256. The error path exists for defensive completeness.
func HashApiKey(apiKey string) (string, error) {
	dk, err := pbkdf2.Key(sha256.New, apiKey, []byte(""), 100000, 64)
	if err != nil {
		return "", fmt.Errorf("pbkdf2 key derivation failed: %w", err)
	}
	return hex.EncodeToString(dk), nil
}
