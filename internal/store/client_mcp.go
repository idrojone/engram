package store

// ─── MCP Types (from Legacy SQLite, preserved for compat) ───────────

type Relation struct {
	ID                    int64    `json:"id"`
	SyncID                string   `json:"sync_id"`
	SourceID              string   `json:"source_id"`
	TargetID              string   `json:"target_id"`
	Relation              string   `json:"relation"`
	Reason                *string  `json:"reason,omitempty"`
	Evidence              *string  `json:"evidence,omitempty"`
	Confidence            *float64 `json:"confidence,omitempty"`
	JudgmentStatus        string   `json:"judgment_status"`
	MarkedByActor         *string  `json:"marked_by_actor,omitempty"`
	MarkedByKind          *string  `json:"marked_by_kind,omitempty"`
	MarkedByModel         *string  `json:"marked_by_model,omitempty"`
	SessionID             *string  `json:"session_id,omitempty"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

type ObservationRelations struct {
	AsSource []Relation `json:"as_source"`
	AsTarget []Relation `json:"as_target"`
}

type JudgeRelationParams struct {
	JudgmentID    string   `json:"judgment_id"`
	Relation      string   `json:"relation"`
	Reason        *string  `json:"reason,omitempty"`
	Evidence      *string  `json:"evidence,omitempty"`
	Confidence    *float64 `json:"confidence,omitempty"`
	MarkedByActor string   `json:"marked_by_actor"`
	MarkedByKind  string   `json:"marked_by_kind"`
	MarkedByModel string   `json:"marked_by_model"`
	SessionID     string   `json:"session_id,omitempty"`
}

type Candidate struct {
	ID         int64    `json:"id"`
	SyncID     string   `json:"sync_id"`
	Title      string   `json:"title"`
	Type       string   `json:"type"`
	TopicKey   *string  `json:"topic_key,omitempty"`
	Score      float64  `json:"score"`
	JudgmentID string   `json:"judgment_id"`
}

type CandidateOptions struct {
	Project   string
	Scope     string
	Type      string
	Limit     int
	BM25Floor *float64
}

type MergeResult struct {
	Canonical            string   `json:"canonical"`
	SourcesMerged        []string `json:"sources_merged"`
	ObservationsUpdated  int      `json:"observations_updated"`
	SessionsUpdated      int      `json:"sessions_updated"`
	PromptsUpdated       int      `json:"prompts_updated"`
}

// ─── MCP Stubs ──────────────────────────────────────────────────

func (s *Store) GetRelationsForObservations(syncIDs []string) (map[string]ObservationRelations, error) {
	return map[string]ObservationRelations{}, nil
}

func (s *Store) JudgeRelation(p JudgeRelationParams) (*Relation, error) {
	return nil, nil
}

func (s *Store) FindCandidates(id int64, opts CandidateOptions) ([]Candidate, error) {
	return nil, nil
}

func (s *Store) MergeProjects(from []string, to string) (*MergeResult, error) {
	return &MergeResult{Canonical: to, SourcesMerged: from}, nil
}

func (s *Store) ProjectExists(name string) (bool, error) {
	return true, nil
}
