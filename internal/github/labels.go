package github

import (
	"context"
	"fmt"
	"strings"
)

// Label is a repository label that can be applied to an issue.
type Label struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
}

// ListRepoLabels returns every label defined on the given "owner/name"
// repository.
func (c *Client) ListRepoLabels(ctx context.Context, fullName string) ([]Label, error) {
	owner, name, ok := strings.Cut(strings.TrimSpace(fullName), "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("expected owner/name, got %q", fullName)
	}
	var out []Label
	cursor := ""
	for {
		var resp struct {
			Repository *struct {
				Labels struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						Color       string `json:"color"`
						Description string `json:"description"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"repository"`
		}
		vars := map[string]any{"owner": owner, "name": name, "cursor": nullable(cursor)}
		if err := c.Query(ctx, queryRepoLabels, vars, &resp); err != nil {
			return nil, err
		}
		if resp.Repository == nil {
			return nil, fmt.Errorf("repository %s not found", fullName)
		}
		for _, n := range resp.Repository.Labels.Nodes {
			out = append(out, Label{
				ID: n.ID, Name: n.Name, Color: n.Color, Description: n.Description,
			})
		}
		if !resp.Repository.Labels.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Repository.Labels.PageInfo.EndCursor
	}
}

// CreateLabel adds a label to a repository. color is a 6-digit hex without
// the leading '#'; an empty color uses GitHub's default grey.
func (c *Client) CreateLabel(ctx context.Context, repoID, name, color string) (*Label, error) {
	name = strings.TrimSpace(name)
	if repoID == "" || name == "" {
		return nil, fmt.Errorf("repository id and label name are required")
	}
	if color == "" {
		color = "ededed"
	}
	var resp struct {
		CreateLabel struct {
			Label struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Color string `json:"color"`
			} `json:"label"`
		} `json:"createLabel"`
	}
	vars := map[string]any{"repositoryId": repoID, "name": name, "color": color}
	if err := c.Query(ctx, mutationCreateLabel, vars, &resp); err != nil {
		return nil, err
	}
	return &Label{
		ID: resp.CreateLabel.Label.ID,
		Name: resp.CreateLabel.Label.Name,
		Color: resp.CreateLabel.Label.Color,
	}, nil
}

// SetIssueLabels replaces the label list on an issue. Pass an empty
// slice to clear every label. Draft issues are not labelable.
func (c *Client) SetIssueLabels(ctx context.Context, issueID string, labelIDs []string) error {
	if issueID == "" {
		return fmt.Errorf("missing issue id")
	}
	if labelIDs == nil {
		labelIDs = []string{}
	}
	vars := map[string]any{"issueId": issueID, "labelIds": labelIDs}
	return c.Query(ctx, mutationSetIssueLabels, vars, nil)
}

// ResolveLabelIDs maps label names to node ids for a repository. Names
// that do not exist yet are created (neutral grey) so a local tag the
// user typed can still land on GitHub.
func (c *Client) ResolveLabelIDs(ctx context.Context, repo, repoID string, names []string) ([]string, error) {
	names = NormalizeLabels(names)
	if len(names) == 0 {
		return []string{}, nil
	}
	byName := map[string]string{}
	if repo != "" {
		labels, err := c.ListRepoLabels(ctx, repo)
		if err == nil {
			for _, l := range labels {
				byName[strings.ToLower(l.Name)] = l.ID
			}
		}
	}
	ids := make([]string, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(name)
		if id := byName[key]; id != "" {
			ids = append(ids, id)
			continue
		}
		if repoID == "" {
			continue
		}
		created, err := c.CreateLabel(ctx, repoID, name, "")
		if err != nil {
			continue
		}
		byName[strings.ToLower(created.Name)] = created.ID
		ids = append(ids, created.ID)
	}
	return ids, nil
}

// NormalizeLabels trims, de-duplicates (case-insensitive), and drops blanks.
func NormalizeLabels(names []string) []string {
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		key := strings.ToLower(n)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
	}
	return out
}
