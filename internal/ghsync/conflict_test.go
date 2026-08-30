package ghsync

import (
	"testing"

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
			name: "both sides edited is a conflict",
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decide(tc.task, tc.remoteUpdated); got != tc.want {
				t.Errorf("decide() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePicksTheNewerEdit(t *testing.T) {
	task := model.Task{UpdatedAt: 500}

	if got := resolve(task, 900); got != sideRemote {
		t.Errorf("newer remote should win, got %v", got)
	}
	if got := resolve(task, 100); got != sideLocal {
		t.Errorf("newer local should win, got %v", got)
	}
	// A dead heat keeps the local copy, because the user is looking at it
	// and a surprise overwrite is worse than a redundant push.
	if got := resolve(task, 500); got != sideLocal {
		t.Errorf("a tie should keep local, got %v", got)
	}
}

func TestClearConflictSetsWatermark(t *testing.T) {
	task := model.Task{
		GitHub: &model.GitHubLink{Conflict: true, Pending: true, RemoteUpdatedAt: 100},
	}
	clearConflict(&task, 400, 900)

	if task.GitHub.Conflict || task.GitHub.Pending {
		t.Error("a clean reconcile should clear both flags")
	}
	if task.GitHub.SyncedAt != 900 {
		t.Errorf("SyncedAt = %d, want 900", task.GitHub.SyncedAt)
	}
	if task.GitHub.RemoteUpdatedAt != 400 {
		t.Errorf("RemoteUpdatedAt = %d, want 400", task.GitHub.RemoteUpdatedAt)
	}
}

func TestClearConflictKeepsRemoteStampWhenUnknown(t *testing.T) {
	// A freshly created remote item has no updatedAt to record yet, and
	// zero must not clobber a stamp we already had.
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
