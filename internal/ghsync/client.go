package ghsync

import (
	"context"

	"devdeck/internal/github"
)

// syncClient is the subset of github.Client the reconcile engine needs.
// Tests supply a fake; production passes *github.Client.
type syncClient interface {
	GetProject(ctx context.Context, projectID string) (*github.Project, error)
	ListItems(ctx context.Context, projectID string) ([]github.Item, error)
	DeleteItem(ctx context.Context, projectID, itemID string) error
	UpdateContent(ctx context.Context, contentType, contentID, title, body string) error
	CreateIssue(ctx context.Context, repoID, title, body string) (*github.CreatedIssue, error)
	AddContentItem(ctx context.Context, projectID, contentID string) (string, error)
	AddDraftItem(ctx context.Context, projectID, title, body string) (*github.CreatedDraft, error)
	ConvertDraftToIssue(ctx context.Context, itemID, repositoryID string) (*github.CreatedIssue, error)
	SetSingleSelect(ctx context.Context, projectID, itemID, fieldID, optionID string) error
	SetValue(ctx context.Context, projectID, itemID string, f github.Field, v github.ItemFieldValue) error
	SetNumber(ctx context.Context, projectID, itemID, fieldID string, n float64) error
	SetIssueAssignees(ctx context.Context, issueID string, assigneeIDs []string) error
	SetIssueLabels(ctx context.Context, issueID string, labelIDs []string) error
	ListAssignableUsers(ctx context.Context, fullName string) ([]github.User, error)
	LookupUser(ctx context.Context, login string) (*github.User, error)
	ListRepoLabels(ctx context.Context, fullName string) ([]github.Label, error)
	CreateLabel(ctx context.Context, repoID, name, color string) (*github.Label, error)
}
