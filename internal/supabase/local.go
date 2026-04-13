package supabase

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// LocalStatus holds the connection details returned by a local Supabase instance.
type LocalStatus struct {
	APIURL         string `json:"API URL"`
	GraphQLURL     string `json:"GraphQL URL"`
	DBURL          string `json:"DB URL"`
	StudioURL      string `json:"Studio URL"`
	InbucketURL    string `json:"Inbucket URL"`
	JWTSecret      string `json:"JWT secret"`
	AnonKey        string `json:"anon key"`
	ServiceRoleKey string `json:"service_role key"`
}

// CheckCLI verifies that the Supabase CLI is installed.
func CheckCLI() error {
	if _, err := exec.LookPath("supabase"); err != nil {
		return fmt.Errorf("Supabase CLI is not installed. Install it with: brew install supabase/tap/supabase")
	}
	return nil
}

// Init runs "supabase init" in the project directory if config.toml doesn't already exist.
func Init(projectDir string) error {
	configPath := filepath.Join(projectDir, "supabase", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return nil // already initialized
	}

	cmd := exec.Command("supabase", "init")
	cmd.Dir = projectDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("supabase init failed: %w", err)
	}
	return nil
}

// Start runs "supabase start" in the project directory. This pulls Docker images
// and starts all local Supabase services (Postgres, Auth, Storage, etc.).
func Start(projectDir string) error {
	cmd := exec.Command("supabase", "start")
	cmd.Dir = projectDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("supabase start failed: %w", err)
	}
	return nil
}

// Status runs "supabase status --output json" and returns the parsed connection details.
func Status(projectDir string) (*LocalStatus, error) {
	cmd := exec.Command("supabase", "status", "--output", "json")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("supabase status failed: %w", err)
	}

	var status LocalStatus
	if err := json.Unmarshal(out, &status); err != nil {
		return nil, fmt.Errorf("failed to parse supabase status: %w", err)
	}

	if status.AnonKey == "" || status.ServiceRoleKey == "" {
		return nil, fmt.Errorf("local Supabase returned empty keys — is the instance running?")
	}

	return &status, nil
}

// Stop runs "supabase stop" in the project directory.
func Stop(projectDir string) error {
	cmd := exec.Command("supabase", "stop")
	cmd.Dir = projectDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("supabase stop failed: %w", err)
	}
	return nil
}
