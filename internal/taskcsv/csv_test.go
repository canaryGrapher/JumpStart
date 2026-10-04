package taskcsv

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"devdeck/internal/model"
)

func TestRoundTrip(t *testing.T) {
	in := []model.Task{
		{
			ID:          "a1",
			Title:       "Ship CSV",
			Type:        "story",
			Status:      "inprogress",
			Priority:    "high",
			Description: "Line one\nLine two",
			Assignee:    "yash",
			Labels:      []string{"bulk", "export"},
			StoryPoints: 5,
			SprintID:    "s1",
			Done:        false,
			Subtasks: []model.Subtask{
				{ID: "st1", Title: "Write encoder", Done: true},
				{ID: "st2", Title: "Write docs", Done: false},
			},
			Acceptance: []model.Subtask{
				{ID: "ac1", Title: "Round-trips labels", Done: false},
			},
			CreatedAt: 1000,
			UpdatedAt: 2000,
			Milestone: "v1",
			IssueType: "Feature",
			ParentKey: "ORG-1",
			Reviewers: []string{"ada", "linus"},
			LinkedPRs: []string{"#12", "#15"},
		},
		{
			ID:       "a2",
			Title:    "Child task",
			Type:     "task",
			Status:   "todo",
			ParentID: "a1",
		},
	}
	sprints := []model.Sprint{{ID: "s1", Name: "Sprint 1"}}

	var buf bytes.Buffer
	if err := Encode(&buf, in, sprints, nil); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	header := strings.Split(strings.Split(buf.String(), "\n")[0], ",")
	if len(header) != len(Header) {
		t.Fatalf("header cols = %d, want %d (%v)", len(header), len(Header), header)
	}
	if !strings.Contains(buf.String(), "Sprint 1") {
		t.Fatalf("expected sprint name in CSV: %s", buf.String())
	}

	records, err := Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(records) != 3 { // header + 2
		t.Fatalf("records = %d, want 3", len(records))
	}

	out, sprintOut, res, err := Apply(nil, sprints, records, ModeAdd, SprintScopeAll, func() string { return "new-id" }, nil)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if res.Created != 2 || res.Updated != 0 {
		t.Fatalf("result = %+v, want 2 created", res)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d", len(out))
	}
	if out[0].Title != "Ship CSV" || out[0].Description != "Line one\nLine two" {
		t.Fatalf("task[0] = %+v", out[0])
	}
	if got := strings.Join(out[0].Labels, ","); got != "bulk,export" {
		t.Fatalf("labels = %q", got)
	}
	if out[1].ParentID != "a1" {
		t.Fatalf("parentId = %q", out[1].ParentID)
	}
	if len(out[0].Subtasks) != 2 || !out[0].Subtasks[0].Done || out[0].Subtasks[1].Done {
		t.Fatalf("subtasks = %+v", out[0].Subtasks)
	}
	if out[0].Subtasks[0].Title != "Write encoder" {
		t.Fatalf("subtask title = %+v", out[0].Subtasks[0])
	}
	if out[0].Milestone != "v1" || out[0].IssueType != "Feature" {
		t.Fatalf("gh fields = milestone=%q issueType=%q", out[0].Milestone, out[0].IssueType)
	}
	if got := strings.Join(out[0].Reviewers, ","); got != "ada,linus" {
		t.Fatalf("reviewers = %q", got)
	}
	if got := strings.Join(out[0].LinkedPRs, ","); got != "#12,#15" {
		t.Fatalf("linkedPrs = %q", got)
	}
	if out[0].SprintID != "s1" {
		t.Fatalf("sprintId = %q, want s1 (matched by name)", out[0].SprintID)
	}
	if len(sprintOut) != 1 {
		t.Fatalf("sprints = %+v", sprintOut)
	}
}

