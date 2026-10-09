// Package dashboard stores the customizable dashboard layout
// (<data dir>/dashboard.json) and computes each widget's data. The app, the
// MCP server and the Raycast extension all read the same layout and data,
// so a widget looks the same everywhere it can be shown.
package dashboard

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"devdeck/internal/daterange"
	"devdeck/internal/model"
	"devdeck/internal/query"
)

const fileName = "dashboard.json"

// Widget types. Built-in panels render app data (usage, ports, projects);
// task widgets ("due", "filter", "custom") are backed by a task query and
// work outside the app too; "html" is user code in a sandbox.
const (
	TypeStats    = "stats"
	TypeFlow     = "flow"
	TypeDonut    = "donut"
	TypeRecent   = "recent"
	TypeActivity = "activity"
	TypePorts    = "ports"
	TypeImport   = "import"
	TypeDue      = "due"
	TypeFilter   = "filter"
	TypeCustom   = "custom"
	TypeHTML     = "html"
)

// Types lists every widget type.
var Types = []string{TypeStats, TypeFlow, TypeDonut, TypeRecent, TypeActivity, TypePorts, TypeImport, TypeDue, TypeFilter, TypeCustom, TypeHTML}

// DueRanges are the ranges a "due" widget can show.
var DueRanges = []string{"overdue", "today", "tomorrow", "this_week", "next_week", "this_month", "next_month", "this_quarter", "next_quarter", "this_year", "no_date"}

// Displays for task widgets; GroupBys for count/bar/donut/table.
var (
	Displays = []string{"list", "count", "bar", "donut", "table"}
	GroupBys = []string{"status", "priority", "type", "assignee", "label", "project", "sprint", "due"}
)

// MaxHTML caps an HTML widget's source.
const MaxHTML = 200 * 1024

// Widget is one dashboard tile.
type Widget struct {
	ID    string `json:"id,omitempty"`
	Type  string `json:"type" jsonschema:"due, filter, custom, html, or a built-in panel (see widget_schema)"`
	Title string `json:"title,omitempty"`
	// Size is s, m or l (one, two or three grid columns).
	Size string `json:"size,omitempty"`
	// ProjectID limits a task widget to one project; empty means all.
	ProjectID string `json:"projectId,omitempty"`
	// Range is the due-date range of a "due" widget.
	Range string `json:"range,omitempty"`
	// FilterID is the saved filter behind a "filter" widget.
	FilterID string `json:"filterId,omitempty"`
	// Query, Display and GroupBy define a declarative "custom" widget (and
	// optionally refine "due"/"filter" widgets' presentation).
	Query   *model.TaskQuery `json:"query,omitempty"`
	Display string           `json:"display,omitempty"`
	GroupBy string           `json:"groupBy,omitempty"`
	// IncludeDone shows finished tasks too (task widgets hide them by default).
	IncludeDone bool `json:"includeDone,omitempty"`
	// HTML is the source of an "html" widget, run in a sandboxed iframe that
	// receives this widget's task data (from Query) by postMessage.
	HTML string `json:"html,omitempty"`
}

// Layout is the whole dashboard.
type Layout struct {
	Version int      `json:"version"`
	Widgets []Widget `json:"widgets"`
}

var mu sync.Mutex

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "w-" + hex.EncodeToString(b)
}

