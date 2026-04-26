package cloudstore

import (
	"context"
	"fmt"
	"time"
)

type CloudObservation struct {
	ID        int64
	SessionID string
	Type      string
	Title     string
	Content   string
	TopicKey  *string
	CreatedAt string
}

// CreateObservation creates a new observation in PostgreSQL, ensuring session ownership.
func (cs *CloudStore) CreateObservation(ctx context.Context, obs CloudObservation, userID int64) (int64, error) {
	if obs.CreatedAt == "" {
		obs.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Verify session ownership
	var sessionOwner int64
	err := cs.db.QueryRowContext(ctx, `
		SELECT p.owner_id 
		FROM cloud_sessions s
		JOIN cloud_projects p ON s.project_id = p.id
		WHERE s.id = $1`, obs.SessionID).Scan(&sessionOwner)
	if err != nil {
		return 0, fmt.Errorf("verify session ownership: %w", err)
	}
	if sessionOwner != userID {
		return 0, fmt.Errorf("session %q belongs to another user", obs.SessionID)
	}

	query := `
		INSERT INTO cloud_observations (session_id, type, title, content, topic_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	
	var newID int64
	err = cs.db.QueryRowContext(ctx, query, 
		obs.SessionID, obs.Type, obs.Title, obs.Content, obs.TopicKey, obs.CreatedAt,
	).Scan(&newID)

	if err != nil {
		return 0, fmt.Errorf("create observation: %w", err)
	}

	return newID, nil
}

// SearchObservations uses tsvector to search, restricted to the authenticated user's data.
func (cs *CloudStore) SearchObservations(ctx context.Context, searchQuery string, limit int, userID int64) ([]CloudObservation, error) {
	query := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at 
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		JOIN cloud_projects p ON s.project_id = p.id
		WHERE p.owner_id = $1 
		  AND o.search_vector @@ plainto_tsquery('english', $2)
		ORDER BY ts_rank(o.search_vector, plainto_tsquery('english', $2)) DESC, o.created_at DESC
		LIMIT $3
	`

	rows, err := cs.db.QueryContext(ctx, query, userID, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search observations: %w", err)
	}
	defer rows.Close()

	var results []CloudObservation
	for rows.Next() {
		var o CloudObservation
		if err := rows.Scan(&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content, &o.TopicKey, &o.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, o)
	}

	return results, rows.Err()
}
