package cloudserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Gentleman-Programming/engram/internal/cloud/auth"
	"github.com/Gentleman-Programming/engram/internal/cloud/cloudstore"
)

// APIRoutes mounts all the REST endpoints for the client MCP proxy.
type APIRoutes struct {
	store *cloudstore.CloudStore
}

func NewAPIRoutes(store *cloudstore.CloudStore) *APIRoutes {
	return &APIRoutes{store: store}
}

// Mount attaches the API routes to the ServeMux, protected by Auth Middleware.
func (a *APIRoutes) Mount(mux *http.ServeMux, authMiddleware *auth.Middleware) {
	mux.Handle("POST /api/v1/sessions", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleCreateSession)))
	mux.Handle("PUT /api/v1/sessions/{id}/end", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleEndSession)))
	mux.Handle("POST /api/v1/observations", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleCreateObservation)))
	mux.Handle("GET /api/v1/search", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleSearch)))
}

func (a *APIRoutes) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	var req cloudstore.CloudSession
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_payload", "invalid session payload")
		return
	}

	if req.StartedAt == "" {
		req.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Make sure the project exists and is owned by the user.
	if err := a.store.EnsureProject(r.Context(), req.ProjectID, userID); err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	if err := a.store.CreateSession(r.Context(), req); err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]any{"status": "ok", "id": req.ID})
}

func (a *APIRoutes) handleEndSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	sessionID := r.PathValue("id")
	
	var req struct {
		EndedAt string  `json:"ended_at"`
		Summary *string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_payload", "invalid session payload")
		return
	}

	if req.EndedAt == "" {
		req.EndedAt = time.Now().UTC().Format(time.RFC3339)
	}

	if err := a.store.EndSession(r.Context(), sessionID, req.EndedAt, req.Summary, userID); err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *APIRoutes) handleCreateObservation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	var req cloudstore.CloudObservation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_payload", "invalid observation payload")
		return
	}

	id, err := a.store.CreateObservation(r.Context(), req, userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]any{"status": "ok", "id": id})
}

func (a *APIRoutes) handleSearch(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := a.store.SearchObservations(r.Context(), query, limit, userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": results})
}
