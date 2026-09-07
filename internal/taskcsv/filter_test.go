package taskcsv

import (
	"testing"

	"devdeck/internal/model"
)

func TestSelectFilter(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Title: "A", Status: "todo", Type: "task", SprintID: "s1", Labels: []string{"api"}, Priority: "high"},
		{ID: "2", Title: "B", Status: "done", Type: "bug", SprintID: "", Labels: []string{"ui"}, Priority: "low"},
		{ID: "3", Title: "C", Status: "todo", Type: "story", SprintID: "s1", Labels: []string{"api", "ui"}, Priority: "high"},
	}

	got := Select(tasks, &Filter{Statuses: []string{"todo"}, Labels: []string{"api"}})
	if len(got) != 2 {
		t.Fatalf("got %d, want 2 (todo+api)", len(got))
	}

	got = Select(tasks, &Filter{SprintIDs: []string{""}})
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("backlog filter = %+v", got)
	}

	got = Select(tasks, nil)
	if len(got) != 3 {
		t.Fatalf("nil filter should keep all, got %d", len(got))
	}
}
