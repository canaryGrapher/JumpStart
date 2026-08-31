package github

import "context"

// Issue is a lightweight view of a repository issue, for the linked
// repository panel in the task tracker. It is distinct from Item, which
// is a Projects v2 board row and carries board-specific field values.
type Issue struct {
	ID        string   `json:"id"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	State     string   `json:"state"` // OPEN | CLOSED
	Author    string   `json:"author,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Comments  int      `json:"comments"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

// PullRequest is a lightweight view of a repository pull request.
type PullRequest struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	State       string `json:"state"` // OPEN | CLOSED | MERGED
	IsDraft     bool   `json:"isDraft"`
	Author      string `json:"author,omitempty"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	Comments    int    `json:"comments"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreatedPullRequest is a freshly opened pull request.
type CreatedPullRequest struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type issueAuthor struct {
	Login string `json:"login"`
}

type issueComments struct {
	TotalCount int `json:"totalCount"`
}

// ListIssues lists a repository's issues, most recently updated first.
// states filters by "OPEN"/"CLOSED"; an empty slice returns both. limit
// defaults to 30 when zero or negative.
func (c *Client) ListIssues(ctx context.Context, owner, name string, states []string, limit int) ([]Issue, error) {
	if limit <= 0 {
		limit = 30
	}
	var resp struct {
		Repository struct {
			Issues struct {
				Nodes []struct {
					ID        string        `json:"id"`
					Number    int           `json:"number"`
					Title     string        `json:"title"`
					URL       string        `json:"url"`
					State     string        `json:"state"`
					CreatedAt string        `json:"createdAt"`
					UpdatedAt string        `json:"updatedAt"`
					Author    *issueAuthor  `json:"author"`
					Comments  issueComments `json:"comments"`
					Labels    struct {
						Nodes []struct {
							Name string `json:"name"`
						} `json:"nodes"`
					} `json:"labels"`
				} `json:"nodes"`
			} `json:"issues"`
		} `json:"repository"`
	}
	vars := map[string]any{
		"owner": owner, "name": name, "limit": limit,
		"states": nullableStates(states), "cursor": nil,
	}
	if err := c.Query(ctx, queryRepoIssues, vars, &resp); err != nil {
		return nil, err
	}
	out := make([]Issue, 0, len(resp.Repository.Issues.Nodes))
	for _, n := range resp.Repository.Issues.Nodes {
		it := Issue{
			ID: n.ID, Number: n.Number, Title: n.Title, URL: n.URL, State: n.State,
			CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt, Comments: n.Comments.TotalCount,
		}
		if n.Author != nil {
			it.Author = n.Author.Login
		}
		for _, l := range n.Labels.Nodes {
			it.Labels = append(it.Labels, l.Name)
		}
		out = append(out, it)
	}
	return out, nil
}

// ListPullRequests lists a repository's pull requests the same way.
func (c *Client) ListPullRequests(ctx context.Context, owner, name string, states []string, limit int) ([]PullRequest, error) {
	if limit <= 0 {
		limit = 30
	}
	var resp struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					ID          string        `json:"id"`
					Number      int           `json:"number"`
					Title       string        `json:"title"`
					URL         string        `json:"url"`
					State       string        `json:"state"`
					IsDraft     bool          `json:"isDraft"`
					CreatedAt   string        `json:"createdAt"`
					UpdatedAt   string        `json:"updatedAt"`
					BaseRefName string        `json:"baseRefName"`
					HeadRefName string        `json:"headRefName"`
					Author      *issueAuthor  `json:"author"`
					Comments    issueComments `json:"comments"`
				} `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}
	vars := map[string]any{
		"owner": owner, "name": name, "limit": limit,
		"states": nullableStates(states), "cursor": nil,
	}
	if err := c.Query(ctx, queryRepoPullRequests, vars, &resp); err != nil {
		return nil, err
	}
	out := make([]PullRequest, 0, len(resp.Repository.PullRequests.Nodes))
	for _, n := range resp.Repository.PullRequests.Nodes {
		pr := PullRequest{
			ID: n.ID, Number: n.Number, Title: n.Title, URL: n.URL, State: n.State,
			IsDraft: n.IsDraft, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
			BaseRefName: n.BaseRefName, HeadRefName: n.HeadRefName, Comments: n.Comments.TotalCount,
		}
		if n.Author != nil {
			pr.Author = n.Author.Login
		}
		out = append(out, pr)
	}
	return out, nil
}

// CreatePullRequest opens a pull request from an already-pushed branch.
func (c *Client) CreatePullRequest(ctx context.Context, repoID, base, head, title, body string, draft bool) (*CreatedPullRequest, error) {
	var resp struct {
		CreatePullRequest struct {
			PullRequest CreatedPullRequest `json:"pullRequest"`
		} `json:"createPullRequest"`
	}
	vars := map[string]any{
		"repoId": repoID, "base": base, "head": head,
		"title": title, "body": body, "draft": draft,
	}
	if err := c.Query(ctx, mutationCreatePullRequest, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.CreatePullRequest.PullRequest, nil
}

// nullableStates turns an empty state filter into a JSON null, which
// GraphQL needs to mean "no filter" for a list input variable.
func nullableStates(states []string) any {
	if len(states) == 0 {
		return nil
	}
	return states
}
