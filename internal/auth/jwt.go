package auth

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ParseJWT validates an HS256-signed JWT and returns the user ID from the "id" claim.
func ParseJWT(tokenStr string, secretKey string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
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
func SignJWT(userID string, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": userID,
	})
	return token.SignedString([]byte(secretKey))
}

// ExtractBearerToken strips the "Bearer " prefix from an Authorization header value.
func ExtractBearerToken(authHeader string) (string, bool) {
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		return authHeader[7:], true
	}
	return "", false
}
