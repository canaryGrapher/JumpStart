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

// Owner is a place a repository can live: the signed-in account itself,
// or an organization it belongs to.
type Owner struct {
	ID        string `json:"id"` // GraphQL node id, needed for createProjectV2(input:{ownerId:...})
	Login     string `json:"login"`
	Type      string `json:"type"` // "user" | "organization"
	AvatarURL string `json:"avatarUrl,omitempty"`
	Name      string `json:"name,omitempty"`
}

// ListOwners returns the signed-in account followed by every organization
// it belongs to, so the "connect GitHub" flow can offer a single list to
// pick where a new repository should live.
func (c *Client) ListOwners(ctx context.Context) ([]Owner, error) {
	var out []Owner
	cursor := ""
	first := true
	for {
		var resp struct {
			Viewer struct {
				ID             string `json:"id"`
				Login          string `json:"login"`
				AvatarURL      string `json:"avatarUrl"`
				Organizations  struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID        string `json:"id"`
						Login     string `json:"login"`
						AvatarURL string `json:"avatarUrl"`
						Name      string `json:"name"`
					} `json:"nodes"`
				} `json:"organizations"`
			} `json:"viewer"`
		}
		vars := map[string]any{"cursor": nullable(cursor)}
		if err := c.Query(ctx, queryViewerOrgs, vars, &resp); err != nil {
			return nil, err
		}
		if first {
			out = append(out, Owner{ID: resp.Viewer.ID, Login: resp.Viewer.Login, Type: "user", AvatarURL: resp.Viewer.AvatarURL})
			first = false
		}
		for _, n := range resp.Viewer.Organizations.Nodes {
			out = append(out, Owner{ID: n.ID, Login: n.Login, Type: "organization", AvatarURL: n.AvatarURL, Name: n.Name})
		}
		if !resp.Viewer.Organizations.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Viewer.Organizations.PageInfo.EndCursor
	}
}

// CreatedRepository is a freshly created GitHub repository, with the
// clone URL the caller needs to wire up the local git remote.
type CreatedRepository struct {
	Repository
	CloneURL string `json:"cloneUrl"`
}

// CreateRepository creates a new repository under owner. When ownerType
// is "organization" it posts to the org's repos endpoint; otherwise it
// creates under the signed-in account.
//
// It is "auto-init: false" (empty, no initial commit) unless
// gitignoreTemplate is set — GitHub only writes a .gitignore file as
// part of an initial commit it makes itself, so requesting one forces
// auto-init on. That is fine for a brand new project with no local
// history yet, but the caller should not offer a gitignore template
// when the local repository already has commits: auto-init's commit
// and the local history would share no common ancestor, so the
// best-effort push GitHubCreateRepository does afterward would fail.
func (c *Client) CreateRepository(ctx context.Context, owner, ownerType, name, description, gitignoreTemplate string, private bool) (*CreatedRepository, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	path := "/user/repos"
	if strings.EqualFold(ownerType, "organization") && owner != "" {
		path = "/orgs/" + owner + "/repos"
	}
	gitignoreTemplate = strings.TrimSpace(gitignoreTemplate)
	body := map[string]any{
		"name":      name,
		"private":   private,
		"auto_init": gitignoreTemplate != "",
	}
	if description = strings.TrimSpace(description); description != "" {
		body["description"] = description
	}
	if gitignoreTemplate != "" {
		body["gitignore_template"] = gitignoreTemplate
	}
	var resp struct {
		ID            int64  `json:"id"`
		NodeID        string `json:"node_id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		HTMLURL       string `json:"html_url"`
		CloneURL      string `json:"clone_url"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := c.restJSON(ctx, "POST", path, body, &resp); err != nil {
		return nil, err
	}
	return &CreatedRepository{
		Repository: Repository{
			ID: resp.NodeID, Name: resp.Name, Owner: resp.Owner.Login,
			FullName: resp.FullName, URL: resp.HTMLURL,
		},
		CloneURL: resp.CloneURL,
	}, nil
}

// SanitizeRepoName turns a local folder name into something GitHub will
// accept as a repository name: letters, digits, dots, dashes and
// underscores only, everything else collapsed to a single dash.
func SanitizeRepoName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if ok {
			b.WriteRune(r)
			lastDash = r == '-'
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "new-repo"
	}
	return out
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

// LinkRepositoryToProject links a Projects v2 board to a repository, so
// the board appears under that repository's own "Projects" tab on
// github.com. Creating a board (CreateProject) only puts it under the
// owning user/org; it stays unlinked from any particular repository
// until this is called. Safe to call more than once; GitHub treats a
// repeat link as a no-op rather than an error.
func (c *Client) LinkRepositoryToProject(ctx context.Context, projectID, repositoryID string) error {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(repositoryID) == "" {
		return fmt.Errorf("project and repository are both required")
	}
	vars := map[string]any{"projectId": projectID, "repositoryId": repositoryID}
	return c.Query(ctx, mutationLinkProjectV2ToRepository, vars, nil)
}
