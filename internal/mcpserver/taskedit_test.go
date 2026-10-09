package mcpserver

import (
	"strings"
	"testing"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

func boolp(b bool) *bool { return &b }

func TestMergeChecklistKeepsIDsAndCheckedState(t *testing.T) {
	existing := []model.Subtask{
		{ID: "a", Title: "Write tests", Done: true},
		{ID: "b", Title: "Ship it", Done: false},
	}
	got, err := mergeChecklist(existing, []ChecklistItem{
		{Title: "write TESTS"}, // matched by title, stays checked
		{ID: "b", Title: "Ship it", Done: boolp(true)},
		{Title: "Brand new"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "a" || !got[0].Done || got[0].Title != "write TESTS" {
		t.Errorf("title match lost id/state: %+v", got[0])
	}
	if got[1].ID != "b" || !got[1].Done {
		t.Errorf("explicit done not applied: %+v", got[1])
	}
	if got[2].ID == "" || got[2].Done {
		t.Errorf("new item should get an id and start unchecked: %+v", got[2])
	}
}

func TestMergeChecklistReplacesAndRejectsBlank(t *testing.T) {
	existing := []model.Subtask{{ID: "a", Title: "Old"}}
	got, _ := mergeChecklist(existing, []ChecklistItem{{Title: "New"}})
	if len(got) != 1 || got[0].Title != "New" || got[0].ID == "a" {
		t.Errorf("list should be replaced: %+v", got)
	}
	if _, err := mergeChecklist(nil, []ChecklistItem{{Title: "  "}}); err == nil {
		t.Error("blank title should fail")
	}
	// An empty (non-nil) list clears the checklist.
	if got, _ := mergeChecklist(existing, []ChecklistItem{}); len(got) != 0 {
		t.Errorf("empty list should clear: %+v", got)
	}
}

func TestMergeChecklistDuplicateTitlesDoNotShareAnID(t *testing.T) {
	existing := []model.Subtask{{ID: "a", Title: "Review"}}
	got, _ := mergeChecklist(existing, []ChecklistItem{{Title: "Review"}, {Title: "Review"}})
	if got[0].ID == got[1].ID {
		t.Errorf("duplicate titles reused one id: %+v", got)
	}
}

func TestMergeLinksValidatesAndKeepsIDs(t *testing.T) {
	existing := []model.TaskLink{{ID: "l1", Title: "Docs", URL: "https://example.com/docs"}}
	got, err := mergeLinks(existing, []LinkItem{
		{URL: "example.com/docs", Title: "Docs v2"},
		{URL: "mailto:team@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].ID != "l1" || got[0].Title != "Docs v2" {
		t.Errorf("existing link not matched by URL: %+v", got[0])
	}
	if got[1].ID == "" || got[1].ID == "l1" {
		t.Errorf("new link id = %q", got[1].ID)
	}
	for _, bad := range []string{"javascript:alert(1)", "file:///etc/passwd", ""} {
		if _, err := mergeLinks(nil, []LinkItem{{URL: bad}}); err == nil {
			t.Errorf("link %q should be rejected", bad)
		}
	}
}

func TestApplyClearAndValidate(t *testing.T) {
	task := model.Task{
		Assignee: "a", Description: "d", Priority: "high", SprintID: "s", ParentID: "p",
		StoryPoints: 5, DueDate: "2026-10-10", Milestone: "m", Labels: []string{"x"},
		Subtasks: []model.Subtask{{ID: "1"}}, Acceptance: []model.Subtask{{ID: "2"}},
		Links: []model.TaskLink{{ID: "3", URL: "https://x.io"}}, Title: "keep",
	}
	all := []string{"assignee", "description", "priority", "sprintId", "parentId", "storyPoints",
		"dueDate", "milestone", "labels", "subtasks", "acceptance", "links"}
	if err := validateClear(all); err != nil {
		t.Fatal(err)
	}
	applyClear(&task, all)
	if task.Title != "keep" {
		t.Error("clear must not touch the title")
	}
	if task.Assignee != "" || task.Description != "" || task.Priority != "" || task.SprintID != "" ||
		task.ParentID != "" || task.StoryPoints != 0 || task.DueDate != "" || task.Milestone != "" ||
		task.Labels != nil || task.Subtasks != nil || task.Acceptance != nil || task.Links != nil {
		t.Errorf("fields not cleared: %+v", task)
	}
	if err := validateClear([]string{"title"}); err == nil {
		t.Error("clearing the title must be rejected")
	}
}

func TestValidateDueDate(t *testing.T) {
	for in, ok := range map[string]bool{
		"": true, "2026-10-09": true, "2026-02-30": false, "10/09/2026": false, "tomorrow": false,
	} {
		if err := validateDueDate(in); (err == nil) != ok {
			t.Errorf("validateDueDate(%q) err=%v, want ok=%v", in, err, ok)
		}
	}
}

func TestWithStatusKeepsDoneFlagInStep(t *testing.T) {
	task := model.Task{}
	withStatus(&task, "done")
	if !task.Done || task.Status != "done" {
		t.Errorf("done: %+v", task)
	}
	withStatus(&task, "testing")
	if task.Done || task.Status != "testing" {
		t.Errorf("reopened: %+v", task)
	}
}

func TestAppendNote(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	task := model.Task{Description: "Original text.\n"}
	appendNote(&task, "  Waiting on Support  ", now)
	want := "Original text.\n\nNote (2026-10-09T12:00:00Z): Waiting on Support"
	if task.Description != want {
		t.Errorf("description = %q", task.Description)
	}
	empty := model.Task{}
	appendNote(&empty, "first", now)
	if !strings.HasPrefix(empty.Description, "Note (") {
		t.Errorf("empty description = %q", empty.Description)
	}
}

func sampleTasks() []model.Task {
	return []model.Task{
		{ID: "1", Title: "Overdue bug", Type: "bug", Status: "todo", Priority: "high", Assignee: "Alex, Sam",
			Labels: []string{"stripe", "urgent"}, DueDate: "2026-10-01", SprintID: "s1",
			Acceptance: []model.Subtask{{ID: "a"}}},
		{ID: "2", Title: "Done and late", Type: "task", Status: "done", Done: true, DueDate: "2026-09-01"},
		{ID: "3", Title: "This week", Type: "task", Status: "inprogress", Priority: "medium", DueDate: "2026-10-09",
			Subtasks: []model.Subtask{{ID: "b"}}, ParentID: "9"},
		{ID: "4", Title: "No date", Status: "backlog"},
		{ID: "5", Title: "Next month", Type: "story", Status: "backlog", DueDate: "2026-11-15", Labels: []string{"Stripe"}},
	}
}

func ids(tasks []model.Task, f TaskFilter, today time.Time, project, global []model.QuarterRange) ([]string, error) {
	due, err := f.DueRange(today, project, global)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, t := range tasks {
		if f.Match(t, due, today) {
			out = append(out, t.ID)
		}
	}
	return out, nil
}

func TestTaskFilter(t *testing.T) {
	today := time.Date(2026, 10, 7, 9, 0, 0, 0, time.Local)
	tasks := sampleTasks()
	cases := []struct {
		name string
		f    TaskFilter
		want string
	}{
		{"no filter", TaskFilter{}, "1,2,3,4,5"},
		{"status list", TaskFilter{Status: "todo, inprogress"}, "1,3"},
		{"priority", TaskFilter{Priority: "HIGH"}, "1"},
		{"type defaults to task", TaskFilter{Type: "task"}, "2,3,4"},
		{"sprint", TaskFilter{SprintID: "s1"}, "1"},
		{"backlog sprint", TaskFilter{SprintID: "backlog"}, "2,3,4,5"},
		{"assignee token", TaskFilter{Assignee: "sam"}, "1"},
		{"label any, case-insensitive", TaskFilter{Label: "stripe"}, "1,5"},
		{"parent", TaskFilter{ParentID: "9"}, "3"},
		{"has acceptance", TaskFilter{HasAcceptance: boolp(true)}, "1"},
		{"no subtasks", TaskFilter{HasSubtasks: boolp(false)}, "1,2,4,5"},
		{"overdue skips done", TaskFilter{Overdue: boolp(true)}, "1"},
		{"no due date", TaskFilter{NoDueDate: boolp(true)}, "4"},
		{"this week", TaskFilter{DuePreset: daterange.ThisWeek}, "3"},
		{"next month", TaskFilter{DuePreset: daterange.NextMonth}, "5"},
		{"q4", TaskFilter{DuePreset: daterange.Q4}, "1,3,5"},
		{"explicit range", TaskFilter{DueFrom: "2026-09-15", DueTo: "2026-10-05"}, "1"},
		{"preset narrowed by bound", TaskFilter{DuePreset: daterange.Q4, DueFrom: "2026-10-05"}, "3,5"},
		{"combined", TaskFilter{Status: "todo", Label: "urgent", Overdue: boolp(true)}, "1"},
	}
	for _, c := range cases {
		got, err := ids(tasks, c.f, today, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if strings.Join(got, ",") != c.want {
			t.Errorf("%s = %v, want %s", c.name, got, c.want)
		}
	}
}

func TestTaskFilterUsesProjectQuartersOverGlobal(t *testing.T) {
	today := time.Date(2026, 10, 7, 9, 0, 0, 0, time.Local)
	fiscal := []model.QuarterRange{
		{Start: "10-01", End: "12-31"}, {Start: "01-01", End: "03-31"},
		{Start: "04-01", End: "06-30"}, {Start: "07-01", End: "09-30"},
	}
	// With an Oct-start fiscal year, Q1 is Oct-Dec, so only tasks 1,3,5 match.
	got, _ := ids(sampleTasks(), TaskFilter{DuePreset: daterange.Q1}, today, fiscal, nil)
	if strings.Join(got, ",") != "1,3,5" {
		t.Errorf("project quarters ignored: %v", got)
	}
	// Global config applies when the project has none.
	got, _ = ids(sampleTasks(), TaskFilter{DuePreset: daterange.Q1}, today, nil, fiscal)
	if strings.Join(got, ",") != "1,3,5" {
		t.Errorf("global quarters ignored: %v", got)
	}
}

func TestTaskFilterRejectsBadDates(t *testing.T) {
	today := time.Now()
	for _, f := range []TaskFilter{
		{DuePreset: "someday"}, {DueFrom: "10/01/2026"}, {DueTo: "2026-13-01"},
	} {
		if _, err := f.DueRange(today, nil, nil); err == nil {
			t.Errorf("filter %+v should be rejected", f)
		}
	}
}
