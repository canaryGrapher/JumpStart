package github

import (
	"context"
	"fmt"
	"strings"
)

// ListRepositories returns repositories the token can open issues in,
// most recently pushed first.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var out []Repository
	cursor := ""
	for {
		var resp struct {
			Viewer struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID            string `json:"id"`
						Name          string `json:"name"`
						URL           string `json:"url"`
						NameWithOwner string `json:"nameWithOwner"`
						Owner         struct {
							Login string `json:"login"`
						} `json:"owner"`
					} `json:"nodes"`
				} `json:"repositories"`
			} `json:"viewer"`
		}
		vars := map[string]any{"cursor": nullable(cursor)}
		if err := c.Query(ctx, queryRepositories, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Viewer.Repositories.Nodes {
			out = append(out, Repository{
				ID: n.ID, Name: n.Name, Owner: n.Owner.Login,
				FullName: n.NameWithOwner, URL: n.URL,
			})
		}
		if !resp.Viewer.Repositories.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Viewer.Repositories.PageInfo.EndCursor
	}
}

// GetRepository resolves "owner/name" to its node id.
func (c *Client) GetRepository(ctx context.Context, fullName string) (*Repository, error) {
	owner, name, ok := strings.Cut(strings.TrimSpace(fullName), "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("expected owner/name, got %q", fullName)
	}
	var resp struct {
		Repository *struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			URL           string `json:"url"`
			NameWithOwner string `json:"nameWithOwner"`
			Owner         struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
	}
	vars := map[string]any{"owner": owner, "name": name}
	if err := c.Query(ctx, queryRepoByName, vars, &resp); err != nil {
		return nil, err
	}
	if resp.Repository == nil {
		return nil, fmt.Errorf("repository %s not found", fullName)
	}
	return &Repository{
		ID: resp.Repository.ID, Name: resp.Repository.Name,
		Owner: resp.Repository.Owner.Login, FullName: resp.Repository.NameWithOwner,
		URL: resp.Repository.URL,
	}, nil
}

// CreatedIssue is a freshly opened issue.
type CreatedIssue struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// CreateIssue opens an issue in a repository, for tasks promoted from a
// draft to a real issue.
func (c *Client) CreateIssue(ctx context.Context, repoID, title, body string) (*CreatedIssue, error) {
	var resp struct {
		CreateIssue struct {
			Issue CreatedIssue `json:"issue"`
		} `json:"createIssue"`
	}
	vars := map[string]any{"repoId": repoID, "title": title, "body": body}
	if err := c.Query(ctx, mutationCreateIssue, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.CreateIssue.Issue, nil
}

// ParseRemote turns a git remote URL into "owner/name", handling both
// HTTPS and SSH forms. It returns "" when the remote is not GitHub.
func ParseRemote(remote string) string {
	r := strings.TrimSpace(remote)
	if r == "" {
		return ""
	}
	r = strings.TrimSuffix(r, ".git")
	switch {
	case strings.HasPrefix(r, "git@github.com:"):
		return strings.TrimPrefix(r, "git@github.com:")
	case strings.HasPrefix(r, "ssh://git@github.com/"):
		return strings.TrimPrefix(r, "ssh://git@github.com/")
	case strings.HasPrefix(r, "https://github.com/"):
		return strings.TrimPrefix(r, "https://github.com/")
	case strings.HasPrefix(r, "http://github.com/"):
		return strings.TrimPrefix(r, "http://github.com/")
	}
	return ""
}