func oneOf(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Default is the layout new installs (and Reset) get: the existing panels
// plus due-date widgets.
func Default() Layout {
	w := func(id, typ, size string, extra func(*Widget)) Widget {
		x := Widget{ID: id, Type: typ, Size: size}
		if extra != nil {
			extra(&x)
		}
		return x
	}
	return Layout{Version: 1, Widgets: []Widget{
		w("stats", TypeStats, "l", nil),
		w("due-overdue", TypeDue, "s", func(x *Widget) { x.Range = "overdue"; x.Title = "Past due" }),
		w("due-today", TypeDue, "s", func(x *Widget) { x.Range = "today"; x.Title = "Due today" }),
		w("due-week", TypeDue, "s", func(x *Widget) { x.Range = "this_week"; x.Title = "Due this week" }),
		w("flow", TypeFlow, "m", nil),
		w("donut", TypeDonut, "s", nil),
		w("recent", TypeRecent, "m", nil),
		w("activity", TypeActivity, "s", nil),
		w("ports", TypePorts, "l", nil),
		w("import", TypeImport, "l", nil),
	}}
}

// Validate checks one widget and fills defaults (size, display).
func Validate(w *Widget) error {
	w.Title = strings.TrimSpace(w.Title)
	if utf8.RuneCountInString(w.Title) > 80 {
		return fmt.Errorf("widget titles are limited to 80 characters")
	}
	if !oneOf(w.Type, Types) {
		return fmt.Errorf("unknown widget type %q (want one of %s)", w.Type, strings.Join(Types, ", "))
	}
	if w.Size == "" {
		w.Size = "m"
	}
	if !oneOf(w.Size, []string{"s", "m", "l"}) {
		return fmt.Errorf("size must be s, m or l")
	}
	if w.Display != "" && !oneOf(w.Display, Displays) {
		return fmt.Errorf("display must be one of %s", strings.Join(Displays, ", "))
	}
	if w.GroupBy != "" && !oneOf(w.GroupBy, GroupBys) {
		return fmt.Errorf("groupBy must be one of %s", strings.Join(GroupBys, ", "))
	}
	if w.Query != nil {
		if err := query.ValidateQuery(*w.Query); err != nil {
			return err
		}
	}
	switch w.Type {
	case TypeDue:
		if !oneOf(w.Range, DueRanges) {
			return fmt.Errorf("a due widget needs range: one of %s", strings.Join(DueRanges, ", "))
		}
	case TypeFilter:
		if strings.TrimSpace(w.FilterID) == "" {
			return fmt.Errorf("a filter widget needs filterId (see list_saved_filters)")
		}
	case TypeCustom:
		if w.Query == nil {
			return fmt.Errorf("a custom widget needs a query")
		}
		if w.Display == "" {
			w.Display = "list"
		}
	case TypeHTML:
		if strings.TrimSpace(w.HTML) == "" {
			return fmt.Errorf("an html widget needs html")
		}
		if len(w.HTML) > MaxHTML {
			return fmt.Errorf("html widgets are limited to %d KB", MaxHTML/1024)
		}
	}
	if w.Type != TypeHTML {
		w.HTML = ""
	}
	return nil
}

// ValidateLayout checks every widget, assigns missing ids and rejects
// duplicates.
func ValidateLayout(l *Layout) error {
	if len(l.Widgets) > 60 {
		return fmt.Errorf("a dashboard holds at most 60 widgets")
	}
	seen := map[string]bool{}
	for i := range l.Widgets {
		w := &l.Widgets[i]
		if w.ID == "" {
			w.ID = newID()
		}
		if seen[w.ID] {
			return fmt.Errorf("duplicate widget id %q", w.ID)
		}
		seen[w.ID] = true
		if err := Validate(w); err != nil {
			label := w.Title
			if label == "" {
				label = w.Type
			}
			return fmt.Errorf("widget %d (%s): %w", i+1, label, err)
		}
	}
	l.Version = 1
	if l.Widgets == nil {
		l.Widgets = []Widget{}
	}
	return nil
}

// Load reads the layout; a missing or unreadable file gives the default.
func Load(dir string) Layout {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return Default()
	}
	var l Layout
	if json.Unmarshal(data, &l) != nil || ValidateLayout(&l) != nil {
		return Default()
	}
	return l
}

