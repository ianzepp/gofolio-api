package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/ai/ai.controller.ts

Routes + request/response shape:
- GET /api/v1/ai/prompt/{mode} (HandleGetAiPrompt)
  Request: path mode: AiPromptMode (arg: 'mode')
  Auth/permission: jwt / permissions.readAiPrompt
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetAiPrompt(w http.ResponseWriter, r *http.Request) {
	mode := r.PathValue("mode")
	_ = mode
	JSON(w, http.StatusOK, map[string]any{})
}
