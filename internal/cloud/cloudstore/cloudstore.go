package cloudstore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Gentleman-Programming/engram/internal/cloud"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type CloudStore struct {
	db *sql.DB
}

var ErrChunkNotFound = errors.New("cloudstore: chunk not found")
var ErrChunkConflict = errors.New("cloudstore: chunk id conflict")

func New(cfg cloud.Config) (*CloudStore, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("cloudstore: database dsn is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("cloudstore: open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cloudstore: ping postgres: %w", err)
	}

	cs := &CloudStore{
		db: db,
	}

	if err := cs.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("cloudstore: migrate schema: %w", err)
	}

	return cs, nil
}

func (cs *CloudStore) Close() error {
	return cs.db.Close()
}

func (cs *CloudStore) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS cloud_users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL DEFAULT '',
			api_key TEXT UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS cloud_projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id BIGINT REFERENCES cloud_users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS cloud_sessions (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES cloud_projects(id) ON DELETE CASCADE,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ended_at TIMESTAMPTZ,
			summary TEXT,
			directory TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cloud_sessions_project ON cloud_sessions(project_id)`,
		`CREATE TABLE IF NOT EXISTS cloud_observations (
			id BIGSERIAL PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES cloud_sessions(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			topic_key TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			search_vector tsvector GENERATED ALWAYS AS (
				setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
				setweight(to_tsvector('english', coalesce(content, '')), 'B')
			) STORED
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cloud_observations_session ON cloud_observations(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cloud_observations_search ON cloud_observations USING GIN (search_vector)`,
		`CREATE TABLE IF NOT EXISTS cloud_prompts (
			id BIGSERIAL PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES cloud_sessions(id) ON DELETE CASCADE,
			content TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cloud_prompts_session ON cloud_prompts(session_id)`,
		// ── Collaborative memberships ──
		`CREATE TABLE IF NOT EXISTS cloud_project_members (
			project_id TEXT NOT NULL REFERENCES cloud_projects(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES cloud_users(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT 'member',
			joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (project_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cloud_project_members_user ON cloud_project_members(user_id)`,
	}

	for i, query := range queries {
		if _, err := cs.db.Exec(query); err != nil {
			return fmt.Errorf("migration step %d failed: %w", i, err)
		}
	}
	return nil
}
