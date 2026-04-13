package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const supabaseBaseURL = "https://api.supabase.com"

// Project represents a Supabase project.
type Project struct {
	ID             string `json:"id"`
	Ref            string `json:"ref"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	Region         string `json:"region"`
	OrganizationID string `json:"organization_id"`
}

// APIKey represents a Supabase API key.
type APIKey struct {
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
}

// SupabaseClient is a client for the Supabase Management API.
type SupabaseClient struct {
	Token  string
	OrgID  string
	client *http.Client
}

// NewSupabaseClient creates a new SupabaseClient.
func NewSupabaseClient(token, orgID string) *SupabaseClient {
	return &SupabaseClient{
		Token:  token,
		OrgID:  orgID,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *SupabaseClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, supabaseBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	return c.client.Do(req)
}

func supabaseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return fmt.Errorf("Supabase error (%d): %s", resp.StatusCode, snippet)
}

// ListProjects returns all projects in the organization.
func (c *SupabaseClient) ListProjects() ([]Project, error) {
	resp, err := c.doRequest(http.MethodGet, "/v1/projects", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, supabaseError(resp)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return projects, nil
}

// CreateProject creates a new Supabase project.
func (c *SupabaseClient) CreateProject(name, dbPass, region string) (*Project, error) {
	payload := map[string]string{
		"name":            name,
		"organization_id": c.OrgID,
		"db_pass":         dbPass,
		"region":          region,
		"plan":            "pro",
	}

	resp, err := c.doRequest(http.MethodPost, "/v1/projects", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, supabaseError(resp)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

// GetProject returns details for a single project by ref.
func (c *SupabaseClient) GetProject(ref string) (*Project, error) {
	resp, err := c.doRequest(http.MethodGet, "/v1/projects/"+ref, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, supabaseError(resp)
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &project, nil
}

// WaitForProject polls GetProject until the project status is ACTIVE_HEALTHY or the timeout is reached.
func (c *SupabaseClient) WaitForProject(ref string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	poll := 5 * time.Second

	for time.Now().Before(deadline) {
		project, err := c.GetProject(ref)
		if err != nil {
			return fmt.Errorf("polling project status: %w", err)
		}
		if project.Status == "ACTIVE_HEALTHY" {
			return nil
		}
		remaining := time.Until(deadline)
		if remaining < poll {
			time.Sleep(remaining)
		} else {
			time.Sleep(poll)
		}
	}
	return fmt.Errorf("timeout waiting for project %s to become ACTIVE_HEALTHY", ref)
}

// DeleteProject deletes a Supabase project by ref. Returns nil if already deleted (404).
func (c *SupabaseClient) DeleteProject(ref string) error {
	resp, err := c.doRequest(http.MethodDelete, "/v1/projects/"+ref, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return supabaseError(resp)
	}
	return nil
}

// GetAPIKeys returns the anon and service_role API keys for a project.
func (c *SupabaseClient) GetAPIKeys(ref string) (anonKey string, serviceRoleKey string, err error) {
	resp, err := c.doRequest(http.MethodGet, "/v1/projects/"+ref+"/api-keys", nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", supabaseError(resp)
	}

	var keys []APIKey
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	for _, k := range keys {
		switch k.Name {
		case "anon":
			anonKey = k.APIKey
		case "service_role":
			serviceRoleKey = k.APIKey
		}
	}

	if anonKey == "" {
		return "", "", fmt.Errorf("anon key not found for project %s", ref)
	}
	if serviceRoleKey == "" {
		return "", "", fmt.Errorf("service_role key not found for project %s", ref)
	}
	return anonKey, serviceRoleKey, nil
}

// RunSQL executes a SQL query against a project's database.
func (c *SupabaseClient) RunSQL(ref, sql string) error {
	payload := map[string]string{
		"query": sql,
	}

	resp, err := c.doRequest(http.MethodPost, "/v1/projects/"+ref+"/database/query", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return supabaseError(resp)
	}
	return nil
}
