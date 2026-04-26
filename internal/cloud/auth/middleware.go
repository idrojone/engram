package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/Gentleman-Programming/engram/internal/cloud/cloudstore"
)

type contextKey string

const UserContextKey contextKey = "user_id"

// Middleware struct encapsulates the cloudstore dependency for auth.
type Middleware struct {
	store *cloudstore.CloudStore
}

func NewMiddleware(store *cloudstore.CloudStore) *Middleware {
	return &Middleware{store: store}
}

// RequireAPIKey is an HTTP middleware that extracts the Bearer token
// and validates it against the cloud_users API keys.
func (m *Middleware) RequireAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "authorization must use Bearer token", http.StatusUnauthorized)
			return
		}

		apiKey := strings.TrimSpace(parts[1])
		if apiKey == "" {
			http.Error(w, "bearer token is empty", http.StatusUnauthorized)
			return
		}

		user, err := m.store.AuthenticateAPIKey(r.Context(), apiKey)
		if err != nil {
			if err == cloudstore.ErrInvalidAPIKey {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
			} else {
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Inject the authenticated user ID into the request context
		ctx := context.WithValue(r.Context(), UserContextKey, user.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the request context.
func GetUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserContextKey).(int64)
	return id, ok
}
