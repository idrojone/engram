package cloudserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Gentleman-Programming/engram/internal/cloud/cloudstore"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler groups the HTTP handlers related to user authentication.
type AuthHandler struct {
	store *cloudstore.CloudStore
}

func NewAuthHandler(store *cloudstore.CloudStore) *AuthHandler {
	return &AuthHandler{store: store}
}

// HandleRegister registers a new user and returns their API Key.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "auth", "invalid_payload", "invalid JSON payload")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeActionableError(w, http.StatusBadRequest, "auth", "missing_fields", "username, email, and password are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeActionableError(w, http.StatusInternalServerError, "auth", "internal_error", "failed to process password")
		return
	}
	passwordHash := string(hash)

	user, err := h.store.RegisterUser(r.Context(), req.Username, req.Email, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			writeActionableError(w, http.StatusConflict, "auth", "user_exists", "username or email already exists")
			return
		}
		writeActionableError(w, http.StatusInternalServerError, "auth", "internal_error", "failed to register user")
		return
	}

	// Return the newly generated API Key so the client can configure it.
	jsonResponse(w, http.StatusCreated, map[string]any{
		"status":   "ok",
		"user_id":  user.ID,
		"username": user.Username,
		"api_key":  user.APIKey, // The client must save this securely
	})
}

// HandleLogin authenticates a user and returns their API Key.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeActionableError(w, http.StatusBadRequest, "auth", "invalid_payload", "invalid JSON payload")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		writeActionableError(w, http.StatusBadRequest, "auth", "missing_fields", "username and password are required")
		return
	}

	user, err := h.store.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		// Generic message to avoid username enumeration
		writeActionableError(w, http.StatusUnauthorized, "auth", "invalid_credentials", "invalid username or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeActionableError(w, http.StatusUnauthorized, "auth", "invalid_credentials", "invalid username or password")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"user_id":  user.ID,
		"username": user.Username,
		"api_key":  user.APIKey,
	})
}
