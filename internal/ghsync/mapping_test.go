package ghsync

import (
	"testing"
	"time"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

func selectField(names ...string) github.Field {
	f := github.Field{ID: "f1", Name: "Status", DataType: github.FieldSingleSelect}
	for i, n := range names {
		f.Options = append(f.Options, github.SelectOption{ID: string(rune('a' + i)), Name: n})
	}
	return f
}

func TestBuildStatusMap(t *testing.T) {
	cases := []struct {
		name    string
		options []string
		want    map[string]string // column -> option name
	}{
		{
			name:    "GitHub's default board",
			options: []string{"Todo", "In Progress", "Done"},
			want:    map[string]string{"todo": "Todo", "inprogress": "In Progress", "done": "Done"},
		},
		{
			name:    "a board using its own words",
			options: []string{"Icebox", "Ready", "Doing", "Shipped"},
			want: map[string]string{
				"backlog": "Icebox", "todo": "Ready",
				"inprogress": "Doing", "done": "Shipped",
			},
		},
		{
			name:    "casing and spacing are ignored",
			options: []string{"BACKLOG", "to do", "  In Progress  ", "DONE"},
			want: map[string]string{
				"backlog": "BACKLOG", "todo": "to do",
				"inprogress": "  In Progress  ", "done": "DONE",
			},
		},
		{
			name:    "unrecognized options are left for the user to map",
			options: []string{"Blocked", "Needs Design"},
			want:    map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := selectField(tc.options...)
			byID := map[string]string{}
			for _, o := range f.Options {
				byID[o.ID] = o.Name
			}

			got := BuildStatusMap(f)
			if len(got) != len(tc.want) {
				t.Fatalf("mapped %d columns, want %d: %v", len(got), len(tc.want), got)
			}
			for col, wantName := range tc.want {
				if byID[got[col]] != wantName {
					t.Errorf("column %q mapped to %q, want %q", col, byID[got[col]], wantName)
				}
			}
		})
	}
}

func TestBuildStatusMapNeverReusesAnOption(t *testing.T) {
	// "Done" and "Complete" are both aliases for the done column; only
	// one of them may claim it, and neither may be claimed twice.
	f := selectField("Done", "Complete")
	got := BuildStatusMap(f)

	seen := map[string]bool{}
	for col, id := range got {
		if seen[id] {
			t.Errorf("option %q claimed by more than one column (%q)", id, col)
		}
		seen[id] = true
	}
}

func TestFindStatusField(t *testing.T) {
	text := github.Field{ID: "t", Name: "Notes", DataType: github.FieldText}
	stage := github.Field{ID: "s", Name: "Stage", DataType: github.FieldSingleSelect}
	status := github.Field{ID: "st", Name: "Status", DataType: github.FieldSingleSelect}

	if f, ok := FindStatusField([]github.Field{text, stage, status}); !ok || f.ID != "st" {
		t.Errorf("a field named Status should win, got %+v (ok=%v)", f, ok)
	}
	if f, ok := FindStatusField([]github.Field{text, stage}); !ok || f.ID != "s" {
		t.Errorf("should fall back to the first single-select, got %+v (ok=%v)", f, ok)
	}
	if _, ok := FindStatusField([]github.Field{text}); ok {
		t.Error("a board with no single-select should report none")
	}
}

func TestColumnFor(t *testing.T) {
	m := map[string]string{"todo": "a", "done": "c"}

	if got := columnFor(m, "c"); got != "done" {
		t.Errorf("columnFor(c) = %q, want done", got)
	}
	if got := columnFor(m, "zzz"); got != "" {
		t.Errorf("an unmapped option should return empty, got %q", got)
	}
	if got := columnFor(m, ""); got != "" {
		t.Errorf("no option should return empty, got %q", got)
	}
}

func TestApplyRemoteCopiesBoardState(t *testing.T) {
	cfg := &model.GitHubSync{
		ProjectID:     "p1",
		StatusFieldID: "f1",
		StatusMap:     map[string]string{"inprogress": "opt-doing"},
	}
	item := github.Item{
		ID: "i1", ContentID: "c1", ContentType: "Issue", Number: 42,
		Title: "Fix the thing", Body: "It is broken",
		URL: "https://github.com/o/r/issues/42", State: "OPEN", Repo: "o/r",
		Assignees: []string{"yash"}, Labels: []string{"bug"},
		Milestone: "v2", IssueType: "Bug", UpdatedAt: "2026-08-30T10:00:00Z",
		Values: map[string]github.ItemFieldValue{
			"f1": {FieldID: "f1", DataType: github.FieldSingleSelect, OptionID: "opt-doing"},
		},
	}

	task := model.Task{Title: "old", Status: "todo"}
	if !applyRemote(&task, item, cfg) {
		t.Fatal("applyRemote should report a change")
	}

	if task.Title != "Fix the thing" || task.Description != "It is broken" {
		t.Errorf("content not copied: %+v", task)
	}
	if task.Status != "inprogress" || task.Done {
		t.Errorf("status = %q done = %v, want inprogress and not done", task.Status, task.Done)
	}
	if task.Assignee != "yash" || task.Milestone != "v2" {
		t.Errorf("board columns not copied: %+v", task)
	}
	if task.GitHub == nil || task.GitHub.Number != 42 || task.GitHub.Repo != "o/r" {
		t.Errorf("link not populated: %+v", task.GitHub)
	}
	if task.GitHub.RemoteUpdatedAt == 0 {
		t.Error("remote timestamp should be parsed")
	}

	// A second pass over identical data must be a no-op, otherwise every
	// poll would look like a change and push forever.
	if applyRemote(&task, item, cfg) {
		t.Error("applying the same item twice should report no change")
	}
}

