package taskjson

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"devdeck/internal/model"
)

var now = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func sample() model.Project {
	return model.Project{
		ID: "p1", Name: "Demo", Root: "/work/demo", Description: "desc",
		Processes: []model.Process{{ID: "web", Name: "web", Env: map[string]string{"API_KEY": "s3cret", "PORT": "3000"}}},
		Sprints:   []model.Sprint{{ID: "s1", Name: "Sprint 1"}},
		Tasks: []model.Task{
			{ID: "t1", Title: "Ship", Status: "todo", Type: "task", DueDate: "2026-10-20", CreatedAt: 1,
				Acceptance:  []model.Subtask{{ID: "a1", Title: "Works", Done: true}},
				Attachments: []model.Attachment{{ID: "f1", Name: "spec.pdf", File: "f1.pdf"}},
				GitHub:      &model.GitHubLink{ItemID: "PVTI_1"}},
			{ID: "t2", Title: "Story", Status: "backlog", Type: "story", CreatedAt: 1},
			{ID: "t3", Title: "Child", Status: "todo", Type: "task", ParentID: "t2", CreatedAt: 1},
		},
	}
}

func TestExportRedactsEnvUnlessAsked(t *testing.T) {
	data, err := Export(sample(), false, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "s3cret") || !strings.Contains(string(data), Redacted) {
		t.Fatalf("env not redacted:\n%s", data)
	}
	var doc Document
	json.Unmarshal(data, &doc)
	if doc.Format != Format || doc.Version != 1 || len(doc.Project.Tasks) != 3 || doc.Project.Tasks[0].DueDate != "2026-10-20" {
		t.Errorf("doc = %+v", doc)
	}
	if p := sample(); p.Processes[0].Env["API_KEY"] != "s3cret" {
		t.Error("export must not modify the source project")
	}
	full, _ := Export(sample(), true, now)
	if !strings.Contains(string(full), "s3cret") {
		t.Error("includeEnv should keep values")
	}
}

func TestRoundTripIsANoOp(t *testing.T) {
	data, _ := Export(sample(), false, now)
	out, pv, err := Plan(sample(), data, ModeMerge, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Added)+len(pv.Updated)+len(pv.Removed)+len(pv.Project) != 0 || pv.Unchanged != 3 {
		t.Errorf("unchanged export should change nothing: %+v", pv)
	}
	if out.Processes[0].Env["API_KEY"] != "s3cret" {
		t.Error("redacted env in the JSON must never overwrite real values")
	}
}

func TestMergeUpdatesAddsAndKeepsFilesAndGitHub(t *testing.T) {
	data, _ := Export(sample(), false, now)
	var doc Document
	json.Unmarshal(data, &doc)
	doc.Project.Description = "new desc"
	doc.Project.Tasks[0].Status = "done"
	doc.Project.Tasks[0].Attachments = nil // even if the editor drops them
	doc.Project.Tasks[0].GitHub = nil
	doc.Project.Tasks = append(doc.Project.Tasks, model.Task{Title: "Brand new", Status: "inprogress", SprintID: "s9"},
		model.Task{ID: "x9", Title: "New child", ParentID: "x8"})
	doc.Project.Sprints = append(doc.Project.Sprints, model.Sprint{ID: "s9", Name: "Sprint 9"})
	edited, _ := json.Marshal(doc)

	out, pv, err := Plan(sample(), edited, ModeMerge, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Updated) != 1 || pv.Updated[0].ID != "t1" || strings.Join(pv.Updated[0].Fields, ",") != "done,status" {
		t.Errorf("updated = %+v", pv.Updated)
	}
	if strings.Join(pv.Added, ",") != "Brand new,New child" || strings.Join(pv.Project, ",") != "description" || strings.Join(pv.Sprints, ",") != "Sprint 9" {
		t.Errorf("preview = %+v", pv)
	}
	t1 := out.Tasks[0]
	if !t1.Done || len(t1.Attachments) != 1 || t1.GitHub == nil || t1.CreatedAt != 1 || t1.UpdatedAt != now.UnixMilli() {
		t.Errorf("t1 = %+v", t1)
	}
	var added model.Task
	for _, x := range out.Tasks {
		if x.Title == "Brand new" {
			added = x
		}
	}
	if added.ID == "" || added.SprintID == "" || added.SprintID == "s9" && len(out.Sprints) != 2 {
		t.Errorf("new task = %+v sprints=%+v", added, out.Sprints)
	}
	if len(pv.Warnings) == 0 || !strings.Contains(strings.Join(pv.Warnings, " "), "missing parent") {
		t.Errorf("dangling parent should be reported: %v", pv.Warnings)
	}
	if len(sample().Tasks) != 3 {
		t.Error("source modified")
	}
}

