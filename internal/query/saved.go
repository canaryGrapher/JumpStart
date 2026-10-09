package query

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

const filtersFile = "filters.json"

var filtersMu sync.Mutex

type filtersDoc struct {
	Version int                 `json:"version"`
	Seeded  bool                `json:"seeded"` // defaults were added once; deleting one is respected
	Filters []model.SavedFilter `json:"filters"`
}

// Defaults are the starter filters. Each can be deleted; RestoreDefaults
// brings back any that are missing.
func Defaults() []model.SavedFilter {
	f := func(id, name string, q model.TaskQuery) model.SavedFilter {
		return model.SavedFilter{ID: "default-" + id, Name: name, Query: q, Builtin: true}
	}
	return []model.SavedFilter{
		f("due-today", "Due today", model.TaskQuery{DuePreset: daterange.Today}),
		f("due-this-week", "Due this week", model.TaskQuery{DuePreset: daterange.ThisWeek}),
		f("past-due", "Past due", model.TaskQuery{Overdue: true}),
		f("due-this-month", "Due this month", model.TaskQuery{DuePreset: daterange.ThisMonth}),
		f("due-this-quarter", "Due this quarter", model.TaskQuery{DuePreset: daterange.ThisQuarter}),
		f("high-priority", "High priority", model.TaskQuery{Priorities: []string{"high"}}),
		f("needs-my-action", "Needs my action", model.TaskQuery{Labels: []string{"needs-user-action"}}),
		f("blocked", "Blocked", model.TaskQuery{Labels: []string{"blocked"}}),
		f("no-due-date", "No due date", model.TaskQuery{NoDueDate: true}),
		f("missing-criteria", "Missing acceptance criteria", model.TaskQuery{Acceptance: "none"}),
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "f-" + hex.EncodeToString(b)
}

func readDoc(dir string) filtersDoc {
	var doc filtersDoc
	if data, err := os.ReadFile(filepath.Join(dir, filtersFile)); err == nil {
		_ = json.Unmarshal(data, &doc)
	}
	return doc
}

func writeDoc(dir string, doc filtersDoc) error {
	doc.Version = 1
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, filtersFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, filtersFile))
}

// AppFilters returns the app-wide saved filters, seeding the defaults the
// first time.
func AppFilters(dir string) ([]model.SavedFilter, error) {
	filtersMu.Lock()
	defer filtersMu.Unlock()
	doc := readDoc(dir)
	if !doc.Seeded {
		doc.Filters = append(Defaults(), doc.Filters...)
		doc.Seeded = true
		if err := writeDoc(dir, doc); err != nil {
			return nil, err
		}
	}
	if doc.Filters == nil {
		doc.Filters = []model.SavedFilter{}
	}
	return doc.Filters, nil
}

// Validate checks a filter's name and query.
func Validate(f model.SavedFilter) error {
	if strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("a saved filter needs a name")
	}
	if len(f.Name) > 80 {
		return fmt.Errorf("filter names are limited to 80 characters")
	}
	return ValidateQuery(f.Query)
}

// ValidateQuery checks a TaskQuery's presets, dates and has/none values.
func ValidateQuery(q TaskQuery) error {
	if q.DuePreset != "" {
		if _, err := daterange.Resolve(q.DuePreset, time.Now(), daterange.DefaultQuarters()); err != nil {
			return err
		}
	}
	for _, d := range []string{q.DueFrom, q.DueTo} {
		if d != "" {
			if _, err := daterange.ParseDate(d); err != nil {
				return fmt.Errorf("dates must be YYYY-MM-DD, got %q", d)
			}
		}
	}
	for _, v := range []string{q.Acceptance, q.Subtasks} {
		if v != "" && v != "has" && v != "none" {
			return fmt.Errorf("acceptance/subtasks must be has or none")
		}
	}
	return nil
}

// Upsert adds or updates a filter in list (by id; a blank id creates one).
// Names must be unique within the list, ignoring case.
func Upsert(list []model.SavedFilter, f model.SavedFilter, now time.Time) ([]model.SavedFilter, model.SavedFilter, error) {
	f.Name = strings.TrimSpace(f.Name)
	if err := Validate(f); err != nil {
		return list, f, err
	}
	for _, x := range list {
		if strings.EqualFold(x.Name, f.Name) && x.ID != f.ID {
			return list, f, fmt.Errorf("a filter named %q already exists", f.Name)
		}
	}
	out := append([]model.SavedFilter(nil), list...)
	for i := range out {
		if out[i].ID == f.ID && f.ID != "" {
			f.Builtin = out[i].Builtin
			f.CreatedAt = out[i].CreatedAt
			out[i] = f
			return out, f, nil
		}
	}
	if f.ID == "" {
		f.ID = newID()
	}
	f.Builtin = false
	f.CreatedAt = now.UnixMilli()
	return append(out, f), f, nil
}

// Remove deletes a filter by id.
func Remove(list []model.SavedFilter, id string) ([]model.SavedFilter, bool) {
	out := []model.SavedFilter{}
	found := false
	for _, f := range list {
		if f.ID == id {
			found = true
			continue
		}
		out = append(out, f)
	}
	return out, found
}

// SaveAppFilter stores an app-wide filter.
func SaveAppFilter(dir string, f model.SavedFilter) (model.SavedFilter, error) {
	list, err := AppFilters(dir)
	if err != nil {
		return f, err
	}
	filtersMu.Lock()
	defer filtersMu.Unlock()
	out, saved, err := Upsert(list, f, time.Now())
	if err != nil {
		return saved, err
	}
	return saved, writeDoc(dir, filtersDoc{Seeded: true, Filters: out})
}

// DeleteAppFilter removes an app-wide filter (defaults included).
func DeleteAppFilter(dir, id string) error {
	list, err := AppFilters(dir)
	if err != nil {
		return err
	}
	filtersMu.Lock()
	defer filtersMu.Unlock()
	out, found := Remove(list, id)
	if !found {
		return fmt.Errorf("filter %s not found", id)
	}
	return writeDoc(dir, filtersDoc{Seeded: true, Filters: out})
}

// RestoreDefaults re-adds any default filter that was deleted.
func RestoreDefaults(dir string) ([]model.SavedFilter, error) {
	list, err := AppFilters(dir)
	if err != nil {
		return nil, err
	}
	filtersMu.Lock()
	defer filtersMu.Unlock()
	have := map[string]bool{}
	for _, f := range list {
		have[f.ID] = true
	}
	for _, d := range Defaults() {
		if !have[d.ID] {
			list = append(list, d)
		}
	}
	return list, writeDoc(dir, filtersDoc{Seeded: true, Filters: list})
}
