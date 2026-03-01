// jwt.go provides JWT creation and validation helpers that interoperate with
// the NestJS Ghostfolio API's token format. The NestJS server signs tokens
// using HS256 with a shared secret and encodes the user's UUID in the "id"
// claim. This file must stay in sync with that convention; changing the claim
// name or algorithm here will break cross-service token compatibility.
package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ParseJWT validates an HS256-signed JWT and returns the user ID from the "id" claim.
//
// The signing method is checked explicitly before returning the secret key to
// prevent algorithm-confusion attacks where an attacker could craft a token
// signed with "none" or RS256 and pass the HMAC verification with a public key.
// Returns an error if the token is expired, malformed, or lacks a non-empty "id" claim.
func ParseJWT(tokenStr string, secretKey string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		// SAFETY: Reject any algorithm other than HMAC variants to prevent
		// algorithm-confusion attacks (e.g., alg:none or RSA confusion).
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid claims")
	}

	id, ok := claims["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("missing id claim")
	}

	return id, nil
}

// SignJWT creates an HS256-signed JWT with the given user ID in the "id" claim.
// The "exp" claim is set to 180 days, matching the NestJS JwtModule configuration
// (signOptions: { expiresIn: '180 days' }). ParseJWT automatically enforces
// expiration via the golang-jwt library.
func SignJWT(userID string, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": time.Now().Add(180 * 24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secretKey))
}

// ExtractBearerToken strips the "Bearer " prefix from an Authorization header value.
// Returns the raw token string and true on success, or ("", false) if the header
// does not match the expected prefix. The prefix comparison is case-insensitive to
// tolerate common client variations (e.g., "bearer " vs "Bearer ").
func ExtractBearerToken(authHeader string) (string, bool) {
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		return authHeader[7:], true
	}
	return "", false
}
