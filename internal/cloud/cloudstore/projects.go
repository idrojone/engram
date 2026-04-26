package cloudstore

import (
	"context"
	"fmt"
)

// EnsureProject inserts the project if it does not exist and validates ownership.
func (cs *CloudStore) EnsureProject(ctx context.Context, projectID string, ownerID int64) error {
	// First, try to insert. If it exists, it will do nothing.
	query := `
		INSERT INTO cloud_projects (id, name, owner_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := cs.db.ExecContext(ctx, query, projectID, projectID, ownerID)
	if err != nil {
		return fmt.Errorf("ensure project insert: %w", err)
	}

	// Verify ownership if it already existed (or was just inserted)
	var actualOwner int64
	err = cs.db.QueryRowContext(ctx, `SELECT owner_id FROM cloud_projects WHERE id = $1`, projectID).Scan(&actualOwner)
	if err != nil {
		return fmt.Errorf("verify project ownership: %w", err)
	}

	if actualOwner != ownerID {
		return fmt.Errorf("project %q is owned by another user", projectID)
	}

	return nil
}
