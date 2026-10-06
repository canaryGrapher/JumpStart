package ghsync

import (
	"testing"
	"time"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

func TestDecide(t *testing.T) {
	// Times are chosen so the ordering is obvious at a glance:
	// 100 is "before the last sync", 300 is "after it".
	const synced = 200

	cases := []struct {
		name          string
		task          model.Task
		remoteUpdated int64
		want          side
	}{
		{
			name: "never linked pushes up",
			task: model.Task{UpdatedAt: 100},
			want: sideLocal,
		},
		{
			name: "linked but never synced pushes up",
			task: model.Task{UpdatedAt: 100, GitHub: &model.GitHubLink{ItemID: "i1"}},
			want: sideLocal,
		},
		{
			name: "quiet on both sides does nothing",
			task: model.Task{
				UpdatedAt: 100,
				GitHub:    &model.GitHubLink{ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced},
			},
			remoteUpdated: synced,
			want:          sideNone,
		},
		{
			name: "local edit since last sync pushes",
			task: model.Task{
				UpdatedAt: 300,
				GitHub:    &model.GitHubLink{ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced},
			},
			remoteUpdated: synced,
			want:          sideLocal,
		},
		{
			name: "remote edit since last sync pulls",
			task: model.Task{
				UpdatedAt: 100,
				GitHub:    &model.GitHubLink{ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced},
			},
			remoteUpdated: 300,
			want:          sideRemote,
		},
		{
			name: "both sides edited is sideBoth for content diff",
			task: model.Task{
				UpdatedAt: 300,
				GitHub:    &model.GitHubLink{ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced},
			},
			remoteUpdated: 400,
			want:          sideBoth,
		},
		{
			name: "a failed earlier push still counts as local",
			task: model.Task{
				UpdatedAt: 100,
				GitHub: &model.GitHubLink{
					ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced, Pending: true,
				},
			},
			remoteUpdated: synced,
			want:          sideLocal,
		},
		{
			name: "ForcePush always pushes even when remote is newer",
			task: model.Task{
				UpdatedAt: 300,
				GitHub: &model.GitHubLink{
					ItemID: "i1", SyncedAt: synced, RemoteUpdatedAt: synced,
					Pending: true, ForcePush: true,
				},
			},
			remoteUpdated: 900,
			want:          sideLocal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decide(tc.task, tc.remoteUpdated); got != tc.want {
				t.Errorf("decide() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDiffTaskSameContentIgnoresNewerRemoteStamp(t *testing.T) {
	task := model.Task{
		Title:  "Same title",
		Status: "todo",
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 200, RemoteUpdatedAt: 200, Pending: true},
	}
	item := github.Item{
		ID:        "i1",
		Title:     "Same title",
		UpdatedAt: time.UnixMilli(900).UTC().Format(time.RFC3339),
	}
	cfg := &model.GitHubSync{}
	if fields := DiffTask(task, item, cfg); len(fields) != 0 {
		t.Fatalf("matching content should not conflict, got %#v", fields)
	}
}

func TestDiffTaskReportsTitleAndStatus(t *testing.T) {
	cfg := &model.GitHubSync{
		StatusFieldID: "sf",
		StatusMap:     map[string]string{"todo": "opt-todo", "done": "opt-done"},
	}
	task := model.Task{
		Title:  "Local title",
		Status: "todo",
		Labels: []string{"bug"},
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 200, RemoteUpdatedAt: 200, Pending: true},
	}
	item := github.Item{
		ID:          "i1",
		Title:       "Remote title",
		ContentType: "Issue",
		Labels:      []string{"bug"},
		Values: map[string]github.ItemFieldValue{
			"sf": {FieldID: "sf", OptionID: "opt-done", OptionName: "Done"},
		},
	}
	fields := DiffTask(task, item, cfg)
	byField := map[string]model.ConflictField{}
	for _, f := range fields {
		byField[f.Field] = f
	}
	if f, ok := byField["title"]; !ok || f.Local != "Local title" || f.Remote != "Remote title" {
		t.Fatalf("title conflict = %#v", f)
	}
	if f, ok := byField["status"]; !ok || f.Local != "todo" || f.Remote != "done" {
		t.Fatalf("status conflict = %#v", f)
	}
}

func TestDiffTaskReportsLabels(t *testing.T) {
	task := model.Task{
		Title:  "T",
		Labels: []string{"a", "b"},
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 1, RemoteUpdatedAt: 1, Pending: true},
	}
	item := github.Item{
		ID: "i1", Title: "T", ContentType: "Issue", Labels: []string{"a", "c"},
	}
	fields := DiffTask(task, item, nil)
	found := false
	for _, f := range fields {
		if f.Field == "labels" {
			found = true
			if f.Local == "" || f.Remote == "" {
				t.Fatalf("labels should show both sides: %#v", f)
			}
		}
	}
	if !found {
		t.Fatal("expected labels conflict")
	}
}

func TestClearConflictSetsWatermark(t *testing.T) {
	task := model.Task{
		GitHub: &model.GitHubLink{
			Conflict: true, Pending: true, ForcePush: true,
			ConflictFields: []model.ConflictField{{Field: "title"}},
			RemoteUpdatedAt: 100,
		},
	}
	clearConflict(&task, 400, 900)

	if task.GitHub.Conflict || task.GitHub.Pending || task.GitHub.ForcePush {
		t.Error("a clean reconcile should clear conflict/pending/forcePush")
	}
	if len(task.GitHub.ConflictFields) != 0 {
		t.Error("ConflictFields should clear")
	}
	if task.GitHub.SyncedAt != 900 {
		t.Errorf("SyncedAt = %d, want 900", task.GitHub.SyncedAt)
	}
	if task.GitHub.RemoteUpdatedAt != 400 {
		t.Errorf("RemoteUpdatedAt = %d, want 400", task.GitHub.RemoteUpdatedAt)
	}
}

func TestClearConflictKeepsRemoteStampWhenUnknown(t *testing.T) {
	task := model.Task{GitHub: &model.GitHubLink{RemoteUpdatedAt: 400}}
	clearConflict(&task, 0, 900)

	if task.GitHub.RemoteUpdatedAt != 400 {
		t.Errorf("RemoteUpdatedAt = %d, want the existing 400", task.GitHub.RemoteUpdatedAt)
	}
}

func TestClearConflictOnUnlinkedTask(t *testing.T) {
	task := model.Task{}
	clearConflict(&task, 100, 900)

	if task.GitHub == nil {
		t.Fatal("clearConflict should create the link rather than panic")
	}
}

func TestMarkKeepLocalForcesPush(t *testing.T) {
	const now int64 = 1000
	task := model.Task{
		UpdatedAt: 500,
		GitHub: &model.GitHubLink{
			ItemID:          "i1",
			Conflict:        true,
			ConflictFields:  []model.ConflictField{{Field: "title", Local: "a", Remote: "b"}},
			SyncedAt:        200,
			RemoteUpdatedAt: 200,
		},
	}
	MarkKeepLocal(&task, now)

	if task.GitHub.Conflict || len(task.GitHub.ConflictFields) != 0 {
		t.Fatal("Overwrite GitHub should clear the conflict badge and reasons")
	}
	if !task.GitHub.Pending || !task.GitHub.ForcePush {
		t.Fatal("Overwrite GitHub should mark pending + ForcePush")
	}
	remote := now + 30_000
	if got := decide(task, remote); got != sideLocal {
		t.Fatalf("follow-up sync should push local, got %v (want sideLocal)", got)
	}
}

func TestMarkDismissConflictQuietsRemote(t *testing.T) {
	const now int64 = 1000
	task := model.Task{
		UpdatedAt: 500,
		GitHub: &model.GitHubLink{
			ItemID: "i1", Conflict: true, SyncedAt: 200, RemoteUpdatedAt: 200,
			ConflictFields: []model.ConflictField{{Field: "title"}},
		},
	}
	MarkDismissConflict(&task, now)
	if task.GitHub.Conflict || task.GitHub.Pending || len(task.GitHub.ConflictFields) != 0 {
		t.Fatal("dismiss should clear conflict, pending, and reasons")
	}
	if decide(task, now) != sideNone {
		t.Fatalf("dismissed remote stamp should be quiet, got %v", decide(task, now))
	}
}

func TestStampAfterPushAllowsContentDiffToQuietEcho(t *testing.T) {
	task := model.Task{
		Title:     "Hello",
		UpdatedAt: 500,
		GitHub: &model.GitHubLink{
			ItemID: "i1", Conflict: true, Pending: true, ForcePush: true,
			RemoteUpdatedAt: 100, SyncedAt: 200,
		},
	}
	const now int64 = 1000
	stampAfterPush(&task, now)

	if task.GitHub.Conflict || task.GitHub.Pending || task.GitHub.ForcePush {
		t.Fatal("stampAfterPush should clear conflict/pending/forcePush")
	}
	if task.GitHub.SyncedAt != now {
		t.Fatalf("SyncedAt = %d, want %d", task.GitHub.SyncedAt, now)
	}
	// Echo with matching content: decide says sideRemote (newer stamp),
	// but DiffTask is empty so the engine stamps quietly.
	echo := now + 500
	if decide(task, echo) != sideRemote {
		t.Fatalf("newer remote stamp alone should look like a pull candidate, got %v", decide(task, echo))
	}
	item := github.Item{ID: "i1", Title: "Hello"}
	if fields := DiffTask(task, item, nil); len(fields) != 0 {
		t.Fatalf("echo with same title must not conflict: %#v", fields)
	}
}

func TestNeedsPendingFlush(t *testing.T) {
	if !needsPendingFlush(model.Task{}) {
		t.Error("unlinked task should flush")
	}
	if !needsPendingFlush(model.Task{GitHub: &model.GitHubLink{ItemID: "i1"}}) {
		t.Error("never-synced linked task should flush")
	}
	if !needsPendingFlush(model.Task{
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 1, Pending: true},
	}) {
		t.Error("pending should flush")
	}
	if needsPendingFlush(model.Task{
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 1, Conflict: true, Pending: true},
	}) {
		t.Error("conflicted without ForcePush should wait on the user")
	}
	if !needsPendingFlush(model.Task{
		GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 1, Conflict: true, ForcePush: true, Pending: true},
	}) {
		t.Error("ForcePush should flush even if conflict flag was leftover")
	}
}
