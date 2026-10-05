// Board creation for the "create a Projects v2 board" step of the
// GitHub connect flow. GitHub's own "Create project" page in the
// browser offers a gallery of built-in templates (Iterative
// development, Bug tracker, and so on) by copying one of GitHub's own
// internal template projects — that mechanism is not exposed through
// the public GraphQL API, so it cannot be reproduced here. What this
// file offers instead is JumpStart's own small set of starter layouts:
// each one just relabels the new board's default Status field, which is
// the part of "picking a template" that actually matters day to day.
package github

import (
	"context"
	"fmt"
	"strings"
)

// StatusColumn is one column to seed on a freshly created board's
// Status field.
type StatusColumn struct {
	Name  string `json:"name"`
	Color string `json:"color"` // GRAY | BLUE | GREEN | YELLOW | ORANGE | RED | PINK | PURPLE
}

// BoardPreset is one of JumpStart's starter layouts for a new board.
type BoardPreset struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Columns     []StatusColumn `json:"columns"`
}

// boardPresets is the fixed list offered when creating a board. Column
// names are chosen to match ghsync's local-column aliases (see
// internal/ghsync/mapping.go) so a freshly created board's items land on
// the right JumpStart Kanban column with no manual remapping.
var boardPresets = []BoardPreset{
	{
		Key: "basic", Label: "Basic",
		Description: "A simple five-column flow with a Testing stage.",
		Columns: []StatusColumn{
			{Name: "Backlog", Color: "GRAY"},
			{Name: "To Do", Color: "BLUE"},
			{Name: "In Progress", Color: "YELLOW"},
			{Name: "Testing", Color: "PURPLE"},
			{Name: "Done", Color: "GREEN"},
		},
	},
	{
		Key: "iterative", Label: "Iterative development",
		Description: "Plan and work through a prioritized backlog.",
		Columns: []StatusColumn{
			{Name: "Backlog", Color: "GRAY"},
			{Name: "Ready", Color: "BLUE"},
			{Name: "In Progress", Color: "YELLOW"},
			{Name: "Testing", Color: "PURPLE"},
			{Name: "Done", Color: "GREEN"},
		},
	},
	{
		Key: "bugs", Label: "Bug triage",
		Description: "Track incoming bugs from report to resolution.",
		Columns: []StatusColumn{
			{Name: "Triage", Color: "GRAY"},
			{Name: "Planned", Color: "BLUE"},
			{Name: "In Progress", Color: "YELLOW"},
			{Name: "Testing", Color: "PURPLE"},
			{Name: "Resolved", Color: "GREEN"},
		},
	},
}

// BoardPresets lists the starter layouts offered when creating a board.
func BoardPresets() []BoardPreset {
	return append([]BoardPreset(nil), boardPresets...)
}

// BoardPresetByKey looks up one preset by its key.
func BoardPresetByKey(key string) (BoardPreset, bool) {
	for _, p := range boardPresets {
		if p.Key == key {
			return p, true
		}
	}
	return BoardPreset{}, false
}

// CreateProject creates a new Projects v2 board under ownerID (a user or
// organization node id — see Owner.ID from ListOwners).
func (c *Client) CreateProject(ctx context.Context, ownerID, title string) (*Project, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("a project name is required")
	}
	if strings.TrimSpace(ownerID) == "" {
		return nil, fmt.Errorf("owner is required")
	}
	var resp struct {
		CreateProjectV2 struct {
			ProjectV2 struct {
				projectNode
				Owner struct {
					Typename string `json:"__typename"`
					Login    string `json:"login"`
				} `json:"owner"`
			} `json:"projectV2"`
		} `json:"createProjectV2"`
	}
	vars := map[string]any{"ownerId": ownerID, "title": title}
	if err := c.Query(ctx, mutationCreateProjectV2, vars, &resp); err != nil {
		return nil, err
	}
	n := resp.CreateProjectV2.ProjectV2
	ownerType := "user"
	if n.Owner.Typename == "Organization" {
		ownerType = "organization"
	}
	proj := n.projectNode.toProject(n.Owner.Login, ownerType)
	return &proj, nil
}

// ApplyStatusPreset finds a board's Status field and replaces its
// options with the given columns, in order.
func (c *Client) ApplyStatusPreset(ctx context.Context, projectID string, columns []StatusColumn) error {
	if len(columns) == 0 {
		return nil
	}
	board, err := c.GetProject(ctx, projectID)
	if err != nil {
		return err
	}
	var statusFieldID string
	for _, f := range board.Fields {
		if f.DataType == FieldSingleSelect && strings.EqualFold(f.Name, "Status") {
			statusFieldID = f.ID
			break
		}
	}
	if statusFieldID == "" {
		return fmt.Errorf("this board has no Status field to configure")
	}

	options := make([]map[string]any, 0, len(columns))
	for _, col := range columns {
		options = append(options, map[string]any{
			"name": col.Name, "color": col.Color, "description": "",
		})
	}
	vars := map[string]any{"fieldId": statusFieldID, "options": options}
	return c.Query(ctx, mutationUpdateSingleSelectField, vars, nil)
}

// ImportResult reports how many repository items were added to a board.
type ImportResult struct {
	Added   int `json:"added"`
	Skipped int `json:"skipped"`
}

// ImportProgress reports how far ImportRepoItems has got through the
// one-by-one AddContentItem loop, so the UI can show "Importing 3/13…"
// instead of a frozen Working spinner.
type ImportProgress func(done, total int)

// ImportRepoItems adds a repository's open issues and/or pull requests
// to a board as items. One item failing to add (GitHub rejects a handful
// of content types on some boards) is counted as skipped rather than
// aborting the rest of the import. onProgress may be nil.
func (c *Client) ImportRepoItems(ctx context.Context, projectID, owner, repo string, includeIssues, includePRs bool, onProgress ImportProgress) (*ImportResult, error) {
	res := &ImportResult{}
	var contentIDs []string

	if includeIssues {
		issues, err := c.ListIssues(ctx, owner, repo, []string{"OPEN"}, 100)
		if err != nil {
			return res, err
		}
		for _, is := range issues {
			contentIDs = append(contentIDs, is.ID)
		}
	}
	if includePRs {
		prs, err := c.ListPullRequests(ctx, owner, repo, []string{"OPEN"}, 100)
		if err != nil {
			return res, err
		}
		for _, pr := range prs {
			contentIDs = append(contentIDs, pr.ID)
		}
	}

	total := len(contentIDs)
	if onProgress != nil && total > 0 {
		onProgress(0, total)
	}
	for i, id := range contentIDs {
		if _, err := c.AddContentItem(ctx, projectID, id); err != nil {
			res.Skipped++
		} else {
			res.Added++
		}
		if onProgress != nil {
			onProgress(i+1, total)
		}
	}
	return res, nil
}
