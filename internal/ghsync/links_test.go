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
