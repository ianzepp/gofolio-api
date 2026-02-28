package handler

import "net/http"

// DELETE /admin/queue/job
func HandleDeleteQueueJobs(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /admin/queue/job
func HandleGetQueueJobs(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

// DELETE /admin/queue/job/:id
func HandleDeleteQueueJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /admin/queue/job/:id/execute
func HandleExecuteQueueJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}
