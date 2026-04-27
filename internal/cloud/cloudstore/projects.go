package cloudstore

import (
	"context"
	"fmt"
)

// EnsureProject creates the project if it doesn't exist (first user becomes owner),
// then adds the user as a member. Any authenticated user can join a project.
func (cs *CloudStore) EnsureProject(ctx context.Context, projectID string, userID int64) error {
	// Insert the project if it doesn't exist. The first user to create it becomes the owner.
	_, err := cs.db.ExecContext(ctx, `
		INSERT INTO cloud_projects (id, name, owner_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`, projectID, projectID, userID)
	if err != nil {
		return fmt.Errorf("ensure project insert: %w", err)
	}

	// Add the user as a member (idempotent via ON CONFLICT).
	// If the user is the owner, they get the "owner" role; otherwise "member".
	_, err = cs.db.ExecContext(ctx, `
		INSERT INTO cloud_project_members (project_id, user_id, role)
		VALUES ($1, $2,
			CASE WHEN (SELECT owner_id FROM cloud_projects WHERE id = $1) = $2
			     THEN 'owner' ELSE 'member' END
		)
		ON CONFLICT (project_id, user_id) DO NOTHING
	`, projectID, userID)
	if err != nil {
		return fmt.Errorf("ensure project membership: %w", err)
	}

	return nil
}

// IsMember checks if a user has access to a project.
func (cs *CloudStore) IsMember(ctx context.Context, projectID string, userID int64) (bool, error) {
	var exists bool
	err := cs.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM cloud_project_members
			WHERE project_id = $1 AND user_id = $2
		)
	`, projectID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return exists, nil
}
