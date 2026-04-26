package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfig returns the default configuration for the cloud client.
func DefaultConfig() (Config, error) {
	return LoadConfig()
}

// FallbackConfig returns a config with a specific data dir (legacy, but kept for main.go compat).
func FallbackConfig(dataDir string) Config {
	return Config{
		BaseURL: "http://localhost:18080",
	}
}

// LoadConfig reads the configuration from ~/.engram/config.json and 
// applies environment variable overrides.
func LoadConfig() (Config, error) {
	var cfg Config

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, fmt.Errorf("engram: determine home directory: %w", err)
	}

	configPath := filepath.Join(home, ".engram", "config.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("engram: parse config.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("engram: read config.json: %w", err)
	}

	// Environment variable overrides
	if envURL := os.Getenv("ENGRAM_API_URL"); envURL != "" {
		cfg.BaseURL = envURL
	}
	if envKey := os.Getenv("ENGRAM_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	}

	// Fallback to defaults if totally empty
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:18080"
	}

	return cfg, nil
}
