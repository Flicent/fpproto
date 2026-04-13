package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const vercelBaseURL = "https://api.vercel.com"

// VercelProject represents a Vercel project.
type VercelProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// URL returns the default Vercel deployment URL for the project.
func (p *VercelProject) URL() string {
	return fmt.Sprintf("https://%s.vercel.app", p.Name)
}

// VercelClient is a client for the Vercel REST API.
type VercelClient struct {
	Token  string
	TeamID string
	client *http.Client
}

// NewVercelClient creates a new VercelClient.
func NewVercelClient(token, teamID string) *VercelClient {
	return &VercelClient{
		Token:  token,
		TeamID: teamID,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *VercelClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, vercelBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.client.Do(req)
}

func vercelError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return fmt.Errorf("Vercel error (%d): %s", resp.StatusCode, snippet)
}

// GetTeam validates the connection by fetching team details.
func (c *VercelClient) GetTeam() error {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/v2/teams/%s", c.TeamID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return vercelError(resp)
	}
	return nil
}

// CreateProject creates a new Vercel project linked to a GitHub repository.
func (c *VercelClient) CreateProject(name, gitRepo, gitOrg string) (*VercelProject, error) {
	payload := map[string]interface{}{
		"name":      name,
		"framework": "nextjs",
		"gitRepository": map[string]string{
			"type": "github",
			"repo": gitOrg + "/" + name,
		},
	}

	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/v1/projects?teamId=%s", c.TeamID), payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, vercelError(resp)
	}

	var project VercelProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

// SetEnvVars sets environment variables on a Vercel project.
// Each variable is set for production, preview, and development targets with encrypted type.
func (c *VercelClient) SetEnvVars(projectID string, envVars map[string]string) error {
	type envVar struct {
		Key    string   `json:"key"`
		Value  string   `json:"value"`
		Target []string `json:"target"`
		Type   string   `json:"type"`
	}

	var batch []envVar
	for key, value := range envVars {
		batch = append(batch, envVar{
			Key:    key,
			Value:  value,
			Target: []string{"production", "preview", "development"},
			Type:   "encrypted",
		})
	}

	// Vercel supports batch creation by posting an array.
	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/v1/projects/%s/env?teamId=%s", projectID, c.TeamID), batch)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Fall back to sending one at a time if batch fails.
		resp.Body.Close()
		for _, ev := range batch {
			r, err := c.doRequest(http.MethodPost, fmt.Sprintf("/v1/projects/%s/env?teamId=%s", projectID, c.TeamID), ev)
			if err != nil {
				return fmt.Errorf("set env var %s: %w", ev.Key, err)
			}
			defer r.Body.Close()

			if r.StatusCode < 200 || r.StatusCode >= 300 {
				return fmt.Errorf("set env var %s: %w", ev.Key, vercelError(r))
			}
		}
	}

	return nil
}

// GetProject returns a Vercel project by name. Returns nil, nil if not found (404).
func (c *VercelClient) GetProject(name string) (*VercelProject, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/v1/projects/%s?teamId=%s", name, c.TeamID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, vercelError(resp)
	}

	var project VercelProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

// DeleteProject deletes a Vercel project by ID. Returns nil if already deleted (404).
func (c *VercelClient) DeleteProject(projectID string) error {
	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("/v1/projects/%s?teamId=%s", projectID, c.TeamID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return vercelError(resp)
	}
	return nil
}
