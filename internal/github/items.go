package github

import (
	"context"
	"strings"
)

type contentNode struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	State     string `json:"state"`
	UpdatedAt string `json:"updatedAt"`
	IssueType *struct {
		Name string `json:"name"`
	} `json:"issueType"`
	Repository *struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Parent *struct {
		Title string `json:"title"`
	} `json:"parent"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer struct {
				Login string `json:"login"`
				Name  string `json:"name"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
}

type itemNode struct {
	ID          string      `json:"id"`
	IsArchived  bool        `json:"isArchived"`
	UpdatedAt   string      `json:"updatedAt"`
	Type        string      `json:"type"`
	Content     contentNode `json:"content"`
	FieldValues struct {
		Nodes []fieldValueNode `json:"nodes"`
	} `json:"fieldValues"`
}

// ListItems walks every row on a board, following pagination until the
// end. Archived rows are skipped: they are off the board as far as the
// local Kanban is concerned.
func (c *Client) ListItems(ctx context.Context, projectID string) ([]Item, error) {
	var out []Item
	cursor := ""
	for {
		var resp struct {
			Node struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []itemNode `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		}
		vars := map[string]any{"id": projectID, "cursor": nullable(cursor)}
		if err := c.Query(ctx, queryProjectItems, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Node.Items.Nodes {
			if n.IsArchived {
				continue
			}
			out = append(out, n.toItem())
		}
		if !resp.Node.Items.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Node.Items.PageInfo.EndCursor
	}
}

func (n itemNode) toItem() Item {
	c := n.Content
	it := Item{
		ID:          n.ID,
		ContentID:   c.ID,
		ContentType: normalizeType(n.Type),
		Number:      c.Number,
		Title:       c.Title,
		Body:        c.Body,
		URL:         c.URL,
		State:       c.State,
		IsArchived:  n.IsArchived,
		UpdatedAt:   pickTime(c.UpdatedAt, n.UpdatedAt),
		Values:      map[string]ItemFieldValue{},
	}
	if c.Repository != nil {
		it.Repo = c.Repository.NameWithOwner
	}
	if c.Milestone != nil {
		it.Milestone = c.Milestone.Title
	}
	if c.Parent != nil {
		it.ParentTitle = c.Parent.Title
	}
	if c.IssueType != nil {
		it.IssueType = c.IssueType.Name
	}
	for _, a := range c.Assignees.Nodes {
		it.Assignees = append(it.Assignees, a.Login)
	}
	for _, l := range c.Labels.Nodes {
		it.Labels = append(it.Labels, l.Name)
	}
	for _, r := range c.ReviewRequests.Nodes {
		if name := firstNonEmpty(r.RequestedReviewer.Login, r.RequestedReviewer.Name); name != "" {
			it.Reviewers = append(it.Reviewers, name)
		}
	}
	for _, fv := range n.FieldValues.Nodes {
		v, ok := fv.toValue()
		if !ok {
			continue
		}
		it.Values[v.FieldID] = v
		// The board's own read-only rollups are easier to consume off the
		// item than out of the field map, so mirror the useful ones.
		switch v.DataType {
		case FieldLinkedPRs:
			it.LinkedPRs = append(it.LinkedPRs, splitDisplay(v.Display)...)
		case FieldSubIssues:
			it.SubIssues = v.Display
		case FieldReviewers:
			if len(it.Reviewers) == 0 {
				it.Reviewers = splitDisplay(v.Display)
			}
		}
	}
	return it
}

// AddDraftItem creates a draft issue row on the board and returns its
// item id. Drafts are the default for a new local task: they need no
// repository and can be promoted to a real issue later.
func (c *Client) AddDraftItem(ctx context.Context, projectID, title, body string) (string, error) {
	var resp struct {
		Add struct {
			ProjectItem struct {
				ID string `json:"id"`
			} `json:"projectItem"`
		} `json:"addProjectV2DraftIssue"`
	}
	vars := map[string]any{"projectId": projectID, "title": title, "body": body}
	if err := c.Query(ctx, mutationAddDraft, vars, &resp); err != nil {
		return "", err
	}
	return resp.Add.ProjectItem.ID, nil
}

// AddContentItem puts an existing issue or pull request on the board.
func (c *Client) AddContentItem(ctx context.Context, projectID, contentID string) (string, error) {
	var resp struct {
		Add struct {
			Item struct {
				ID string `json:"id"`
			} `json:"item"`
		} `json:"addProjectV2ItemById"`
	}
	vars := map[string]any{"projectId": projectID, "contentId": contentID}
	if err := c.Query(ctx, mutationAddItem, vars, &resp); err != nil {
		return "", err
	}
	return resp.Add.Item.ID, nil
}

// DeleteItem removes a row from the board. For an issue-backed row this
// unlinks it from the board and leaves the issue itself alone.
func (c *Client) DeleteItem(ctx context.Context, projectID, itemID string) error {
	vars := map[string]any{"projectId": projectID, "itemId": itemID}
	return c.Query(ctx, mutationDeleteItem, vars, nil)
}

// UpdateContent rewrites the title and body of the draft or issue behind
// a row. contentType decides which mutation applies.
func (c *Client) UpdateContent(ctx context.Context, contentType, contentID, title, body string) error {
	if contentID == "" {
		return nil
	}
	switch normalizeType(contentType) {
	case "DraftIssue":
		vars := map[string]any{"draftId": contentID, "title": title, "body": body}
		return c.Query(ctx, mutationUpdateDraft, vars, nil)
	case "Issue":
		vars := map[string]any{"issueId": contentID, "title": title, "body": body}
		return c.Query(ctx, mutationUpdateIssue, vars, nil)
	}
	// Pull requests are never rewritten from the board.
	return nil
}

func normalizeType(t string) string {
	switch strings.ToUpper(t) {
	case "DRAFT_ISSUE", "DRAFTISSUE":
		return "DraftIssue"
	case "ISSUE":
		return "Issue"
	case "PULL_REQUEST", "PULLREQUEST":
		return "PullRequest"
	}
	return t
}

func pickTime(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func splitDisplay(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ", ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
