// Package handler — auth.go implements handlers for /api/v1/auth/*.
//
// Implemented flows:
//   - Anonymous login: POST /auth/anonymous accepts an access token in the body
//     and GET /auth/anonymous/:token accepts it in the path. Both hash the token
//     with HMAC-SHA512(ACCESS_TOKEN_SALT), look up the user, and return a signed JWT.
//   - OAuth redirects: Google and OIDC routes redirect to the configured OAuth
//     provider. Callback routes redirect back to the Angular SPA with the JWT.
//   - WebAuthn: Stubs only — passkey/FIDO2 flows are not yet ported.
//
// OIDC is feature-flag gated via the ENABLE_FEATURE_AUTH_OIDC env var.
//
// Parity reference: apps/api/src/app/auth/auth.controller.ts
package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"os"

	"github.com/ianzepp/gofolio-api/internal/auth"
)

// AuthAnonymous handles POST /api/v1/auth/anonymous.
// Accepts {"accessToken": "..."}, hashes it, looks up the user, and returns
// a signed JWT as {"authToken": "..."}.
func AuthAnonymous(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if !Bind(w, r, &body) {
			return
		}

		authToken, err := validateAnonymousLogin(r, deps, body.AccessToken)
		if err != nil {
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		JSON(w, http.StatusCreated, map[string]any{
			"authToken": authToken,
		})
	}
}

// GetAuthAnonymous handles GET /api/v1/auth/anonymous/{accessToken}.
// Deprecated path-based variant of the anonymous login flow.
func GetAuthAnonymous(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		accessToken := r.PathValue("accessToken")

		authToken, err := validateAnonymousLogin(r, deps, accessToken)
		if err != nil {
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		JSON(w, http.StatusOK, map[string]any{
			"authToken": authToken,
		})
	}
}

// validateAnonymousLogin hashes the access token, looks up the user, and
// returns a signed JWT. Returns an error if the token is invalid.
func validateAnonymousLogin(r *http.Request, deps *Deps, accessToken string) (string, error) {
	hashed, err := hashAccessToken(accessToken)
	if err != nil {
		return "", err
	}

	user, err := deps.UserService.GetUserByAccessToken(r.Context(), hashed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("invalid access token")
		}
		return "", err
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	return auth.SignJWT(user.ID, jwtSecret)
}

// GET /auth/google
func HandleGetAuthGoogle(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, rootURL()+"/oauth/google", http.StatusFound)
}

// GET /auth/google/callback
func HandleGetAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	jwt := r.URL.Query().Get("jwt")
	if jwt != "" {
		http.Redirect(w, r, rootURL()+"/"+defaultLanguageCode+"/auth/"+jwt, http.StatusFound)
		return
	}
	http.Redirect(w, r, rootURL()+"/"+defaultLanguageCode+"/auth", http.StatusFound)
}

// GET /auth/oidc
func HandleGetAuthOidc(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENABLE_FEATURE_AUTH_OIDC") != "true" {
		Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
		return
	}
	http.Redirect(w, r, rootURL()+"/oauth/oidc", http.StatusFound)
}

// GET /auth/oidc/callback
func HandleGetAuthOidcCallback(w http.ResponseWriter, r *http.Request) {
	jwt := r.URL.Query().Get("jwt")
	if jwt != "" {
		http.Redirect(w, r, rootURL()+"/"+defaultLanguageCode+"/auth/"+jwt, http.StatusFound)
		return
	}
	http.Redirect(w, r, rootURL()+"/"+defaultLanguageCode+"/auth", http.StatusFound)
}

// POST /auth/webauthn/generate-authentication-options — stub
func HandleWebauthnGenerateAuthenticationOptions(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/webauthn/generate-registration-options — stub
func HandleGetWebauthnGenerateRegistrationOptions(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /auth/webauthn/verify-attestation — stub
func HandleWebauthnVerifyAttestation(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	JSON(w, http.StatusCreated, map[string]any{})
}

// POST /auth/webauthn/verify-authentication — stub
func HandleWebauthnVerifyAuthentication(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	JSON(w, http.StatusOK, map[string]any{})
}

const defaultLanguageCode = "en"

func rootURL() string {
	if value := os.Getenv("ROOT_URL"); value != "" {
		return value
	}
	return "http://localhost:4200"
}
