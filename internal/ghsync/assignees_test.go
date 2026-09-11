package ghsync

import (
	"testing"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

func TestApplyRemoteNormalizesAssignees(t *testing.T) {
	task := model.Task{Assignee: "old"}
	item := github.Item{Assignees: []string{"bob", "alice"}}
	cfg := &model.GitHubSync{}

	if !applyRemote(&task, item, cfg) {
		t.Fatal("expected change")
	}
	if task.Assignee != "alice, bob" {
		t.Errorf("Assignee = %q, want sorted join", task.Assignee)
	}
}
