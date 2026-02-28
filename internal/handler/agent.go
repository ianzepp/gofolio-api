package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/agent/agent.controller.ts

Routes + request/response shape:
- POST /api/v1/agent/chat (HandleCreateAgentChat)
  Request: body body: AgentChatRequest
  Auth/permission: jwt / permissions.accessAssistant
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/agent/feedback (HandleCreateAgentFeedback)
  Request: body body: AgentFeedbackRequest
  Auth/permission: jwt / permissions.accessAssistant
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/agent/session/{id} (HandleGetAgentSession)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.accessAssistant
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleCreateAgentChat(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleCreateAgentFeedback(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleGetAgentSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}
