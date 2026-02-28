package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/user/user.controller.ts

Routes + request/response shape:
- DELETE /api/v1/user (HandleDeleteUser)
  Request: body data: DeleteOwnUserDto
  Auth/permission: jwt / permissions.deleteOwnUser
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/user (HandleGetUser)
  Request: headers acceptLanguage: string (arg: 'accept-language')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/user (HandleCreateUser)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/user/{id} (HandleDeleteUserById)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteUser
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/user/{id}/access-token (HandleCreateUserAccessTokenById)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/user/access-token (HandleCreateUserAccessToken)
  Request: body data: UpdateOwnAccessTokenDto
  Auth/permission: jwt / permissions.updateOwnAccessToken
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PUT /api/v1/user/setting (HandleUpdateUserSetting)
  Request: body data: UpdateUserSettingDto
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

// DELETE /user
func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /user
func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("Accept-Language")
	_ = lang // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /user
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}

// DELETE /user/:id
func HandleDeleteUserById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /user/:id/access-token
func HandleCreateUserAccessTokenById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// POST /user/access-token
func HandleCreateUserAccessToken(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// PUT /user/setting
func HandleUpdateUserSetting(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
