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
	mux.Handle("GET /api/v1/observations/{id}", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleGetObservation)))
	mux.Handle("PUT /api/v1/observations/{id}", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleUpdateObservation)))
	mux.Handle("DELETE /api/v1/observations/{id}", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleDeleteObservation)))

	mux.Handle("GET /api/v1/context/observations", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleRecentObservations)))
	mux.Handle("GET /api/v1/context/sessions", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleRecentSessions)))

	mux.Handle("GET /api/v1/stats", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleStats)))
	mux.Handle("GET /api/v1/observations/{id}/timeline", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleTimeline)))
	mux.Handle("PUT /api/v1/sessions/{id}/summary", authMiddleware.RequireAPIKey(http.HandlerFunc(a.handleUpdateSessionSummary)))
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

	// Make sure the project exists and the user is a member.
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

func (a *APIRoutes) handleGetObservation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_id", "invalid observation id")
		return
	}

	obs, err := a.store.GetObservation(r.Context(), id, userID)
	if err != nil {
		writeActionableError(w, http.StatusNotFound, "api", "not_found", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": obs})
}

func (a *APIRoutes) handleUpdateObservation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_id", "invalid observation id")
		return
	}

	var req struct {
		Type     *string `json:"type"`
		Title    *string `json:"title"`
		Content  *string `json:"content"`
		TopicKey *string `json:"topic_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_payload", "invalid payload")
		return
	}

	err = a.store.UpdateObservation(r.Context(), id, userID, req.Title, req.Content, req.Type, req.TopicKey)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *APIRoutes) handleDeleteObservation(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_id", "invalid observation id")
		return
	}

	err = a.store.DeleteObservation(r.Context(), id, userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *APIRoutes) handleRecentObservations(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := a.store.RecentObservations(r.Context(), userID, project, limit)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": results})
}

func (a *APIRoutes) handleRecentSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	project := r.URL.Query().Get("project")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := a.store.RecentSessions(r.Context(), userID, project, limit)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": results})
}

func (a *APIRoutes) handleStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	stats, err := a.store.Stats(r.Context(), userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": stats})
}

func (a *APIRoutes) handleTimeline(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_id", "invalid observation id")
		return
	}

	before := 5
	if bStr := r.URL.Query().Get("before"); bStr != "" {
		if b, err := strconv.Atoi(bStr); err == nil && b >= 0 {
			before = b
		}
	}
	after := 5
	if aStr := r.URL.Query().Get("after"); aStr != "" {
		if a, err := strconv.Atoi(aStr); err == nil && a >= 0 {
			after = a
		}
	}

	results, err := a.store.Timeline(r.Context(), id, before, after, userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok", "data": results})
}

func (a *APIRoutes) handleUpdateSessionSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		writeActionableError(w, http.StatusUnauthorized, "api", "unauthorized", "user not found in context")
		return
	}

	sessionID := r.PathValue("id")

	var req struct {
		Summary *string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "api", "invalid_payload", "invalid payload")
		return
	}

	err := a.store.UpdateSessionSummary(r.Context(), sessionID, req.Summary, userID)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "api", "db_error", err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"status": "ok"})
}
