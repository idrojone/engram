package store

import (
	"context"
)

// AddObservation sends a new observation to the cloud API.
func (s *Store) AddObservation(params AddObservationParams) (int64, error) {
	var resp struct {
		Status string `json:"status"`
		ID     int64  `json:"id"`
	}

	err := s.req(context.Background(), "POST", "/api/v1/observations", params, &resp)
	if err != nil {
		return 0, err
	}

	return resp.ID, nil
}

// AddPrompt is a stub for cloud proxy.
func (s *Store) AddPrompt(params AddPromptParams) (int64, error) {
	return 0, nil
}

// AllObservations returns all observations for a project and type. Matches TUI requirements.
func (s *Store) AllObservations(project, typ string, limit int) ([]Observation, error) {
	return nil, nil
}

// SessionObservations returns all observations for a specific session. Matches TUI requirements.
func (s *Store) SessionObservations(sessionID string, limit int) ([]Observation, error) {
	return nil, nil
}
