package cloudstore

import (
	"context"
	"fmt"
	"time"
)

type CloudObservation struct {
	ID        int64   `json:"id"`
	SessionID string  `json:"session_id"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	TopicKey  *string `json:"topic_key,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
}

// CreateObservation creates a new observation, ensuring the user is a member of the session's project.
func (cs *CloudStore) CreateObservation(ctx context.Context, obs CloudObservation, userID int64) (int64, error) {
	if obs.CreatedAt == "" {
		obs.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	// Verify the user is a member of the project that owns this session.
	var memberCount int
	err := cs.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM cloud_sessions s
		JOIN cloud_projects p ON s.project_id = p.id
		LEFT JOIN cloud_project_members m ON p.id = m.project_id
		WHERE s.id = $1 AND (p.owner_id = $2 OR m.user_id = $2 OR p.owner_id IS NULL)`, obs.SessionID, userID).Scan(&memberCount)
	if err != nil {
		return 0, fmt.Errorf("verify session membership: %w", err)
	}
	if memberCount == 0 && userID != 0 {
		return 0, fmt.Errorf("session %q: user is not a member of this project", obs.SessionID)
	}

	topicKey := ""
	if obs.TopicKey != nil {
		topicKey = *obs.TopicKey
	}

	query := `
		INSERT INTO cloud_observations (session_id, type, title, content, topic_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var newID int64
	err = cs.db.QueryRowContext(ctx, query,
		obs.SessionID, obs.Type, obs.Title, obs.Content, topicKey, obs.CreatedAt,
	).Scan(&newID)

	if err != nil {
		return 0, fmt.Errorf("create observation: %w", err)
	}

	return newID, nil
}

// SearchObservations uses tsvector to search across all projects the user is a member of.
func (cs *CloudStore) SearchObservations(ctx context.Context, searchQuery string, limit int, userID int64) ([]CloudObservation, error) {
	query := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at 
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		JOIN cloud_project_members m ON s.project_id = m.project_id
		WHERE m.user_id = $1 
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

// Stats returns system statistics for the user.
func (cs *CloudStore) Stats(ctx context.Context, userID int64) (map[string]any, error) {
	var totalObs, totalSess, totalProj int64

	err := cs.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT o.id), COUNT(DISTINCT s.id), COUNT(DISTINCT m.project_id)
		FROM cloud_project_members m
		LEFT JOIN cloud_sessions s ON s.project_id = m.project_id
		LEFT JOIN cloud_observations o ON o.session_id = s.id
		WHERE m.user_id = $1
	`, userID).Scan(&totalObs, &totalSess, &totalProj)
	if err != nil {
		return nil, fmt.Errorf("stats: %w", err)
	}

	stats := map[string]any{
		"total_observations": int(totalObs),
		"total_sessions":     int(totalSess),
		"total_projects":     int(totalProj),
	}

	return stats, nil
}

// Timeline returns observations around a specific observation ID.
func (cs *CloudStore) Timeline(ctx context.Context, observationID int64, before, after int, userID int64) ([]CloudObservation, error) {
	// 1. Get the target observation to find its created_at and project_id
	target, err := cs.GetObservation(ctx, observationID, userID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch 'before' observations in the same project
	queryBefore := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		WHERE s.project_id = (SELECT project_id FROM cloud_sessions WHERE id = $1)
		  AND o.created_at < $2
		ORDER BY o.created_at DESC
		LIMIT $3
	`
	rowsBefore, err := cs.db.QueryContext(ctx, queryBefore, target.SessionID, target.CreatedAt, before)
	if err != nil {
		return nil, fmt.Errorf("timeline before: %w", err)
	}
	defer rowsBefore.Close()
	var resultsBefore []CloudObservation
	for rowsBefore.Next() {
		var o CloudObservation
		if err := rowsBefore.Scan(&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content, &o.TopicKey, &o.CreatedAt); err != nil {
			return nil, err
		}
		resultsBefore = append([]CloudObservation{o}, resultsBefore...) // prepend
	}

	// 3. Fetch 'after' observations
	queryAfter := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		WHERE s.project_id = (SELECT project_id FROM cloud_sessions WHERE id = $1)
		  AND o.created_at > $2
		ORDER BY o.created_at ASC
		LIMIT $3
	`
	rowsAfter, err := cs.db.QueryContext(ctx, queryAfter, target.SessionID, target.CreatedAt, after)
	if err != nil {
		return nil, fmt.Errorf("timeline after: %w", err)
	}
	defer rowsAfter.Close()
	var resultsAfter []CloudObservation
	for rowsAfter.Next() {
		var o CloudObservation
		if err := rowsAfter.Scan(&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content, &o.TopicKey, &o.CreatedAt); err != nil {
			return nil, err
		}
		resultsAfter = append(resultsAfter, o)
	}

	// 4. Combine
	timeline := append(resultsBefore, *target)
	timeline = append(timeline, resultsAfter...)

	return timeline, nil
}

// GetObservation retrieves an observation by ID, ensuring the user has access.
func (cs *CloudStore) GetObservation(ctx context.Context, id int64, userID int64) (*CloudObservation, error) {
	query := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at 
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		JOIN cloud_project_members m ON s.project_id = m.project_id
		WHERE o.id = $1 AND m.user_id = $2
	`
	var o CloudObservation
	err := cs.db.QueryRowContext(ctx, query, id, userID).Scan(
		&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content, &o.TopicKey, &o.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get observation: %w", err)
	}
	return &o, nil
}

// UpdateObservation modifies an existing observation.
func (cs *CloudStore) UpdateObservation(ctx context.Context, id int64, userID int64, title, content, typ, topicKey *string) error {
	// First verify ownership
	_, err := cs.GetObservation(ctx, id, userID)
	if err != nil {
		return err
	}

	// We only update provided fields. To keep it simple, we use a query that coalesces.
	query := `
		UPDATE cloud_observations
		SET 
			title = COALESCE($2, title),
			content = COALESCE($3, content),
			type = COALESCE($4, type),
			topic_key = COALESCE($5, topic_key)
		WHERE id = $1
	`
	_, err = cs.db.ExecContext(ctx, query, id, title, content, typ, topicKey)
	if err != nil {
		return fmt.Errorf("update observation: %w", err)
	}
	return nil
}

// DeleteObservation removes an observation.
func (cs *CloudStore) DeleteObservation(ctx context.Context, id int64, userID int64) error {
	// Verify ownership implicitly via the join in a subselect or explicitly.
	// Explicit is safer:
	_, err := cs.GetObservation(ctx, id, userID)
	if err != nil {
		return err
	}

	query := `DELETE FROM cloud_observations WHERE id = $1`
	_, err = cs.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete observation: %w", err)
	}
	return nil
}

// RecentObservations retrieves the most recent observations across the user's projects.
func (cs *CloudStore) RecentObservations(ctx context.Context, userID int64, project string, limit int) ([]CloudObservation, error) {
	query := `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.topic_key, o.created_at 
		FROM cloud_observations o
		JOIN cloud_sessions s ON o.session_id = s.id
		JOIN cloud_project_members m ON s.project_id = m.project_id
		WHERE m.user_id = $1
	`
	args := []any{userID}
	
	if project != "" {
		query += ` AND s.project_id = $2`
		args = append(args, project)
		query += ` ORDER BY o.created_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		query += ` ORDER BY o.created_at DESC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := cs.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("recent observations: %w", err)
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
