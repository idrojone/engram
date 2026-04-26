package cloudserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Gentleman-Programming/engram/internal/cloud/auth"
	"github.com/Gentleman-Programming/engram/internal/cloud/cloudstore"
)

type Option func(*CloudServer)

type Authenticator interface {
	Authorize(r *http.Request) error
}

type CloudServer struct {
	store          *cloudstore.CloudStore
	auth           Authenticator
	port           int
	host           string
	mux            *http.ServeMux
	listenAndServe func(addr string, handler http.Handler) error
}

const defaultHost = "0.0.0.0"

func WithHost(host string) Option {
	return func(s *CloudServer) {
		s.host = strings.TrimSpace(host)
	}
}

func New(store *cloudstore.CloudStore, authSvc Authenticator, port int, opts ...Option) *CloudServer {
	s := &CloudServer{
		store:          store,
		auth:           authSvc,
		port:           port,
		host:           defaultHost,
		listenAndServe: http.ListenAndServe,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.routes()
	return s
}

func (s *CloudServer) Start() error {
	host := strings.TrimSpace(s.host)
	if host == "" {
		host = defaultHost
	}
	addr := fmt.Sprintf("%s:%d", host, s.port)
	log.Printf("[engram-cloud] listening on %s", addr)
	return s.listenAndServe(addr, s.Handler())
}

func (s *CloudServer) Handler() http.Handler {
	if s.mux == nil {
		s.routes()
	}
	return s.mux
}

func (s *CloudServer) routes() {
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Centralized Auth & API Routes
	authHandler := NewAuthHandler(s.store)
	s.mux.HandleFunc("POST /api/v1/auth/register", authHandler.HandleRegister)

	authMiddleware := auth.NewMiddleware(s.store)
	apiRoutes := NewAPIRoutes(s.store)
	apiRoutes.Mount(s.mux, authMiddleware)
}

func (s *CloudServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "engram-cloud"}`))
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("[cloudserver] json encode failed: %v", err)
	}
}

func writeActionableError(w http.ResponseWriter, status int, class, code, message string) {
	jsonResponse(w, status, map[string]any{
		"error_class": class,
		"error_code":  code,
		"error":       message,
	})
}

