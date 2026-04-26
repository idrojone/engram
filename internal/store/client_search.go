package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Search queries the cloud API for observations.
func (s *Store) Search(query string, opts SearchOptions) ([]SearchResult, error) {
	var resp struct {
		Status string        `json:"status"`
		Data   []Observation `json:"data"`
	}

	q := url.Values{}
	q.Add("q", query)
	if opts.Limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", opts.Limit))
	}

	err := s.req(context.Background(), "GET", "/api/v1/search?"+q.Encode(), nil, &resp)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(resp.Data))
	for i, obs := range resp.Data {
		results[i] = SearchResult{
			Observation: obs,
			Rank:        1.0, // Since ts_rank isn't returned by default in the struct yet
		}
	}

	return results, nil
}

// FormatContext retrieves search results and formats them.
func (s *Store) FormatContext(project, scope string) (string, error) {
	// Fallback/stub
	return "Context formatting moved to cloud.", nil
}

func (s *Store) Timeline(observationID int64, before, after int) (*TimelineResult, error) {
	return nil, errors.New("Timeline not implemented in cloud proxy")
}

func (s *Store) Stats() (*Stats, error) {
	return &Stats{}, nil
}

func (s *Store) Export() (*ExportData, error) {
	return nil, errors.New("Export not implemented in cloud proxy")
}

func (s *Store) GetObservation(id int64) (*Observation, error) {
	return nil, errors.New("GetObservation not implemented")
}

func (s *Store) HardDeleteObservation(id int64) error {
	return errors.New("HardDeleteObservation not implemented")
}

func (s *Store) DeleteSession(id string) error {
	return errors.New("DeleteSession not implemented")
}

func (s *Store) DeleteProject(project string) error {
	return errors.New("DeleteProject not implemented")
}

func (s *Store) EnrolledProjects() ([]string, error) {
	return nil, nil
}

func (s *Store) DeletePrompt(id int64) error {
	return errors.New("DeletePrompt not implemented")
}

func (s *Store) UpdateSession(id string, summary *string) error {
	return errors.New("UpdateSession not implemented")
}

func (s *Store) RecentSessions(project string, limit int) ([]SessionSummary, error) {
	return nil, errors.New("RecentSessions not implemented")
}

func (s *Store) RecentObservations(project, scope string, limit int) ([]Observation, error) {
	return nil, errors.New("RecentObservations not implemented")
}

func (s *Store) RecentPrompts(project string, limit int) ([]Prompt, error) {
	return nil, errors.New("RecentPrompts not implemented")
}

func (s *Store) SearchPrompts(query, project string, limit int) ([]Prompt, error) {
	return nil, errors.New("SearchPrompts not implemented")
}

func (s *Store) PassiveCapture(params PassiveCaptureParams) (PassiveCaptureResult, error) {
	return PassiveCaptureResult{}, errors.New("PassiveCapture not implemented")
}

func (s *Store) IsProjectSyncEnabled(project string) (bool, error) {
	return false, nil
}

func (s *Store) ListProjectNames() ([]string, error) {
	return nil, nil
}

func (s *Store) CountObservationsForProject(project string) (int, error) {
	return 0, nil
}

func (s *Store) ExportProject(project string) (*ExportData, error) {
	return nil, nil
}

type ImportResult struct {
	SessionsImported     int  `json:"sessions_imported"`
	ObservationsImported int  `json:"observations_imported"`
	PromptsImported      int  `json:"prompts_imported"`
	Migrated             bool `json:"migrated"`
	ObservationsUpdated  int  `json:"observations_updated"`
	SessionsUpdated      int  `json:"sessions_updated"`
	PromptsUpdated       int  `json:"prompts_updated"`
}

func (s *Store) Import(payload *ExportData) (*ImportResult, error) {
	return &ImportResult{}, nil
}

func (s *Store) MigrateProject(project string, targetDir string) (*ImportResult, error) {
	return &ImportResult{}, nil
}

func (s *Store) MaxObservationLength() int {
	return 50000
}

func (s *Store) CreateSession(id string, project string, directory string) error {
	_, err := s.StartSession(project, directory)
	return err
}

func (s *Store) DeleteObservation(id int64, hard bool) error {
	return errors.New("DeleteObservation not implemented")
}

func (s *Store) UpdateObservation(id int64, params UpdateObservationParams) (*Observation, error) {
	return nil, errors.New("UpdateObservation not implemented")
}
