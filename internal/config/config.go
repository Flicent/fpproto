package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPath returns the full path to ~/.fpproto/config.json.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, ConfigFile)
}

// PrototypesPath returns the full path to ~/prototypes.
func PrototypesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, PrototypesDir)
}

// Load reads the config from disk. If the file does not exist it returns an
// error instructing the user to run "fpproto setup".
func Load() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s — run `fpproto setup` to create it", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk as pretty-printed JSON. It creates the config
// directory with 0700 permissions if it does not already exist.
func Save(cfg *Config) error {
	path := ConfigPath()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Append a trailing newline for cleaner file output.
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsurePrototypesDir creates the ~/prototypes directory if it does not exist.
func EnsurePrototypesDir() error {
	path := PrototypesPath()
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create prototypes directory: %w", err)
	}
	return nil
}
