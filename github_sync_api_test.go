package main

import (
	"testing"

	"devdeck/internal/model"
)

func TestMergeConcurrentKeepsAnEditMadeDuringTheSync(t *testing.T) {
	const started = 1000

	// The user retitled this card while the pass was in flight. The
	// synced copy carries the pre-edit title, and must not win.
	current := []model.Task{
		{ID: "t1", Title: "renamed while syncing", UpdatedAt: started + 50},
	}
	synced := []model.Task{
		{ID: "t1", Title: "old title", UpdatedAt: 900, GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: started}},
	}

	got, _ := mergeConcurrent(current, synced, started, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got))
	}
	if got[0].Title != "renamed while syncing" {
		t.Errorf("title = %q, want the live edit to survive", got[0].Title)
	}
	if got[0].GitHub == nil || !got[0].GitHub.Pending {
		t.Error("the surviving edit must be marked pending so the next pass pushes it")
	}
	if got[0].GitHub.ItemID != "i1" {
		t.Error("the link from the synced copy should be carried over")
	}
}

func TestMergeConcurrentTakesTheSyncedCopyWhenNothingChanged(t *testing.T) {
	const started = 1000

	current := []model.Task{{ID: "t1", Title: "old title", UpdatedAt: 900}}
	synced := []model.Task{{ID: "t1", Title: "pulled from GitHub", UpdatedAt: 900}}

	got, _ := mergeConcurrent(current, synced, started, nil)
	if got[0].Title != "pulled from GitHub" {
		t.Errorf("title = %q, want the synced copy", got[0].Title)
	}
	if got[0].GitHub != nil && got[0].GitHub.Pending {
		t.Error("an untouched task should not be marked pending")
	}
}

func TestMergeConcurrentKeepsTasksCreatedByTheSync(t *testing.T) {
	const started = 1000

	// A board row JumpStart had never seen becomes a new local task; it
	// has no counterpart in current and must still come through.
	current := []model.Task{{ID: "t1", UpdatedAt: 900}}
	synced := []model.Task{
		{ID: "t1", UpdatedAt: 900},
		{ID: "gh-i2", Title: "from the board", UpdatedAt: started},
	}

	got, _ := mergeConcurrent(current, synced, started, map[string]bool{"t1": true})
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	if got[1].ID != "gh-i2" {
		t.Errorf("new task from the board was dropped: %+v", got)
	}
}

func TestMergeConcurrentDropsTasksTheSyncRemoved(t *testing.T) {
	const started = 1000

	// The synced list is authoritative about what exists. A task only
	// present locally was deleted during the pass and stays deleted.
	current := []model.Task{{ID: "t1"}, {ID: "t2"}}
	synced := []model.Task{{ID: "t1"}}

	got, _ := mergeConcurrent(current, synced, started, nil)
	if len(got) != 1 {
		t.Errorf("expected the synced list to be authoritative, got %d tasks", len(got))
	}
}

func TestMergeConcurrentHonorsLocalDeleteDuringSync(t *testing.T) {
	const started = 1000

	// Sync started with t1 linked; the user deleted it while the pass
	// was in flight. The synced copy must not resurrect it, and the
	// board row id must be queued for deletion.
	current := []model.Task{}
	synced := []model.Task{{
		ID:     "t1",
		Title:  "pulled while deleting",
		GitHub: &model.GitHubLink{ItemID: "i1"},
	}}
	baseline := map[string]bool{"t1": true}

	got, pending := mergeConcurrent(current, synced, started, baseline)
	if len(got) != 0 {
		t.Fatalf("local delete during sync must stick, got %+v", got)
	}
	if len(pending) != 1 || pending[0] != "i1" {
		t.Fatalf("pending deletes = %v, want [i1]", pending)
	}
}

func TestIndexOfProject(t *testing.T) {
	projects := []model.Project{{ID: "a"}, {ID: "b"}}

	if got := indexOfProject(projects, "b"); got != 1 {
		t.Errorf("indexOfProject(b) = %d, want 1", got)
	}
	if got := indexOfProject(projects, "missing"); got != -1 {
		t.Errorf("a missing project should return -1, got %d", got)
	}
	if got := indexOfProject(nil, "a"); got != -1 {
		t.Errorf("an empty list should return -1, got %d", got)
	}
}
