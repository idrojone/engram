package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Sentinel errors
var (
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionHasObservations = errors.New("session still has observations")
	ErrObservationNotFound    = errors.New("observation not found")
	ErrPromptNotFound         = errors.New("prompt not found")
)

// ─── Types (from Legacy SQLite, preserved for compat) ───────────

type Session struct {
	ID        string  `json:"id"`
	Project   string  `json:"project"`
	Directory string  `json:"directory"`
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at,omitempty"`
	Summary   *string `json:"summary,omitempty"`
}

type Observation struct {
	ID             int64   `json:"id"`
	SyncID         string  `json:"sync_id"`
	SessionID      string  `json:"session_id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ToolName       *string `json:"tool_name,omitempty"`
	Project        *string `json:"project,omitempty"`
	Scope          string  `json:"scope"`
	TopicKey       *string `json:"topic_key,omitempty"`
	RevisionCount  int     `json:"revision_count"`
	DuplicateCount int     `json:"duplicate_count"`
	LastSeenAt     *string `json:"last_seen_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
}

type SearchResult struct {
	Observation
	Rank       float64 `json:"rank"`
	Score      float64 `json:"score,omitempty"`
	JudgmentID *int64  `json:"judgment_id,omitempty"`
}

type SearchOptions struct {
	Type    string `json:"type,omitempty"`
	Project string `json:"project,omitempty"`
	Scope   string `json:"scope,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type AddObservationParams struct {
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ToolName  string `json:"tool_name,omitempty"`
	Project   string `json:"project,omitempty"`
	Scope     string `json:"scope,omitempty"`
	TopicKey  string `json:"topic_key,omitempty"`
}

type AddPromptParams struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
}

type UpdateObservationParams struct {
	Type     *string `json:"type,omitempty"`
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	Project  *string `json:"project,omitempty"`
	Scope    *string `json:"scope,omitempty"`
	TopicKey *string `json:"topic_key,omitempty"`
}

type TimelineEntry struct {
	ID             int64   `json:"id"`
	SessionID      string  `json:"session_id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ToolName       *string `json:"tool_name,omitempty"`
	Project        *string `json:"project,omitempty"`
	Scope          string  `json:"scope"`
	TopicKey       *string `json:"topic_key,omitempty"`
	RevisionCount  int     `json:"revision_count"`
	DuplicateCount int     `json:"duplicate_count"`
	LastSeenAt     *string `json:"last_seen_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	IsFocus        bool    `json:"is_focus"` 
}

type TimelineResult struct {
	Focus        Observation     `json:"focus"`        
	Before       []TimelineEntry `json:"before"`       
	After        []TimelineEntry `json:"after"`        
	SessionInfo  *Session        `json:"session_info"` 
	TotalInRange int             `json:"total_in_range"`
}

type Stats struct {
	TotalSessions     int      `json:"total_sessions"`
	TotalObservations int      `json:"total_observations"`
	TotalPrompts      int      `json:"total_prompts"`
	Projects          []string `json:"projects"`
}

type Prompt struct {
	ID        int64  `json:"id"`
	SyncID    string `json:"sync_id"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
	CreatedAt string `json:"created_at"`
}

type ExportData struct {
	Version      string        `json:"version"`
	ExportedAt   string        `json:"exported_at"`
	Sessions     []Session     `json:"sessions"`
	Observations []Observation `json:"observations"`
}

type SessionSummary struct {
	ID               string  `json:"id"`
	Project          string  `json:"project"`
	StartedAt        string  `json:"started_at"`
	EndedAt          *string `json:"ended_at,omitempty"`
	Summary          *string `json:"summary,omitempty"`
	ObservationCount int     `json:"observation_count"`
}

type PassiveCaptureParams struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
	Source    string `json:"source,omitempty"`
}

type PassiveCaptureResult struct {
	Extracted  int `json:"extracted"`
	Saved      int `json:"saved"`
	Duplicates int `json:"duplicates"`
}

type SyncMutation struct {
	Seq        int64   `json:"seq"`
	TargetKey  string  `json:"target_key"`
	Entity     string  `json:"entity"`
	EntityKey  string  `json:"entity_key"`
	Op         string  `json:"op"`
	Payload    string  `json:"payload"`
	Source     string  `json:"source"`
	Project    string  `json:"project"`
	OccurredAt string  `json:"occurred_at"`
	AckedAt    *string `json:"acked_at,omitempty"`
}

const (
	DefaultSyncTargetKey  = "cloud"
	SyncEntitySession     = "session"
	SyncEntityObservation = "observation"
	SyncEntityPrompt      = "prompt"

	SyncOpUpsert = "upsert"
	SyncOpDelete = "delete"

	RelationSupersedes = "supersedes"
	RelationPending    = "pending"
)

var ErrSessionDeleteBlocked = errors.New("session deletion is blocked while cloud sync enrollment is active")

func NormalizeProject(project string) (string, string) {
	if project == "" {
		return "", ""
	}
	n := strings.TrimSpace(strings.ToLower(project))
	for strings.Contains(n, "--") {
		n = strings.ReplaceAll(n, "--", "-")
	}
	for strings.Contains(n, "__") {
		n = strings.ReplaceAll(n, "__", "_")
	}
	return n, ""
}

func SuggestTopicKey(typ, title, content string) string {
	return "migrated-topic"
}

// ─── HTTP Proxy Store ───────────────────────────────────────────

type Config struct {
	BaseURL string
	APIKey  string
}

type Store struct {
	cfg    Config
	client *http.Client
}

// New creates a new HTTP-based Store connecting to the Cloud API.
func New(cfg Config) (*Store, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("cloud store: base_url is required")
	}

	return &Store{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (s *Store) Close() error {
	return nil // No db to close!
}

// req executes an HTTP request with JSON and Auth Header.
func (s *Store) req(ctx context.Context, method, endpoint string, payload any, response any) error {
	var bodyReader io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := strings.TrimRight(s.cfg.BaseURL, "/") + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return err
		}
	}
	return nil
}
