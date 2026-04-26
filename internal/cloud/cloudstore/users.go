package cloudstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrUserNotFound = errors.New("cloudstore: user not found")
var ErrInvalidAPIKey = errors.New("cloudstore: invalid api key")

type CloudUser struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	APIKey       string
}

// AuthenticateAPIKey verifies the API key and returns the associated user.
func (cs *CloudStore) AuthenticateAPIKey(ctx context.Context, apiKey string) (*CloudUser, error) {
	if cs == nil || cs.db == nil {
		return nil, fmt.Errorf("cloudstore: not initialized")
	}

	var user CloudUser
	var dbAPIKey sql.NullString

	query := `SELECT id, username, email, password_hash, api_key FROM cloud_users WHERE api_key = $1`
	err := cs.db.QueryRowContext(ctx, query, apiKey).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&dbAPIKey,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidAPIKey
	}
	if err != nil {
		return nil, fmt.Errorf("authenticate api key: %w", err)
	}

	if dbAPIKey.Valid {
		user.APIKey = dbAPIKey.String
	}

	return &user, nil
}

// GenerateAPIKey creates a secure random API key.
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RegisterUser creates a new user and generates an API Key.
func (cs *CloudStore) RegisterUser(ctx context.Context, username, email, passwordHash string) (*CloudUser, error) {
	apiKey, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	query := `
		INSERT INTO cloud_users (username, email, password_hash, api_key) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id
	`
	
	var id int64
	err = cs.db.QueryRowContext(ctx, query, username, email, passwordHash, apiKey).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}

	return &CloudUser{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		APIKey:       apiKey,
	}, nil
}

// GetUserByUsername retrieves a user by their username.
func (cs *CloudStore) GetUserByUsername(ctx context.Context, username string) (*CloudUser, error) {
	if cs == nil || cs.db == nil {
		return nil, fmt.Errorf("cloudstore: not initialized")
	}

	var user CloudUser
	var dbAPIKey sql.NullString

	query := `SELECT id, username, email, password_hash, api_key FROM cloud_users WHERE username = $1`
	err := cs.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&dbAPIKey,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	if dbAPIKey.Valid {
		user.APIKey = dbAPIKey.String
	}

	return &user, nil
}
