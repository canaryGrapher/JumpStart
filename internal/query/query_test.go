package query

import (
	"strings"
	"testing"
	"time"

	"devdeck/internal/model"
)

var today = time.Date(2026, 10, 7, 9, 0, 0, 0, time.Local)

func tasks() []model.Task {
	return []model.Task{
		{ID: "1", Title: "Overdue bug", Status: "todo", Priority: "high", Type: "bug", SprintID: "s1", Assignee: "Alex, Sam", Labels: []string{"Stripe", "urgent"}, DueDate: "2026-10-01", Acceptance: []model.Subtask{{}}},
		{ID: "2", Title: "Done late", Status: "done", Done: true, DueDate: "2026-09-01", Subtasks: []model.Subtask{{}}},
		{ID: "3", Title: "Story", Status: "inprogress", Priority: "medium", Type: "story", SprintID: "s1", Assignee: "sam", Labels: []string{"stripe"}, DueDate: "2026-10-09", Description: "Checkout revamp"},
		{ID: "4", Title: "Backlog", Status: "backlog", Priority: "low", Assignee: "Ada", Labels: []string{"needs-user-action"}},
	}
}

func ids(t *testing.T, q model.TaskQuery) string {
	t.Helper()
	got, err := Run(q, model.Project{Tasks: tasks()}, nil, today)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, x := range got {
		out = append(out, x.ID)
	}
	return strings.Join(out, ",")
}

func TestQueryMatchesFrontendSemantics(t *testing.T) {
	cases := map[string]struct {
		q    model.TaskQuery
		want string
	}{
		"empty":                {model.TaskQuery{}, "1,2,3,4"},
		"status OR":            {model.TaskQuery{Statuses: []string{"todo", "inprogress"}}, "1,3"},
		"priority none":        {model.TaskQuery{Priorities: []string{"none"}}, "2"},
		"type default task":    {model.TaskQuery{Types: []string{"task"}}, "2,4"},
		"backlog sprint":       {model.TaskQuery{Sprints: []string{""}}, "2,4"},
		"assignee token":       {model.TaskQuery{Assignees: []string{"SAM"}}, "1,3"},
		"unassigned":           {model.TaskQuery{Assignees: []string{"__none__"}}, "2"},
		"label any ci":         {model.TaskQuery{Labels: []string{"STRIPE"}}, "1,3"},
		"acceptance none":      {model.TaskQuery{Acceptance: "none"}, "2,3,4"},
		"subtasks has":         {model.TaskQuery{Subtasks: "has"}, "2"},
		"overdue skips done":   {model.TaskQuery{Overdue: true}, "1"},
		"no due date":          {model.TaskQuery{NoDueDate: true}, "4"},
		"this week":            {model.TaskQuery{DuePreset: "this_week"}, "3"},
		"today (none due)":     {model.TaskQuery{DuePreset: "today"}, ""},
		"this quarter":         {model.TaskQuery{DuePreset: "this_quarter"}, "1,3"},
		"range narrows preset": {model.TaskQuery{DuePreset: "this_year", DueFrom: "2026-10-05"}, "3"},
		"text":                 {model.TaskQuery{Text: "revamp"}, "3"},
		"AND across groups":    {model.TaskQuery{Labels: []string{"stripe"}, Statuses: []string{"todo"}}, "1"},
	}
	for name, c := range cases {
		if got := ids(t, c.q); got != c.want {
			t.Errorf("%s = %q, want %q", name, got, c.want)
		}
	}
	if !Empty(model.TaskQuery{}) || Empty(model.TaskQuery{Overdue: true}) {
		t.Error("Empty is wrong")
	}
	if _, err := Run(model.TaskQuery{DuePreset: "someday"}, model.Project{}, nil, today); err == nil {
		t.Error("bad preset should error")
	}
}

func TestDefaultsAreValidAndDeletableAndRestorable(t *testing.T) {
	dir := t.TempDir()
	list, err := AppFilters(dir)
	if err != nil || len(list) != len(Defaults()) {
		t.Fatalf("seeded %d, %v", len(list), err)
	}
	for _, f := range list {
		if err := Validate(f); err != nil {
			t.Errorf("default %q invalid: %v", f.Name, err)
		}
	}
	if err := DeleteAppFilter(dir, "default-blocked"); err != nil {
		t.Fatal(err)
	}
	list, _ = AppFilters(dir)
	for _, f := range list {
		if f.ID == "default-blocked" {
			t.Fatal("deleted default came back without asking")
		}
	}
	list, _ = RestoreDefaults(dir)
	if len(list) != len(Defaults()) {
		t.Errorf("restore gave %d", len(list))
	}
}

func TestSaveRenameAndValidate(t *testing.T) {
	dir := t.TempDir()
	f, err := SaveAppFilter(dir, model.SavedFilter{Name: "  Mine  ", Query: model.TaskQuery{Overdue: true}})
	if err != nil || f.ID == "" || f.Name != "Mine" || f.Builtin {
		t.Fatalf("save = %+v, %v", f, err)
	}
	f.Name = "Renamed"
	if _, err := SaveAppFilter(dir, f); err != nil {
		t.Fatal(err)
	}
	list, _ := AppFilters(dir)
	n := 0
	for _, x := range list {
		if x.Name == "Renamed" {
			n++
		}
		if x.Name == "Mine" {
			t.Error("rename created a copy")
		}
	}
	if n != 1 {
		t.Errorf("renamed count = %d", n)
	}
	for _, bad := range []model.SavedFilter{
		{Name: ""}, {Name: "Past due"}, {Name: "x", Query: model.TaskQuery{DuePreset: "soon"}},
		{Name: "y", Query: model.TaskQuery{DueFrom: "10/1/2026"}}, {Name: "z", Query: model.TaskQuery{Acceptance: "maybe"}},
	} {
		if _, err := SaveAppFilter(dir, bad); err == nil {
			t.Errorf("%+v should be rejected", bad)
		}
	}
	if err := DeleteAppFilter(dir, "nope"); err == nil {
		t.Error("deleting a missing filter should error")
	}
}
