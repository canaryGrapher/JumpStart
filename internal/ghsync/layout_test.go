package ghsync

import (
	"strings"
	"testing"

	"devdeck/internal/model"
)

func project() *model.Project {
	return &model.Project{
		Tasks: []model.Task{
			{ID: "1", Status: "backlog"}, {ID: "2", Status: "testing"}, {ID: "3", Status: "testing"}, {ID: "4", Status: "done", Done: true},
		},
		GitHub: &model.GitHubSync{StatusMap: map[string]string{"testing": "opt-qa", "done": "opt-done"}},
	}
}

func TestLayoutReorderRenameFromDefaults(t *testing.T) {
	p := project()
	cols := []model.BoardColumn{
		{ID: "todo", Label: "Ready"}, {ID: "backlog", Label: "Backlog"}, {ID: "inprogress", Label: "Doing"},
		{ID: "testing", Label: "QA"}, {ID: "done", Label: "Done"},
	}
	moved, err := ApplyBoardLayout(p, cols, nil)
	if err != nil || moved != 0 {
		t.Fatalf("moved=%d err=%v", moved, err)
	}
	if p.Columns[0].ID != "todo" || p.Columns[0].Label != "Ready" || p.Columns[0].Order != 0 || p.Columns[3].Label != "QA" {
		t.Errorf("columns = %+v", p.Columns)
	}
	if len(p.GitHub.StatusMap) != 2 {
		t.Errorf("mappings for kept columns must survive: %v", p.GitHub.StatusMap)
	}
}

func TestLayoutDeleteMovesTasksAndDropsMapping(t *testing.T) {
	p := project()
	cols := []model.BoardColumn{{ID: "backlog", Label: "Backlog"}, {ID: "todo", Label: "To Do"}, {ID: "inprogress", Label: "In Progress"}, {ID: "done", Label: "Done"}}
	moved, err := ApplyBoardLayout(p, cols, map[string]string{"testing": "inprogress"})
	if err != nil || moved != 2 {
		t.Fatalf("moved=%d err=%v", moved, err)
	}
	for _, task := range p.Tasks[1:3] {
		if task.Status != "inprogress" || task.Done {
			t.Errorf("task not moved: %+v", task)
		}
	}
	if _, ok := p.GitHub.StatusMap["testing"]; ok {
		t.Error("mapping for the deleted column should be removed")
	}
	if p.GitHub.StatusMap["done"] != "opt-done" {
		t.Error("unrelated mapping changed")
	}
}

func TestLayoutDeletingDoneMovesClearDoneFlag(t *testing.T) {
	p := project()
	cols := []model.BoardColumn{{ID: "backlog", Label: "Backlog"}, {ID: "todo", Label: "To Do"}, {ID: "testing", Label: "Testing"}}
	if _, err := ApplyBoardLayout(p, cols, map[string]string{"done": "testing", "inprogress": "todo"}); err != nil {
		t.Fatal(err)
	}
	if p.Tasks[3].Status != "testing" || p.Tasks[3].Done {
		t.Errorf("done task moved out of done must not stay done: %+v", p.Tasks[3])
	}
}

func TestLayoutAddColumnGetsDerivedID(t *testing.T) {
	p := project()
	cols := append(DefaultBoardColumns(), model.BoardColumn{Label: "Waiting on Vendor"}, model.BoardColumn{Label: "Waiting on Vendor"})
	if _, err := ApplyBoardLayout(p, cols, nil); err != nil {
		t.Fatal(err)
	}
	if n := len(p.Columns); n != 7 {
		t.Fatalf("columns = %d", n)
	}
	if p.Columns[5].ID == "" || p.Columns[5].ID == p.Columns[6].ID {
		t.Errorf("new ids must be derived and unique: %q %q", p.Columns[5].ID, p.Columns[6].ID)
	}
}

func TestLayoutRejectsUnsafeChangesWithoutModifying(t *testing.T) {
	cases := []struct {
		name  string
		cols  []model.BoardColumn
		moves map[string]string
		want  string
	}{
		{"no destination", []model.BoardColumn{{ID: "backlog", Label: "B"}, {ID: "todo", Label: "T"}, {ID: "inprogress", Label: "I"}, {ID: "done", Label: "D"}}, nil, "choose where"},
		{"destination removed", []model.BoardColumn{{ID: "backlog", Label: "B"}, {ID: "done", Label: "D"}}, map[string]string{"testing": "todo"}, "not on the board"},
		{"empty", nil, nil, "at least one"},
		{"duplicate", []model.BoardColumn{{ID: "todo", Label: "A"}, {ID: "TODO", Label: "B"}}, nil, "duplicate"},
		{"blank label", []model.BoardColumn{{ID: "todo", Label: " "}}, nil, "needs a label"},
	}
	for _, c := range cases {
		p := project()
		before := len(p.Tasks)
		_, err := ApplyBoardLayout(p, c.cols, c.moves)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
		if p.Columns != nil || len(p.Tasks) != before || p.Tasks[1].Status != "testing" {
			t.Errorf("%s: project modified on error", c.name)
		}
	}
}
