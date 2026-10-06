package ghsync

import (
	"testing"

	"devdeck/internal/model"
)

func TestPreserveGitHubLinksRestoresWipedLink(t *testing.T) {
	before := []model.Task{{
		ID: "a", Title: "old",
		GitHub: &model.GitHubLink{ItemID: "item-1", ContentID: "c1", ContentType: "Issue"},
	}}
	next := []model.Task{{ID: "a", Title: "updated"}}

	got := PreserveGitHubLinks(before, next)
	if got[0].GitHub == nil || got[0].GitHub.ItemID != "item-1" {
		t.Fatalf("link not restored: %+v", got[0].GitHub)
	}
	if got[0].GitHub.ContentID != "c1" {
		t.Errorf("content id should be kept, got %+v", got[0].GitHub)
	}
	if !got[0].GitHub.Pending {
		t.Error("restored link should be pending so the next pass pushes")
	}
	if got[0].Title != "updated" {
		t.Errorf("title should stay from next, got %q", got[0].Title)
	}
}

func TestPreserveGitHubLinksKeepsIncomingLink(t *testing.T) {
	before := []model.Task{{
		ID: "a", GitHub: &model.GitHubLink{ItemID: "item-old"},
	}}
	next := []model.Task{{
		ID: "a", GitHub: &model.GitHubLink{ItemID: "item-new"},
	}}
	got := PreserveGitHubLinks(before, next)
	if got[0].GitHub.ItemID != "item-new" {
		t.Fatalf("incoming link should win, got %+v", got[0].GitHub)
	}
}

func TestPreserveGitHubLinksIgnoresUnknownTasks(t *testing.T) {
	before := []model.Task{{
		ID: "a", GitHub: &model.GitHubLink{ItemID: "item-1"},
	}}
	next := []model.Task{{ID: "b", Title: "new"}}
	got := PreserveGitHubLinks(before, next)
	if got[0].GitHub != nil {
		t.Fatalf("new task must not inherit another card's link: %+v", got[0].GitHub)
	}
}

func TestMergeIncomingTasksKeepsFresherWatermarksByID(t *testing.T) {
	before := []model.Task{{
		ID:        "a",
		Title:     "from store",
		UpdatedAt: 1000,
		GitHub: &model.GitHubLink{
			ItemID: "i1", SyncedAt: 2000, RemoteUpdatedAt: 2500,
		},
	}}
	// Stale modal draft: newer title, but older sync stamps.
	next := []model.Task{{
		ID:        "a",
		Title:     "edited in modal",
		UpdatedAt: 3000,
		GitHub: &model.GitHubLink{
			ItemID: "i1", SyncedAt: 500, RemoteUpdatedAt: 500,
		},
	}}

	got := MergeIncomingTasks(before, next)
	if got[0].Title != "edited in modal" {
		t.Fatalf("content should come from the edit, got %q", got[0].Title)
	}
	if got[0].GitHub.SyncedAt != 2000 || got[0].GitHub.RemoteUpdatedAt != 2500 {
		t.Fatalf("stale save rewound watermarks: %+v", got[0].GitHub)
	}
	if !got[0].GitHub.Pending {
		t.Fatal("content edit after reconcile must be pending so the next pass pushes")
	}
}

func TestMergeIncomingTasksDoesNotTouchOtherIDs(t *testing.T) {
	before := []model.Task{
		{ID: "a", Title: "A", GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 2000}},
		{ID: "b", Title: "B", GitHub: &model.GitHubLink{ItemID: "i2", SyncedAt: 2000}},
	}
	next := []model.Task{
		{ID: "a", Title: "A2", UpdatedAt: 3000, GitHub: &model.GitHubLink{ItemID: "i1", SyncedAt: 2000}},
		{ID: "b", Title: "B-stale", UpdatedAt: 900, GitHub: &model.GitHubLink{ItemID: "i2", SyncedAt: 100}},
	}

	got := MergeIncomingTasks(before, next)
	if got[1].GitHub.SyncedAt != 2000 {
		t.Fatalf("task b watermarks clobbered: %+v", got[1].GitHub)
	}
	if got[1].Title != "B-stale" {
		t.Fatalf("title for b should still apply by id, got %q", got[1].Title)
	}
}
