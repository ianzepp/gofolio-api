// Package handler — agent.go implements handlers for /api/v1/agent/*.
//
// The agent chat endpoint streams responses as NDJSON (newline-delimited JSON)
// rather than a single JSON blob. Each emitted chunk is flushed immediately via
// http.Flusher to support real-time streaming to the client. The stream protocol
// uses typed event objects:
//
//	{ "type": "chunk", "content": "..." }   — a text fragment
//	{ "type": "done",  "response": {...} }  — stream terminator
//
// Currently HandleCreateAgentChat is a stub that echoes the message back in the
// "done" event. The actual LLM call and tool-use loop will replace the body once
// the agent service layer is wired.
//
// Parity reference: apps/api/src/app/agent/agent.controller.ts
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

import (
	"encoding/json"
	"net/http"
	"strings"
)

func HandleCreateAgentChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message string `json:"message"`
	}
	if !Bind(w, r, &body) {
		return
	}

	message := strings.TrimSpace(body.Message)
	if message == "" {
		Error(w, http.StatusBadRequest, "Message is required")
		return
	}
	if len(message) > 4000 {
		Error(w, http.StatusBadRequest, "Message must be 4000 characters or less")
		return
	}

	// Legacy behavior note:
	// Agent chat is streamed as NDJSON/chunked response, not a single JSON blob.
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeStreamEvent(w, map[string]any{
		"type":    "chunk",
		"content": "",
	})
	writeStreamEvent(w, map[string]any{
		"type": "done",
		"response": map[string]any{
			"message": message,
		},
	})
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

// writeStreamEvent marshals event to JSON, appends a newline, writes it to w,
// and flushes the underlying connection if it implements http.Flusher. Errors
// from Marshal are silently swallowed because a partial stream is preferable to
// an abrupt close; errors from Write are also silently swallowed (client gone).
func writeStreamEvent(w http.ResponseWriter, event map[string]any) {
	encoded, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, _ = w.Write(append(encoded, '\n'))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
