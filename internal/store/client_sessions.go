package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// StartSession initiates a new session in the cloud.
func (s *Store) StartSession(project, directory string) (string, error) {
	// Generate a unique session ID
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	payload := Session{
		ID:        sessionID,
		Project:   project,
		Directory: directory,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	err := s.req(context.Background(), "POST", "/api/v1/sessions", payload, nil)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

// EndSession concludes a session in the cloud.
func (s *Store) EndSession(id, summary string) error {
	payload := map[string]string{
		"ended_at": time.Now().UTC().Format(time.RFC3339),
		"summary":  summary,
	}

	return s.req(context.Background(), "PUT", "/api/v1/sessions/"+id+"/end", payload, nil)
}

// GetSession is a stub since we removed local SQLite queries.
func (s *Store) GetSession(id string) (*Session, error) {
	return nil, errors.New("GetSession not implemented in cloud proxy")
}

// AllSessions returns all sessions for a project. Matches TUI requirements.
func (s *Store) AllSessions(project string, limit int) ([]SessionSummary, error) {
	// Stub for TUI compatibility
	return nil, nil
}
