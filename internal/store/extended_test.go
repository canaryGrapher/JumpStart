package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"devdeck/internal/model"
)

func newTestStore(t *testing.T) *Store {
	dir := t.TempDir()
	return &Store{path: filepath.Join(dir, "config.json")}
}

func rich() []model.Project {
	return []model.Project{{
		ID: "p1", Name: "P",
		Quarters: []model.QuarterRange{{Start: "07-01", End: "09-30"}, {Start: "10-01", End: "12-31"}, {Start: "01-01", End: "03-31"}, {Start: "04-01", End: "06-30"}},
		Tasks: []model.Task{
			{ID: "t1", Title: "A", DueDate: "2026-10-16", Links: []model.TaskLink{{ID: "l1", URL: "https://x.io"}},
				Attachments: []model.Attachment{{ID: "f1", Name: "a.png", File: "f1.png"}}},
			{ID: "t2", Title: "B"},
		},
	}}
}

// simulateOldApp rewrites config.json the way a build without the new
// fields does: decode into a struct that lacks them, re-encode.
func simulateOldApp(t *testing.T, s *Store, edit func(p map[string]any)) {
	t.Helper()
	data, _ := os.ReadFile(s.path)
	var raw []map[string]any
	json.Unmarshal(data, &raw)
	for _, p := range raw {
		delete(p, "quarters")
		for _, tk := range p["tasks"].([]any) {
			m := tk.(map[string]any)
			delete(m, "dueDate")
			delete(m, "links")
			delete(m, "attachments")
		}
		if edit != nil {
			edit(p)
		}
	}
	out, _ := json.Marshal(raw)
	os.WriteFile(s.path, out, 0o644)
}

func TestFieldsDroppedByOlderBuildAreRestored(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(rich()); err != nil {
		t.Fatal(err)
	}
	// The old app renames a task and adds one; it cannot see the new fields.
	simulateOldApp(t, s, func(p map[string]any) {
		tasks := p["tasks"].([]any)
		tasks[0].(map[string]any)["title"] = "A renamed in old app"
		p["tasks"] = append(tasks, map[string]any{"id": "t3", "title": "Added in old app"})
	})
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	t1 := got[0].Tasks[0]
	if t1.Title != "A renamed in old app" {
		t.Error("edits made by the old app must be kept")
	}
	if t1.DueDate != "2026-10-16" || len(t1.Links) != 1 || len(t1.Attachments) != 1 {
		t.Errorf("new fields not restored: %+v", t1)
	}
	if len(got[0].Quarters) != 4 {
		t.Error("project quarters not restored")
	}
	if len(got[0].Tasks) != 3 {
		t.Error("task added by the old app was lost")
	}
}

func TestClearingInTheNewAppIsNotUndone(t *testing.T) {
	s := newTestStore(t)
	p := rich()
	s.Save(p)
	p[0].Tasks[0].DueDate = ""
	p[0].Tasks[0].Links = nil
	p[0].Quarters = nil
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load()
	if got[0].Tasks[0].DueDate != "" || len(got[0].Tasks[0].Links) != 0 || len(got[0].Quarters) != 0 {
		t.Errorf("a deliberate clear came back: %+v", got[0])
	}
	if len(got[0].Tasks[0].Attachments) != 1 {
		t.Error("untouched field lost")
	}
}

func TestNewerValueWinsOverShadow(t *testing.T) {
	s := newTestStore(t)
	s.Save(rich())
	// Someone (e.g. a newer build) edits config.json directly with a new date.
	data, _ := os.ReadFile(s.path)
	var raw []map[string]any
	json.Unmarshal(data, &raw)
	raw[0]["tasks"].([]any)[0].(map[string]any)["dueDate"] = "2026-12-01"
	out, _ := json.Marshal(raw)
	os.WriteFile(s.path, out, 0o644)
	got, _ := s.Load()
	if got[0].Tasks[0].DueDate != "2026-12-01" {
		t.Errorf("present value must not be overwritten: %q", got[0].Tasks[0].DueDate)
	}
}

func TestMissingOrCorruptShadowIsHarmless(t *testing.T) {
	s := newTestStore(t)
	s.Save(rich())
	os.WriteFile(s.extendedPath(), []byte("{broken"), 0o644)
	if got, err := s.Load(); err != nil || got[0].Tasks[0].DueDate != "2026-10-16" {
		t.Fatalf("load with corrupt shadow: %v %+v", err, got)
	}
	os.Remove(s.extendedPath())
	if got, err := s.Load(); err != nil || len(got) != 1 {
		t.Fatalf("load without shadow: %v", err)
	}
}
