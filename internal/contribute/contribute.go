// Package contribute talks to the JumpStart GitHub repository on behalf of
// the user: creating issues (bug reports and feature requests) and listing
// open issues worth picking up. It uses the GitHub REST API directly with a
// personal access token supplied by the caller.
package contribute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Repo identifies the upstream repository issues are filed against.
type Repo struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// FullName is "owner/repo".
func (r Repo) FullName() string { return r.Owner + "/" + r.Name }

// URL is the repository's web page.
func (r Repo) URL() string { return "https://github.com/" + r.FullName() }

// NewIssueURL is the manual "create an issue" page, used when no token is
// configured so the user can still file the issue in a browser.
func (r Repo) NewIssueURL() string { return r.URL() + "/issues/new" }

// IssueRequest is a single issue to create.
type IssueRequest struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

// Issue is a trimmed-down GitHub issue for display in the app.
type Issue struct {
	Number   int      `json:"number"`
	Title    string   `json:"title"`
	URL      string   `json:"url"`
	Labels   []string `json:"labels"`
	Comments int      `json:"comments"`
	State    string   `json:"state"`
}

// CreateIssue opens an issue on the repository and returns its web URL.
func CreateIssue(repo Repo, token string, req IssueRequest) (string, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return "", fmt.Errorf("an issue title is required")
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("no GitHub token configured")
	}

	payload := map[string]any{"title": title, "body": req.Body}
	if len(req.Labels) > 0 {
		payload["labels"] = req.Labels
	}
	buf, _ := json.Marshal(payload)

	endpoint := fmt.Sprintf("%s/repos/%s/issues", apiBase, repo.FullName())
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("couldn't reach GitHub. Check your connection and try again")
	}
	defer resp.Body.Close()

	var out struct {
		HTMLURL string `json:"html_url"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 300 {
		return "", apiError(resp.StatusCode, out.Message)
	}
	return out.HTMLURL, nil
}

// ListIssues returns open issues, newest first. When label is non-empty only
// issues carrying that label are returned (for example "good first issue").
// This is an unauthenticated-friendly call: a token is used when available to
// raise the rate limit, but is not required.
func ListIssues(repo Repo, token, label string, limit int) ([]Issue, error) {
	if limit <= 0 || limit > 30 {
		limit = 10
	}
	q := url.Values{}
	q.Set("state", "open")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	q.Set("per_page", fmt.Sprint(limit))
	if strings.TrimSpace(label) != "" {
		q.Set("labels", label)
	}

	endpoint := fmt.Sprintf("%s/repos/%s/issues?%s", apiBase, repo.FullName(), q.Encode())
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach GitHub. Check your connection and try again")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var out struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return nil, apiError(resp.StatusCode, out.Message)
	}

	var raw []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		HTMLURL     string `json:"html_url"`
		Comments    int    `json:"comments"`
		State       string `json:"state"`
		PullRequest *struct {
			URL string `json:"url"`
		} `json:"pull_request"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("unexpected response from GitHub")
	}

	issues := make([]Issue, 0, len(raw))
	for _, r := range raw {
		// The issues endpoint also returns pull requests; skip them.
		if r.PullRequest != nil {
			continue
		}
		labels := make([]string, 0, len(r.Labels))
		for _, l := range r.Labels {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{
			Number:   r.Number,
			Title:    r.Title,
			URL:      r.HTMLURL,
			Labels:   labels,
			Comments: r.Comments,
			State:    r.State,
		})
	}
	return issues, nil
}

// apiError turns a GitHub failure into a message worth showing a user.
func apiError(status int, message string) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		if strings.Contains(strings.ToLower(message), "rate limit") {
			return fmt.Errorf("GitHub rate limit reached. Try again in a few minutes")
		}
		return fmt.Errorf("GitHub rejected the token. Check that it has the \"repo\" (or public_repo) scope")
	case http.StatusNotFound:
		return fmt.Errorf("repository not found, or the token can't see it")
	case http.StatusGone:
		return fmt.Errorf("issues are disabled on this repository")
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("GitHub rejected the issue: %s", fallback(message, "invalid fields"))
	}
	if message != "" {
		return fmt.Errorf("GitHub error: %s", message)
	}
	return fmt.Errorf("GitHub returned status %d", status)
}

func fallback(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