func TestApplyRemoteClosedIssueLandsInDone(t *testing.T) {
	// The board's Status option is unmapped here, so "closed" is the only
	// signal that the work is finished.
	cfg := &model.GitHubSync{StatusFieldID: "f1", StatusMap: map[string]string{}}
	item := github.Item{ID: "i1", Title: "Done thing", State: "CLOSED"}

	task := model.Task{Title: "Done thing", Status: "inprogress"}
	applyRemote(&task, item, cfg)

	if task.Status != "done" || !task.Done {
		t.Errorf("a closed issue should land in done, got %q", task.Status)
	}
}

func TestApplyRemoteLeavesLocalOnlyFieldsAlone(t *testing.T) {
	cfg := &model.GitHubSync{}
	item := github.Item{ID: "i1", Title: "Thing", Body: "plain body"}

	task := model.Task{
		Title:    "Thing",
		SprintID: "sprint-3",
		ParentID: "story-1",
		Subtasks: []model.Subtask{{ID: "s1", Title: "step one"}},
	}
	applyRemote(&task, item, cfg)

	if task.SprintID != "sprint-3" || task.ParentID != "story-1" || len(task.Subtasks) != 1 {
		t.Errorf("sprints, parents, and unmarked-body subtasks must survive: %+v", task)
	}
}

func TestApplyRemotePullsMarkedChecklistsAndStoryPoints(t *testing.T) {
	cfg := &model.GitHubSync{}
	n := 5.0
	body := ComposeBody("Do the thing",
		[]model.Subtask{{Title: "Criterion A"}},
		[]model.Subtask{{Title: "Step 1", Done: true}},
	)
	item := github.Item{
		ID: "i1", Title: "Do the thing", Body: body,
		Values: map[string]github.ItemFieldValue{
			"pts": {FieldID: "pts", FieldName: "Story Points", DataType: github.FieldNumber, Number: &n},
		},
	}
	task := model.Task{Title: "Do the thing", Subtasks: []model.Subtask{{ID: "old", Title: "Step 1"}}}
	if !applyRemote(&task, item, cfg) {
		t.Fatal("expected change")
	}
	if task.Description != "Do the thing" {
		t.Errorf("description = %q", task.Description)
	}
	if len(task.Acceptance) != 1 || task.Acceptance[0].Title != "Criterion A" {
		t.Errorf("acceptance = %+v", task.Acceptance)
	}
	if len(task.Subtasks) != 1 || task.Subtasks[0].ID != "old" || !task.Subtasks[0].Done {
		t.Errorf("subtasks = %+v", task.Subtasks)
	}
	if task.StoryPoints != 5 {
		t.Errorf("story points = %d", task.StoryPoints)
	}
}

func TestStoryPointsField(t *testing.T) {
	num := func(name string) github.Field {
		return github.Field{ID: name, Name: name, DataType: github.FieldNumber}
	}
	for _, name := range []string{"Story Points", "Estimate", "Size", "points"} {
		if _, ok := storyPointsField([]github.Field{num(name)}); !ok {
			t.Errorf("%q should be recognized as a points field", name)
		}
	}
	if _, ok := storyPointsField([]github.Field{num("Budget")}); ok {
		t.Error("an unrelated number field should not be treated as points")
	}
	if _, ok := storyPointsField([]github.Field{{Name: "Story Points", DataType: github.FieldText}}); ok {
		t.Error("a text field must not be used for points")
	}
}

func TestParseTime(t *testing.T) {
	want := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC).UnixMilli()
	if got := parseTime("2026-08-30T10:00:00Z"); got != want {
		t.Errorf("parseTime = %d, want %d", got, want)
	}
	// An offset timestamp must normalize to the same instant, or a board
	// in a non-UTC zone would look permanently newer than it is.
	if got := parseTime("2026-08-30T12:00:00+02:00"); got != want {
		t.Errorf("offset time = %d, want %d", got, want)
	}
	if got := parseTime(""); got != 0 {
		t.Errorf("empty time should be 0, got %d", got)
	}
	if got := parseTime("not a time"); got != 0 {
		t.Errorf("unparseable time should be 0, got %d", got)
	}
}

func TestInferType(t *testing.T) {
	cases := []struct {
		item github.Item
		want string
	}{
		{github.Item{IssueType: "Bug"}, "bug"},
		{github.Item{Labels: []string{"bug"}}, "bug"},
		{github.Item{Labels: []string{"enhancement", "user story"}}, "story"},
		{github.Item{Labels: []string{"chore"}}, "task"},
		{github.Item{}, "task"},
	}
	for _, tc := range cases {
		if got := inferType(tc.item); got != tc.want {
			t.Errorf("inferType(%+v) = %q, want %q", tc.item, got, tc.want)
		}
	}
}
