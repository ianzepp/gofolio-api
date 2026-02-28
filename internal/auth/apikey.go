package auth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ExtractApiKey strips the "Api-Key " prefix from an Authorization header value.
func ExtractApiKey(authHeader string) (string, bool) {
	if len(authHeader) > 8 && strings.EqualFold(authHeader[:8], "Api-Key ") {
		return authHeader[8:], true
	}
	return "", false
}

// HashApiKey hashes an API key using PBKDF2 with SHA-256, matching the NestJS
// ApiKeyService implementation (100k iterations, 64-byte key, empty salt).
func HashApiKey(apiKey string) string {
	dk, err := pbkdf2.Key(sha256.New, apiKey, []byte(""), 100000, 64)
	if err != nil {
		// Should never happen with valid parameters.
		panic("pbkdf2.Key failed: " + err.Error())
	}
	return hex.EncodeToString(dk)
}
