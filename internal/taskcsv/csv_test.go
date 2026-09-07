package taskcsv

import (
	"bytes"
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
		},
		{
			ID:       "a2",
			Title:    "Child task",
			Type:     "task",
			Status:   "todo",
			ParentID: "a1",
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, in, nil); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	records, err := Decode(&buf)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(records) != 3 { // header + 2
		t.Fatalf("records = %d, want 3", len(records))
	}

	out, res, err := Apply(nil, records, func() string { return "new-id" }, nil)
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
	out, res, err := Apply(existing, records, func() string { return "gen-1" }, func(done, total int) {
		progress = append(progress, [2]int{done, total})
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 || res.Created != 1 || res.Total != 2 {
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

func TestDecodeRejectsMissingTitle(t *testing.T) {
	records, err := Decode(strings.NewReader("id,status\nx,todo\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = Apply(nil, records, func() string { return "x" }, nil)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("err = %v, want missing title", err)
	}
}
