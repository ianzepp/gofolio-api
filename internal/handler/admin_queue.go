package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/admin/queue/queue.controller.ts

Routes + request/response shape:
- DELETE /api/v1/admin/queue/job (HandleDeleteQueueJobs)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/queue/job (HandleGetQueueJobs)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/admin/queue/job/{id} (HandleDeleteQueueJob)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/queue/job/{id}/execute (HandleExecuteQueueJob)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import (
	"fmt"
	"net/http"
)

// DELETE /admin/queue/job
func DeleteQueueJobs(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps != nil && deps.QueueService != nil {
			if err := deps.QueueService.DeleteAllJobs(r.Context()); err != nil {
				Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// GET /admin/queue/job
func GetQueueJobs(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.QueueService == nil {
			JSON(w, http.StatusOK, map[string]any{"jobs": []any{}})
			return
		}
		jobs, err := deps.QueueService.ListJobs(r.Context(), 200)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{"jobs": jobs})
	}
}

// DELETE /admin/queue/job/:id
func DeleteQueueJob(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if deps != nil && deps.QueueService != nil {
			if err := deps.QueueService.DeleteJob(r.Context(), id); err != nil {
				Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// GET /admin/queue/job/:id/execute
func ExecuteQueueJob(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if deps != nil && deps.QueueService != nil {
			if err := deps.QueueService.ExecuteJob(r.Context(), id); err != nil {
				Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func enqueueGatherJob(deps *Deps, r *http.Request, queueName, name string, data map[string]any) error {
	if deps == nil || deps.QueueService == nil {
		return nil
	}
	if _, err := deps.QueueService.Enqueue(r.Context(), queueName, name, data); err != nil {
		return fmt.Errorf("enqueue job: %w", err)
	}
	return nil
}
