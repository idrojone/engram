package cloudstore

import (
	"context"
	"errors"
	"fmt"
)

type CloudSession struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	StartedAt string  `json:"started_at,omitempty"`
	EndedAt   *string `json:"ended_at,omitempty"`
	Summary   *string `json:"summary,omitempty"`
	Directory string  `json:"directory,omitempty"`
}

// CreateSession creates a new session in the centralized cloud.
func (cs *CloudStore) CreateSession(ctx context.Context, session CloudSession) error {
	query := `
		INSERT INTO cloud_sessions (id, project_id, started_at, directory) 
		VALUES ($1, $2, $3, $4)
	`
	_, err := cs.db.ExecContext(ctx, query, session.ID, session.ProjectID, session.StartedAt, session.Directory)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// EndSession marks a session as completed, ensuring the user is a member of the project.
func (cs *CloudStore) EndSession(ctx context.Context, sessionID, endedAt string, summary *string, userID int64) error {
	query := `
		UPDATE cloud_sessions 
		SET ended_at = $1, summary = $2
		FROM cloud_project_members m
		WHERE cloud_sessions.id = $3 
		  AND cloud_sessions.project_id = m.project_id 
		  AND m.user_id = $4
	`
	res, err := cs.db.ExecContext(ctx, query, endedAt, summary, sessionID, userID)
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("cloudstore: session not found or unauthorized")
	}

	return nil
}

// RecentSessions retrieves the most recent sessions across the user's projects.
func (cs *CloudStore) RecentSessions(ctx context.Context, userID int64, project string, limit int) ([]CloudSession, error) {
	query := `
		SELECT s.id, s.project_id, s.started_at, s.ended_at, s.summary, s.directory
		FROM cloud_sessions s
		JOIN cloud_project_members m ON s.project_id = m.project_id
		WHERE m.user_id = $1
	`
	args := []any{userID}

	if project != "" {
		query += ` AND s.project_id = $2`
		args = append(args, project)
		query += ` ORDER BY s.started_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		query += ` ORDER BY s.started_at DESC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := cs.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("recent sessions: %w", err)
	}
	defer rows.Close()

	var results []CloudSession
	for rows.Next() {
		var s CloudSession
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.StartedAt, &s.EndedAt, &s.Summary, &s.Directory); err != nil {
			return nil, err
		}
		results = append(results, s)
	}

	return results, rows.Err()
}

// UpdateSessionSummary modifies the summary of a session without ending it.
func (cs *CloudStore) UpdateSessionSummary(ctx context.Context, sessionID string, summary *string, userID int64) error {
	query := `
		UPDATE cloud_sessions 
		SET summary = $1
		FROM cloud_project_members m
		WHERE cloud_sessions.id = $2
		  AND cloud_sessions.project_id = m.project_id 
		  AND m.user_id = $3
	`
	res, err := cs.db.ExecContext(ctx, query, summary, sessionID, userID)
	if err != nil {
		return fmt.Errorf("update session summary: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("cloudstore: session not found or unauthorized")
	}

	return nil
}
