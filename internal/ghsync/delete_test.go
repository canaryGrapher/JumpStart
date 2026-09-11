package ghsync

import (
	"context"
	"testing"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

func TestRemovedLinkedItemIDs(t *testing.T) {
	before := []model.Task{
		{ID: "keep", GitHub: &model.GitHubLink{ItemID: "i-keep"}},
		{ID: "gone", GitHub: &model.GitHubLink{ItemID: "i-gone"}},
		{ID: "local-only"},
	}
	after := []model.Task{
		{ID: "keep", GitHub: &model.GitHubLink{ItemID: "i-keep"}},
	}

	got := RemovedLinkedItemIDs(before, after)
	if len(got) != 1 || got[0] != "i-gone" {
		t.Fatalf("RemovedLinkedItemIDs = %v, want [i-gone]", got)
	}
}

func TestMergePendingDeletesDedupes(t *testing.T) {
	got := MergePendingDeletes([]string{"a", "b"}, []string{"b", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(got), got)
	}
}

func TestSyncDropsLocalTaskWhenBoardRowIsGone(t *testing.T) {
	fake := &fakeSyncClient{
		project: &github.Project{ID: "p1", Title: "Board"},
		items:   nil, // row deleted on GitHub
	}
	engine := NewEngine(fake)
	tasks := []model.Task{{
		ID:    "t1",
		Title: "Still here locally",
		GitHub: &model.GitHubLink{ItemID: "i1"},
	}}
	cfg := &model.GitHubSync{Enabled: true, ProjectID: "p1", Direction: "both"}

	out, res, err := engine.Sync(context.Background(), tasks, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected local task dropped after remote delete, got %+v", out)
	}
	if res.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", res.Deleted)
	}
}

func TestSyncDoesNotRecreatePendingDeletes(t *testing.T) {
	fake := &fakeSyncClient{
		project: &github.Project{ID: "p1"},
		items: []github.Item{{
			ID:          "i-pending",
			Title:       "Should stay deleted",
			ContentType: "DraftIssue",
		}},
	}
	engine := NewEngine(fake)
	cfg := &model.GitHubSync{
		Enabled:        true,
		ProjectID:      "p1",
		Direction:      "both",
		PendingDeletes: []string{"i-pending"},
	}

	out, _, err := engine.Sync(context.Background(), nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("pending delete must not be re-imported, got %+v", out)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "i-pending" {
		t.Fatalf("expected DeleteItem(i-pending), got %v", fake.deleted)
	}
	if len(cfg.PendingDeletes) != 0 {
		t.Errorf("successful deletes should clear PendingDeletes, got %v", cfg.PendingDeletes)
	}
}

func TestSyncPushOnlyKeepsLocalWhenRemoteGone(t *testing.T) {
	fake := &fakeSyncClient{
		project: &github.Project{ID: "p1"},
		items:   nil,
	}
	engine := NewEngine(fake)
	tasks := []model.Task{{
		ID:     "t1",
		Title:  "Local work",
		GitHub: &model.GitHubLink{ItemID: "i1"},
	}}
	cfg := &model.GitHubSync{Enabled: true, ProjectID: "p1", Direction: "push"}

	out, _, err := engine.Sync(context.Background(), tasks, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("push-only should keep the local task, got %d", len(out))
	}
}

// fakeSyncClient covers the Sync read/delete path; push methods panic if used.
type fakeSyncClient struct {
	project *github.Project
	items   []github.Item
	deleted []string
}

func (f *fakeSyncClient) GetProject(context.Context, string) (*github.Project, error) {
	return f.project, nil
}
func (f *fakeSyncClient) ListItems(context.Context, string) ([]github.Item, error) {
	return f.items, nil
}
func (f *fakeSyncClient) DeleteItem(_ context.Context, _, itemID string) error {
	f.deleted = append(f.deleted, itemID)
	// Remove from items so a later ListItems in the same test stays consistent.
	kept := f.items[:0]
	for _, it := range f.items {
		if it.ID != itemID {
			kept = append(kept, it)
		}
	}
	f.items = kept
	return nil
}
func (f *fakeSyncClient) UpdateContent(context.Context, string, string, string, string) error {
	panic("unexpected UpdateContent")
}
func (f *fakeSyncClient) CreateIssue(context.Context, string, string, string) (*github.CreatedIssue, error) {
	panic("unexpected CreateIssue")
}
func (f *fakeSyncClient) AddContentItem(context.Context, string, string) (string, error) {
	panic("unexpected AddContentItem")
}
func (f *fakeSyncClient) AddDraftItem(context.Context, string, string, string) (string, error) {
	panic("unexpected AddDraftItem")
}
func (f *fakeSyncClient) SetSingleSelect(context.Context, string, string, string, string) error {
	panic("unexpected SetSingleSelect")
}
func (f *fakeSyncClient) SetValue(context.Context, string, string, github.Field, github.ItemFieldValue) error {
	panic("unexpected SetValue")
}
func (f *fakeSyncClient) SetNumber(context.Context, string, string, string, float64) error {
	panic("unexpected SetNumber")
}
func (f *fakeSyncClient) SetIssueAssignees(context.Context, string, []string) error {
	panic("unexpected SetIssueAssignees")
}
func (f *fakeSyncClient) ListAssignableUsers(context.Context, string) ([]github.User, error) {
	panic("unexpected ListAssignableUsers")
}
func (f *fakeSyncClient) LookupUser(context.Context, string) (*github.User, error) {
	panic("unexpected LookupUser")
}
