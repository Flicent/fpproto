package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const githubBaseURL = "https://api.github.com"

// GitHubUser represents a GitHub user.
type GitHubUser struct {
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GitHubRepo represents a GitHub repository.
type GitHubRepo struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Archived    bool   `json:"archived"`
	Description string `json:"description"`
}

// GitHubRelease represents a GitHub release.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Name    string         `json:"name"`
	Body    string         `json:"body"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents an asset attached to a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubClient is a client for the GitHub REST API.
type GitHubClient struct {
	Token  string
	Org    string
	client *http.Client
}

// NewGitHubClient creates a new GitHubClient.
func NewGitHubClient(token, org string) *GitHubClient {
	return &GitHubClient{
		Token:  token,
		Org:    org,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GitHubClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, githubBaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.client.Do(req)
}

func githubError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	snippet := string(body)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return fmt.Errorf("GitHub error (%d): %s", resp.StatusCode, snippet)
}

// GetUser returns the authenticated GitHub user.
func (c *GitHubClient) GetUser() (*GitHubUser, error) {
	resp, err := c.doRequest(http.MethodGet, "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, githubError(resp)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &user, nil
}

// GetRepoContents fetches a file from a GitHub repo, returning decoded content, SHA, and error.
func (c *GitHubClient) GetRepoContents(repo, path string) ([]byte, string, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/repos/%s/%s/contents/%s", c.Org, repo, path), nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", githubError(resp)
	}

	var result struct {
		Content  string `json:"content"`
		SHA      string `json:"sha"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("decode response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(result.Content)
	if err != nil {
		return nil, "", fmt.Errorf("decode base64 content: %w", err)
	}
	return decoded, result.SHA, nil
}

// CreateRepoFromTemplate creates a new repository from a template repository.
func (c *GitHubClient) CreateRepoFromTemplate(templateRepo, newName string, private bool) error {
	payload := map[string]interface{}{
		"owner":   c.Org,
		"name":    newName,
		"private": private,
	}

	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("/repos/%s/%s/generate", c.Org, templateRepo), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubError(resp)
	}
	return nil
}

// GetRepo returns a repository by name. Returns nil, nil if the repo is not found (404).
func (c *GitHubClient) GetRepo(name string) (*GitHubRepo, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/repos/%s/%s", c.Org, name), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, githubError(resp)
	}

	var repo GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &repo, nil
}

// ListOrgRepos returns all repositories in the organization, handling pagination.
func (c *GitHubClient) ListOrgRepos() ([]GitHubRepo, error) {
	var allRepos []GitHubRepo
	page := 1

	for {
		resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/repos?type=all&per_page=100&page=%d", c.Org, page), nil)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err = githubError(resp)
			resp.Body.Close()
			return nil, err
		}

		var repos []GitHubRepo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		allRepos = append(allRepos, repos...)

		if len(repos) < 100 {
			break
		}
		page++
	}

	return allRepos, nil
}

// ArchiveRepo archives a repository.
func (c *GitHubClient) ArchiveRepo(name string) error {
	payload := map[string]interface{}{
		"archived": true,
	}

	resp, err := c.doRequest(http.MethodPatch, fmt.Sprintf("/repos/%s/%s", c.Org, name), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubError(resp)
	}
	return nil
}

// CreateOrUpdateFile creates or updates a file in a repository.
// If sha is empty, a new file is created. If sha is provided, the existing file is updated.
func (c *GitHubClient) CreateOrUpdateFile(repo, path, message string, content []byte, sha string) error {
	payload := map[string]interface{}{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(content),
	}
	if sha != "" {
		payload["sha"] = sha
	}

	resp, err := c.doRequest(http.MethodPut, fmt.Sprintf("/repos/%s/%s/contents/%s", c.Org, repo, path), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubError(resp)
	}
	return nil
}

// GetLatestRelease returns the latest release for a repository. Returns nil, nil if no release exists (404).
func (c *GitHubClient) GetLatestRelease(repo string) (*GitHubRelease, error) {
	resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("/repos/%s/%s/releases/latest", c.Org, repo), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, githubError(resp)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &release, nil
}
