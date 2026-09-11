package github

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// User is a GitHub account that can be assigned to an issue.
type User struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// ParseAssignees splits the local assignee string (comma-separated
// logins) into a de-duplicated list, preserving first-seen order.
func ParseAssignees(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		login := strings.TrimSpace(p)
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, login)
	}
	return out
}

// FormatAssignees joins logins for storage on Task.Assignee. Sorted so
// pull/push comparisons stay stable across reorderings in the UI.
func FormatAssignees(logins []string) string {
	cleaned := ParseAssignees(strings.Join(logins, ","))
	if len(cleaned) == 0 {
		return ""
	}
	sorted := append([]string(nil), cleaned...)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i]) < strings.ToLower(sorted[j])
	})
	return strings.Join(sorted, ", ")
}

// AssigneesEqual reports whether two login lists name the same people,
// ignoring order and blank entries.
func AssigneesEqual(a, b []string) bool {
	return FormatAssignees(a) == FormatAssignees(b)
}

// ListAssignableUsers returns everyone who can be assigned on issues in
// the given "owner/name" repository.
func (c *Client) ListAssignableUsers(ctx context.Context, fullName string) ([]User, error) {
	owner, name, ok := strings.Cut(strings.TrimSpace(fullName), "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("expected owner/name, got %q", fullName)
	}
	var out []User
	cursor := ""
	for {
		var resp struct {
			Repository *struct {
				AssignableUsers struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID        string `json:"id"`
						Login     string `json:"login"`
						Name      string `json:"name"`
						AvatarURL string `json:"avatarUrl"`
					} `json:"nodes"`
				} `json:"assignableUsers"`
			} `json:"repository"`
		}
		vars := map[string]any{"owner": owner, "name": name, "cursor": nullable(cursor)}
		if err := c.Query(ctx, queryAssignableUsers, vars, &resp); err != nil {
			return nil, err
		}
		if resp.Repository == nil {
			return nil, fmt.Errorf("repository %s not found", fullName)
		}
		for _, n := range resp.Repository.AssignableUsers.Nodes {
			out = append(out, User{
				ID: n.ID, Login: n.Login, Name: n.Name, AvatarURL: n.AvatarURL,
			})
		}
		if !resp.Repository.AssignableUsers.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Repository.AssignableUsers.PageInfo.EndCursor
	}
}

// LookupUser resolves a login to its node id. Used when a stored
// assignee is not in the assignableUsers page we already fetched.
func (c *Client) LookupUser(ctx context.Context, login string) (*User, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, fmt.Errorf("empty login")
	}
	var resp struct {
		User *struct {
			ID        string `json:"id"`
			Login     string `json:"login"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"user"`
	}
	if err := c.Query(ctx, queryUserByLogin, map[string]any{"login": login}, &resp); err != nil {
		return nil, err
	}
	if resp.User == nil {
		return nil, fmt.Errorf("user %q not found", login)
	}
	return &User{
		ID: resp.User.ID, Login: resp.User.Login,
		Name: resp.User.Name, AvatarURL: resp.User.AvatarURL,
	}, nil
}

// SetIssueAssignees replaces the assignee list on an issue. Pass an
// empty slice to clear everyone. Draft issues are not assignable.
func (c *Client) SetIssueAssignees(ctx context.Context, issueID string, assigneeIDs []string) error {
	if issueID == "" {
		return fmt.Errorf("missing issue id")
	}
	if assigneeIDs == nil {
		assigneeIDs = []string{}
	}
	vars := map[string]any{"issueId": issueID, "assigneeIds": assigneeIDs}
	return c.Query(ctx, mutationSetIssueAssignees, vars, nil)
}

// ResolveAssigneeIDs maps logins to node ids using the assignable set,
// falling back to a per-login lookup for anyone missing from that set.
// Logins that cannot be resolved are skipped so a free-typed local name
// cannot block the rest of a sync push.
func (c *Client) ResolveAssigneeIDs(ctx context.Context, repo string, logins []string) ([]string, error) {
	logins = ParseAssignees(strings.Join(logins, ","))
	if len(logins) == 0 {
		return []string{}, nil
	}
	byLogin := map[string]string{}
	if repo != "" {
		users, err := c.ListAssignableUsers(ctx, repo)
		if err == nil {
			for _, u := range users {
				byLogin[strings.ToLower(u.Login)] = u.ID
			}
		}
	}
	ids := make([]string, 0, len(logins))
	for _, login := range logins {
		key := strings.ToLower(login)
		if id := byLogin[key]; id != "" {
			ids = append(ids, id)
			continue
		}
		u, err := c.LookupUser(ctx, login)
		if err != nil {
			continue
		}
		ids = append(ids, u.ID)
	}
	return ids, nil
}