// Save validates and writes the layout atomically.
func Save(dir string, l Layout) (Layout, error) {
	if err := ValidateLayout(&l); err != nil {
		return l, err
	}
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return l, err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return l, err
	}
	tmp := filepath.Join(dir, fileName+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return l, err
	}
	return l, os.Rename(tmp, filepath.Join(dir, fileName))
}

// Upsert adds a widget (blank or unknown id: appended) or replaces the one
// with the same id, and saves.
func Upsert(dir string, w Widget) (Widget, error) {
	l := Load(dir)
	if w.ID != "" {
		for i := range l.Widgets {
			if l.Widgets[i].ID == w.ID {
				l.Widgets[i] = w
				saved, err := Save(dir, l)
				if err != nil {
					return w, err
				}
				return saved.Widgets[i], nil
			}
		}
	}
	l.Widgets = append(l.Widgets, w)
	saved, err := Save(dir, l)
	if err != nil {
		return w, err
	}
	return saved.Widgets[len(saved.Widgets)-1], nil
}

// Remove deletes a widget by id.
func Remove(dir, id string) error {
	l := Load(dir)
	out := l.Widgets[:0]
	found := false
	for _, w := range l.Widgets {
		if w.ID == id {
			found = true
			continue
		}
		out = append(out, w)
	}
	if !found {
		return fmt.Errorf("no widget with id %q", id)
	}
	l.Widgets = out
	_, err := Save(dir, l)
	return err
}

// File is the import/export format (.jumpstart-widget.json).
type File struct {
	JumpStartWidget int      `json:"jumpstartWidget"`
	Widgets         []Widget `json:"widgets"`
}

// Export packs widgets for sharing. Project ids are kept; an importer on
// another Mac falls back to all projects when the id is unknown.
func Export(widgets []Widget) ([]byte, error) {
	return json.MarshalIndent(File{JumpStartWidget: 1, Widgets: widgets}, "", "  ")
}

// ParseImport reads a widget file (or a single widget object), validates
// every widget, and gives each a fresh id. Warnings note code widgets and
// unknown projects.
func ParseImport(data []byte, knownProjects map[string]bool) ([]Widget, []string, error) {
	var f File
	if err := json.Unmarshal(data, &f); err != nil || (f.JumpStartWidget == 0 && len(f.Widgets) == 0) {
		var one Widget
		if err2 := json.Unmarshal(data, &one); err2 != nil || one.Type == "" {
			return nil, nil, fmt.Errorf("not a JumpStart widget file")
		}
		f.Widgets = []Widget{one}
	}
	if len(f.Widgets) == 0 {
		return nil, nil, fmt.Errorf("the file has no widgets")
	}
	if len(f.Widgets) > 20 {
		return nil, nil, fmt.Errorf("import at most 20 widgets at a time")
	}
	var warnings []string
	for i := range f.Widgets {
		w := &f.Widgets[i]
		w.ID = newID()
		if err := Validate(w); err != nil {
			return nil, nil, fmt.Errorf("widget %d: %w", i+1, err)
		}
		if w.ProjectID != "" && !knownProjects[w.ProjectID] {
			warnings = append(warnings, fmt.Sprintf("%q referred to a project that is not here; it now shows all projects", nameOf(*w)))
			w.ProjectID = ""
		}
		if w.Type == TypeHTML {
			warnings = append(warnings, fmt.Sprintf("%q contains code. It runs sandboxed with no network or app access and sees only its own task data", nameOf(*w)))
		}
	}
	return f.Widgets, warnings, nil
}

func nameOf(w Widget) string {
	if w.Title != "" {
		return w.Title
	}
	return w.Type
}

// DueQuery is the task query behind a due widget's range.
func DueQuery(rng string) model.TaskQuery {
	switch rng {
	case "overdue":
		return model.TaskQuery{Overdue: true}
	case "no_date":
		return model.TaskQuery{NoDueDate: true}
	}
	if oneOf(rng, daterange.Presets) {
		return model.TaskQuery{DuePreset: rng}
	}
	return model.TaskQuery{}
}