func TestReplaceRemovesAbsentTasksAndFixesParents(t *testing.T) {
	data := []byte(`{"tasks":[{"id":"t1","title":"Ship","status":"todo","type":"task","dueDate":"2026-10-20","createdAt":1,"acceptance":[{"id":"a1","title":"Works","done":true}]},{"id":"t3","title":"Child","status":"todo","type":"task","parentId":"t2"}]}`)
	out, pv, err := Plan(sample(), data, ModeReplace, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(pv.Removed, ",") != "Story" || len(out.Tasks) != 2 {
		t.Errorf("removed = %v tasks=%d", pv.Removed, len(out.Tasks))
	}
	for _, x := range out.Tasks {
		if x.ID == "t3" && x.ParentID != "" {
			t.Error("child of a removed story must become top-level")
		}
	}
	// Merge with the same input removes nothing.
	_, pv, _ = Plan(sample(), data, ModeMerge, now)
	if len(pv.Removed) != 0 {
		t.Errorf("merge removed tasks: %v", pv.Removed)
	}
}

func TestValidationRejectsWithoutChanging(t *testing.T) {
	cases := map[string]string{
		`{"tasks":[{"title":"x","status":"nonexistent"}]}`:                         "not a column",
		`{"tasks":[{"title":"x","dueDate":"next week"}]}`:                          "YYYY-MM-DD",
		`{"tasks":[{"title":"x","links":[{"url":"javascript:alert(1)"}]}]}`:        "only http",
		`{"tasks":[{"title":"x","subtasks":[{"title":" "}]}]}`:                     "empty checklist",
		`{"tasks":[{"title":""}]}`:                                                 "no title",
		`{"tasks":[{"id":"t1","title":"a"},{"id":"t1","title":"b"}]}`:              "twice",
		`{not json`:                                                                "not valid JSON",
		`{"project":{"name":"Demo","quarters":[{"start":"01-01","end":"03-31"}]}}`: "quarters",
	}
	for in, want := range cases {
		out, _, err := Plan(sample(), []byte(in), ModeMerge, now)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want %q", in, err, want)
		}
		if len(out.Tasks) != 3 {
			t.Errorf("%s: project changed on error", in)
		}
	}
	if _, _, err := Plan(sample(), []byte(`{}`), "upsert", now); err == nil {
		t.Error("unknown mode should fail")
	}
}

func TestProjectFieldsThatAreNotImported(t *testing.T) {
	data := []byte(`{"project":{"id":"p1","name":"Demo","root":"/elsewhere","description":"desc","columns":[{"id":"x","label":"X"}],"processes":[]}}`)
	out, pv, err := Plan(sample(), data, ModeMerge, now)
	if err != nil {
		t.Fatal(err)
	}
	if out.Root != "/work/demo" || len(out.Processes) != 1 || out.Columns != nil {
		t.Errorf("root/processes/columns must be untouched: %+v", out)
	}
	if len(pv.Warnings) != 2 {
		t.Errorf("warnings = %v", pv.Warnings)
	}
}

func TestPreviewListsAreNeverNull(t *testing.T) {
	_, pv, err := Plan(sample(), []byte(`{"tasks":[]}`), ModeMerge, now)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(pv)
	if strings.Contains(string(data), "null") {
		t.Errorf("preview has null lists: %s", data)
	}
}
