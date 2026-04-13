package auth

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ghUser represents the relevant fields from the GitHub API /user endpoint.
type ghUser struct {
	Email string `json:"email"`
	Login string `json:"login"`
}

// CheckGHInstalled verifies that the GitHub CLI is available on PATH.
func CheckGHInstalled() error {
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GitHub CLI not found. Install it: brew install gh")
	}
	return nil
}

// CheckGHAuth verifies that the user is authenticated with the GitHub CLI.
func CheckGHAuth() error {
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Not logged into GitHub. Run: gh auth login")
	}
	return nil
}

// GetUserEmail retrieves the authenticated user's email from the GitHub API.
// If the email field is empty, it falls back to <login>@users.noreply.github.com.
func GetUserEmail() (string, error) {
	cmd := exec.Command("gh", "api", "user")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}

	var user ghUser
	if err := json.Unmarshal(out, &user); err != nil {
		return "", fmt.Errorf("failed to parse GitHub user JSON: %w", err)
	}

	email := user.Email
	if email == "" {
		email = user.Login + "@users.noreply.github.com"
	}
	return email, nil
}

// GetToken retrieves the current GitHub auth token from the GitHub CLI.
func GetToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureAuth is a convenience function that checks the GitHub CLI is installed
// and authenticated, then returns the user's email and auth token.
// It returns on the first error encountered.
func EnsureAuth() (email string, token string, err error) {
	if err = CheckGHInstalled(); err != nil {
		return "", "", err
	}
	if err = CheckGHAuth(); err != nil {
		return "", "", err
	}
	email, err = GetUserEmail()
	if err != nil {
		return "", "", err
	}
	token, err = GetToken()
	if err != nil {
		return "", "", err
	}
	return email, token, nil
}