func TestApplyUpdatesExisting(t *testing.T) {
	existing := []model.Task{
		{ID: "keep", Title: "Untouched", Status: "todo"},
		{ID: "edit", Title: "Old", Status: "todo", Type: "task"},
	}
	csv := "id,title,type,status,priority,description,assignee,labels,storyPoints,parentId,sprintId,done\n" +
		"edit,New title,bug,done,high,Body,ada,fix,3,,,true\n" +
		",Brand new,task,backlog,,,,,,,\n"

	records, err := Decode(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}

	var progress [][2]int
	out, _, res, err := Apply(existing, nil, records, ModeAdd, SprintScopeAll, func() string { return "gen-1" }, func(done, total int) {
		progress = append(progress, [2]int{done, total})
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 || res.Created != 1 || res.Total != 2 || res.Removed != 0 {
		t.Fatalf("result = %+v", res)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3 (kept untouched)", len(out))
	}
	if out[0].Title != "Untouched" {
		t.Fatalf("untouched changed: %+v", out[0])
	}
	if out[1].Title != "New title" || out[1].Type != "bug" || !out[1].Done || out[1].Status != "done" {
		t.Fatalf("updated = %+v", out[1])
	}
	if out[2].ID != "gen-1" || out[2].Title != "Brand new" {
		t.Fatalf("created = %+v", out[2])
	}
	if len(progress) == 0 || progress[len(progress)-1] != [2]int{2, 2} {
		t.Fatalf("progress = %v", progress)
	}
}

func TestApplyReplaceDropsMissing(t *testing.T) {
	existing := []model.Task{
		{ID: "keep", Title: "Stay", Status: "todo"},
		{ID: "gone", Title: "Remove me", Status: "todo"},
	}
	csv := "id,title,status\n" +
		"keep,Stay updated,done\n" +
		"fresh,Brand new,todo\n"

	records, err := Decode(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	out, _, res, err := Apply(existing, nil, records, ModeReplace, SprintScopeAll, func() string { return "gen" }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 || res.Created != 1 || res.Removed != 1 {
		t.Fatalf("result = %+v", res)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	ids := map[string]bool{}
	for _, tsk := range out {
		ids[tsk.ID] = true
	}
	if !ids["keep"] || !ids["fresh"] || ids["gone"] {
		t.Fatalf("ids = %v", ids)
	}
	if out[0].Title != "Stay updated" || out[0].Status != "done" {
		t.Fatalf("updated keep = %+v", out[0])
	}
}

func TestApplyCreatesSprintByName(t *testing.T) {
	existing := []model.Task{{ID: "t1", Title: "Old", Status: "todo"}}
	sprints := []model.Sprint{{ID: "s1", Name: "Alpha"}}
	csv := "id,title,sprint\n" +
		"t1,Updated,Alpha\n" +
		"t2,New card,Beta\n" +
		"t3,Backlog card,Backlog\n"

	records, err := Decode(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	newID := func() string {
		n++
		return "gen-" + strconv.Itoa(n)
	}

	out, sprintOut, res, err := Apply(existing, sprints, records, ModeAdd, SprintScopeAll, newID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SprintsCreated != 1 {
		t.Fatalf("sprintsCreated = %d, want 1", res.SprintsCreated)
	}
	if len(sprintOut) != 2 {
		t.Fatalf("sprints = %+v", sprintOut)
	}
	if sprintOut[1].Name != "Beta" {
		t.Fatalf("new sprint = %+v", sprintOut[1])
	}
	byID := map[string]model.Task{}
	for _, tsk := range out {
		byID[tsk.ID] = tsk
	}
	if byID["t1"].SprintID != "s1" {
		t.Fatalf("alpha assign = %q", byID["t1"].SprintID)
	}
	if byID["t2"].SprintID != sprintOut[1].ID {
		t.Fatalf("beta assign = %q want %q", byID["t2"].SprintID, sprintOut[1].ID)
	}
	if byID["t3"].SprintID != "" {
		t.Fatalf("backlog assign = %q", byID["t3"].SprintID)
	}
}

func TestDecodeRejectsMissingTitle(t *testing.T) {
	records, err := Decode(strings.NewReader("id,status\nx,todo\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = Apply(nil, nil, records, ModeAdd, SprintScopeAll, func() string { return "x" }, nil)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("err = %v, want missing title", err)
	}
}

func TestParseMode(t *testing.T) {
	if ParseMode("replace") != ModeReplace {
		t.Fatal("replace")
	}
	if ParseMode("ADD") != ModeAdd {
		t.Fatal("add")
	}
	if ParseMode("") != ModeAdd {
		t.Fatal("default")
	}
}

func TestApplyForcesSprintScopeAndScopedReplace(t *testing.T) {
	existing := []model.Task{
		{ID: "s1-a", Title: "In sprint", Status: "todo", SprintID: "s1"},
		{ID: "s1-b", Title: "Also sprint", Status: "todo", SprintID: "s1"},
		{ID: "other", Title: "Other board", Status: "todo", SprintID: "s2"},
		{ID: "back", Title: "Backlog card", Status: "todo", SprintID: ""},
	}
	csv := "id,title,sprint\n" +
		"s1-a,Updated,IgnoreMe\n" +
		",Brand new,AlsoIgnored\n"

	records, err := Decode(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	out, _, res, err := Apply(existing, []model.Sprint{{ID: "s1", Name: "One"}}, records, ModeReplace, "s1", func() string {
		return "fresh"
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 || res.Created != 1 || res.Removed != 1 {
		t.Fatalf("result = %+v", res)
	}
	byID := map[string]model.Task{}
	for _, tsk := range out {
		byID[tsk.ID] = tsk
	}
	if _, ok := byID["s1-b"]; ok {
		t.Fatalf("s1-b should have been removed from scoped replace: %v", byID)
	}
	if byID["other"].Title != "Other board" || byID["back"].Title != "Backlog card" {
		t.Fatalf("other boards should be untouched: %+v", byID)
	}
	if byID["s1-a"].SprintID != "s1" || byID["fresh"].SprintID != "s1" {
		t.Fatalf("forced sprint not applied: %+v", byID)
	}
	if res.SprintsCreated != 0 {
		t.Fatalf("should not mint sprints when scope is forced: %d", res.SprintsCreated)
	}
}
