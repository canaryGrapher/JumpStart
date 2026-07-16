// Package release publishes a GitHub or GitLab release for a project's
// remote repository, using each platform's REST API directly.
package release

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ReleaseOptions describes the release to create.
type ReleaseOptions struct {
	TagName    string `json:"tagName"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ParseRemote extracts the host, owner, and repo name from a git remote
// URL. It supports both HTTPS (https://host/owner/repo.git) and SSH
// (git@host:owner/repo.git) forms.
func ParseRemote(remoteURL string) (host, owner, repo string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	remoteURL = strings.TrimSuffix(remoteURL, ".git")

	if strings.HasPrefix(remoteURL, "git@") {
		// git@host:owner/repo
		rest := strings.TrimPrefix(remoteURL, "git@")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("could not parse remote URL: %s", remoteURL)
		}
		host = parts[0]
		path := strings.Trim(parts[1], "/")
		pathParts := strings.SplitN(path, "/", 2)
		if len(pathParts) != 2 {
			return "", "", "", fmt.Errorf("could not parse owner/repo from remote URL: %s", remoteURL)
		}
		return host, pathParts[0], pathParts[1], nil
	}

	u, perr := url.Parse(remoteURL)
	if perr != nil || u.Host == "" {
		return "", "", "", fmt.Errorf("could not parse remote URL: %s", remoteURL)
	}
	host = u.Host
	path := strings.Trim(u.Path, "/")
	pathParts := strings.SplitN(path, "/", 2)
	if len(pathParts) != 2 {
		return "", "", "", fmt.Errorf("could not parse owner/repo from remote URL: %s", remoteURL)
	}
	return host, pathParts[0], pathParts[1], nil
}

// CreateRelease creates a release on GitHub or GitLab (detected from the
// remote URL's host) and returns the URL of the created release.
func CreateRelease(remoteURL, token string, opts ReleaseOptions) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("no access token configured for this provider")
	}
	if strings.TrimSpace(opts.TagName) == "" {
		return "", fmt.Errorf("tag name is required")
	}
	host, owner, repo, err := ParseRemote(remoteURL)
	if err != nil {
		return "", err
	}
	switch {
	case strings.Contains(host, "github.com"):
		return createGitHubRelease(owner, repo, token, opts)
	case strings.Contains(host, "gitlab.com"):
		return createGitLabRelease(owner, repo, token, opts)
	default:
		return "", fmt.Errorf("unsupported git host: %s (only github.com and gitlab.com are supported)", host)
	}
}

func createGitHubRelease(owner, repo, token string, opts ReleaseOptions) (string, error) {
	body := map[string]any{
		"tag_name":   opts.TagName,
		"name":       opts.Name,
		"body":       opts.Body,
		"draft":      opts.Draft,
		"prerelease": opts.Prerelease,
	}
	buf, _ := json.Marshal(body)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("contacting GitHub: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		HTMLURL string `json:"html_url"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("authentication failed — check your GitHub token")
		}
		if out.Message != "" {
			return "", fmt.Errorf("GitHub error: %s", out.Message)
		}
		return "", fmt.Errorf("GitHub returned %s", resp.Status)
	}
	return out.HTMLURL, nil
}

func createGitLabRelease(owner, repo, token string, opts ReleaseOptions) (string, error) {
	projectPath := url.QueryEscape(owner + "/" + repo)

	// Ensure the tag exists first — GitLab releases require a tag.
	tagURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/repository/tags", projectPath)
	tagBody, _ := json.Marshal(map[string]string{
		"tag_name": opts.TagName,
		"ref":      "HEAD",
	})
	tagReq, err := http.NewRequest(http.MethodPost, tagURL, bytes.NewReader(tagBody))
	if err != nil {
		return "", err
	}
	tagReq.Header.Set("PRIVATE-TOKEN", token)
	tagReq.Header.Set("Content-Type", "application/json")
	tagResp, err := httpClient.Do(tagReq)
	if err != nil {
		return "", fmt.Errorf("contacting GitLab: %w", err)
	}
	tagResp.Body.Close()
	if tagResp.StatusCode == http.StatusUnauthorized || tagResp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("authentication failed — check your GitLab token")
	}
	// 400/409-ish "tag already exists" is fine; anything else that isn't
	// success-ish we still try the release call, since some GitLab
	// versions return odd codes for "already exists".

	relURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/releases", projectPath)
	relBody, _ := json.Marshal(map[string]string{
		"tag_name":    opts.TagName,
		"name":        opts.Name,
		"description": opts.Body,
	})
	relReq, err := http.NewRequest(http.MethodPost, relURL, bytes.NewReader(relBody))
	if err != nil {
		return "", err
	}
	relReq.Header.Set("PRIVATE-TOKEN", token)
	relReq.Header.Set("Content-Type", "application/json")
	relResp, err := httpClient.Do(relReq)
	if err != nil {
		return "", fmt.Errorf("contacting GitLab: %w", err)
	}
	defer relResp.Body.Close()

	var out struct {
		Links struct {
			Self string `json:"self"`
		} `json:"_links"`
		Message any `json:"message"`
	}
	_ = json.NewDecoder(relResp.Body).Decode(&out)
	if relResp.StatusCode >= 300 {
		if relResp.StatusCode == http.StatusUnauthorized || relResp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("authentication failed — check your GitLab token")
		}
		if out.Message != nil {
			return "", fmt.Errorf("GitLab error: %v", out.Message)
		}
		return "", fmt.Errorf("GitLab returned %s", relResp.Status)
	}
	if out.Links.Self != "" {
		return out.Links.Self, nil
	}
	// Fall back to a predictable UI URL.
	return fmt.Sprintf("https://gitlab.com/%s/%s/-/releases/%s", owner, repo, url.PathEscape(opts.TagName)), nil
}
