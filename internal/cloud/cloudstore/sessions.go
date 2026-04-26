package cloudstore

import (
	"context"
	// "database/sql"
	"errors"
	"fmt"
)

type CloudSession struct {
	ID        string
	ProjectID string
	StartedAt string
	EndedAt   *string
	Summary   *string
	Directory string
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

// EndSession marks a session as completed, ensuring it belongs to the authenticated user.
func (cs *CloudStore) EndSession(ctx context.Context, sessionID, endedAt string, summary *string, userID int64) error {
	query := `
		UPDATE cloud_sessions 
		SET ended_at = $1, summary = $2
		FROM cloud_projects
		WHERE cloud_sessions.id = $3 
		  AND cloud_sessions.project_id = cloud_projects.id 
		  AND cloud_projects.owner_id = $4
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
