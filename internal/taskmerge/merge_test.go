package taskmerge

import (
	"testing"

	"devdeck/internal/model"
)

func TestStaleEditDoesNotWipeFieldsSetElsewhere(t *testing.T) {
	base := model.Task{ID: "t", Title: "Old", Status: "todo", UpdatedAt: 1}
	// Elsewhere (MCP) added a due date, links and criteria after the UI loaded.
	stored := base
	stored.DueDate = "2026-10-16"
	stored.Links = []model.TaskLink{{ID: "l", URL: "https://x.io"}}
	stored.Acceptance = []model.Subtask{{ID: "a", Title: "AC"}}
	stored.UpdatedAt = 5
	// The UI (still on base) renames the task.
	mine := base
	mine.Title = "New"
	mine.UpdatedAt = 9
	got := ThreeWay(base, mine, stored)
	if got.Title != "New" || got.DueDate != "2026-10-16" || len(got.Links) != 1 || len(got.Acceptance) != 1 || got.UpdatedAt != 9 {
		t.Errorf("merge = %+v", got)
	}
}

func TestEditorClearsAndChangesWin(t *testing.T) {
	base := model.Task{ID: "t", Title: "T", DueDate: "2026-10-16", Labels: []string{"a"}, Priority: "high"}
	stored := base
	stored.Assignee = "Sam" // concurrent unrelated change
	mine := base
	mine.DueDate = ""     // cleared in the editor
	mine.Labels = nil     // cleared
	mine.Priority = "low" // changed
	got := ThreeWay(base, mine, stored)
	if got.DueDate != "" || len(got.Labels) != 0 || got.Priority != "low" || got.Assignee != "Sam" {
		t.Errorf("merge = %+v", got)
	}
}

func TestSameFieldBothChangedEditorWins(t *testing.T) {
	base := model.Task{ID: "t", Status: "todo"}
	stored := base
	stored.Status = "testing"
	mine := base
	mine.Status = "done"
	mine.Done = true
	if got := ThreeWay(base, mine, stored); got.Status != "done" || !got.Done {
		t.Errorf("merge = %+v", got)
	}
}

func TestApplyDeletesAddsAndSkipsGhosts(t *testing.T) {
	stored := []model.Task{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}, {ID: "c", Title: "C (added elsewhere)"}}
	base := model.Task{ID: "a", Title: "A"}
	changes := []Change{
		{Base: &base, Task: model.Task{ID: "a", Title: "A2"}},
		{Base: nil, Task: model.Task{ID: "n", Title: "New"}},
		{Base: &model.Task{ID: "x"}, Task: model.Task{ID: "x", Title: "edited but deleted elsewhere"}},
	}
	got := Apply(stored, changes, []string{"b", "missing"})
	titles := []string{}
	for _, t := range got {
		titles = append(titles, t.Title)
	}
	want := []string{"A2", "C (added elsewhere)", "New"}
	if len(titles) != len(want) {
		t.Fatalf("titles = %v", titles)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("titles = %v, want %v", titles, want)
		}
	}
}
