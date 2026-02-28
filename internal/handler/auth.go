// Package handler — auth.go implements handlers for /api/v1/auth/*.
//
// Most routes in this file are stubs that return empty success responses — they
// satisfy the API contract while the OAuth/WebAuthn service layer is ported from
// NestJS. The two meaningful flows are:
//
//   - Anonymous login: POST /auth/anonymous accepts an access token in the body
//     and GET /auth/anonymous/:token accepts it in the path. Both variants exist
//     because early Ghostfolio clients used the GET form.
//
//   - OAuth redirects: Google and OIDC routes redirect to the configured OAuth
//     provider using ROOT_URL. The callback routes redirect back to the Angular
//     SPA with the JWT appended to the path.
//
// OIDC is feature-flag gated via the ENABLE_FEATURE_AUTH_OIDC env var. Attempting
// the OIDC flow when disabled returns 403 rather than 404 to match NestJS.
//
// Parity reference: apps/api/src/app/auth/auth.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/auth/auth.controller.ts

Routes + request/response shape:
- POST /api/v1/auth/anonymous (HandleAuthAnonymous)
  Request: body body: { accessToken: string }
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/anonymous/{accessToken} (HandleGetAuthAnonymous)
  Request: path accessToken: string (arg: 'accessToken')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/google (HandleGetAuthGoogle)
  Request: no explicit request params
  Auth/permission: oauth-google / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/google/callback (HandleGetAuthGoogleCallback)
  Request: no explicit request params
  Auth/permission: oauth-google / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/oidc (HandleGetAuthOidc)
  Request: no explicit request params
  Auth/permission: oauth-oidc / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/oidc/callback (HandleGetAuthOidcCallback)
  Request: no explicit request params
  Auth/permission: oauth-oidc / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/auth/webauthn/generate-authentication-options (HandleWebauthnGenerateAuthenticationOptions)
  Request: body body: { deviceId: string }
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/auth/webauthn/generate-registration-options (HandleGetWebauthnGenerateRegistrationOptions)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/auth/webauthn/verify-attestation (HandleWebauthnVerifyAttestation)
  Request: body body: { deviceName: string; credential: AttestationCredentialJSON }
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/auth/webauthn/verify-authentication (HandleWebauthnVerifyAuthentication)
  Request: body body: { deviceId: string; credential: AssertionCredentialJSON }
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import (
	"net/http"
	"os"
)

// POST /auth/anonymous
func HandleAuthAnonymous(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// GET /auth/anonymous/:accessToken
func HandleGetAuthAnonymous(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// The NestJS API supports access-token login via BOTH POST body and GET path.
	// Keep this path variant for contract compatibility.
	accessToken := r.PathValue("accessToken")
	_ = accessToken // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/google
func HandleGetAuthGoogle(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// Auth flows are redirect-based, not JSON APIs.
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
	// Legacy behavior note:
	// OIDC endpoint is feature-flag gated and returns 403 when disabled.
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

// POST /auth/webauthn/generate-authentication-options
func HandleWebauthnGenerateAuthenticationOptions(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/webauthn/generate-registration-options
func HandleGetWebauthnGenerateRegistrationOptions(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /auth/webauthn/verify-attestation
func HandleWebauthnVerifyAttestation(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// POST /auth/webauthn/verify-authentication
func HandleWebauthnVerifyAuthentication(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

const defaultLanguageCode = "en"

// rootURL returns the configured frontend root URL, falling back to the
// default localhost Angular dev server. Used to construct OAuth redirect targets.
func rootURL() string {
	if value := os.Getenv("ROOT_URL"); value != "" {
		return value
	}

	return "http://localhost:4200"
}
