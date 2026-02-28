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

import "net/http"

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
	accessToken := r.PathValue("accessToken")
	_ = accessToken // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/google
func HandleGetAuthGoogle(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/google/callback
func HandleGetAuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/oidc
func HandleGetAuthOidc(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /auth/oidc/callback
func HandleGetAuthOidcCallback(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
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
