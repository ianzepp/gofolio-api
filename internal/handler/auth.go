package handler

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
