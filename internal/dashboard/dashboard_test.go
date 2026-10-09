package dashboard

import (
	"strings"
	"testing"
	"time"

	"devdeck/internal/model"
)

var today = time.Date(2026, 10, 9, 12, 0, 0, 0, time.Local) // Friday

func sources() Sources {
	return Sources{
		Today: today,
		Projects: []model.Project{
			{ID: "web", Name: "Web", Sprints: []model.Sprint{{ID: "s1", Name: "Sprint 1"}},
				SavedFilters: []model.SavedFilter{{ID: "pf", Name: "Bugs", Query: model.TaskQuery{Types: []string{"bug"}}}},
				Tasks: []model.Task{
					{ID: "a", Title: "Overdue bug", Type: "bug", Status: "todo", DueDate: "2026-10-01", Priority: "high", Assignee: "Sam, Alex", Labels: []string{"stripe"}, SprintID: "s1"},
					{ID: "b", Title: "Today", Status: "inprogress", DueDate: "2026-10-09", Assignee: "Sam"},
					{ID: "c", Title: "Later this week", Status: "todo", DueDate: "2026-10-11"},
					{ID: "d", Title: "Done late", Status: "done", Done: true, DueDate: "2026-10-02"},
					{ID: "e", Title: "No date", Status: "backlog"},
				}},
			{ID: "ops", Name: "Ops", Tasks: []model.Task{
				{ID: "o", Title: "Ops today", Status: "todo", DueDate: "2026-10-09", Type: "bug"},
			}},
		},
		AppFilters: []model.SavedFilter{{ID: "high", Name: "High priority", Query: model.TaskQuery{Priorities: []string{"high"}}}},
	}
}

func ids(d Data) string {
	var out []string
	for _, t := range d.Tasks {
		out = append(out, t.TaskID)
	}
	return strings.Join(out, ",")
}

func TestDueWidgets(t *testing.T) {
	s := sources()
	cases := map[string]string{"overdue": "a", "today": "o,b", "this_week": "o,b,c", "no_date": "e"}
	for rng, want := range cases {
		d := WidgetData(Widget{ID: "x", Type: TypeDue, Range: rng}, s)
		if got := ids(d); got != want || d.Error != "" {
			t.Errorf("%s = %s (%s), want %s", rng, got, d.Error, want)
		}
	}
	d := WidgetData(Widget{Type: TypeDue, Range: "today", ProjectID: "web"}, s)
	if ids(d) != "b" || d.Title != "Due today" {
		t.Errorf("project scope: %s %q", ids(d), d.Title)
	}
	d = WidgetData(Widget{Type: TypeDue, Range: "this_month", IncludeDone: true}, s)
	if !strings.Contains(ids(d), "d") {
		t.Errorf("includeDone: %s", ids(d))
	}
}

func TestFilterCustomAndGroups(t *testing.T) {
	s := sources()
	d := WidgetData(Widget{Type: TypeFilter, FilterID: "high"}, s)
	if ids(d) != "a" || d.Title != "High priority" {
		t.Errorf("app filter: %s %q", ids(d), d.Title)
	}
	d = WidgetData(Widget{Type: TypeFilter, FilterID: "pf"}, s)
	if ids(d) != "a,o" || d.Title != "Bugs" { // project filter, run across all projects
		t.Errorf("project filter: %s %q", ids(d), d.Title)
	}
	d = WidgetData(Widget{Type: TypeFilter, FilterID: "gone"}, s)
	if d.Error == "" {
		t.Error("missing filter should report an error")
	}
	q := model.TaskQuery{DuePreset: "this_week"}
	d = WidgetData(Widget{Type: TypeCustom, Query: &q, Display: "bar", GroupBy: "assignee"}, s)
	got := []string{}
	for _, g := range d.Groups {
		got = append(got, g.Label+"="+string(rune('0'+g.Count)))
	}
	if strings.Join(got, ",") != "Unassigned=2,Sam=1" {
		t.Errorf("assignee groups: %v", got)
	}
	d = WidgetData(Widget{Type: TypeCustom, Query: &model.TaskQuery{}, GroupBy: "due"}, s)
	got = got[:0]
	for _, g := range d.Groups {
		got = append(got, g.Key)
	}
	if strings.Join(got, ",") != "overdue,today,this_week,none" {
		t.Errorf("due buckets in order: %v", got)
	}
	d = WidgetData(Widget{Type: TypeCustom, Query: &model.TaskQuery{}, GroupBy: "sprint", ProjectID: "web"}, s)
	if len(d.Groups) != 2 || d.Groups[0].Label != "Backlog" || d.Groups[1].Label != "Sprint 1" {
		t.Errorf("sprint groups: %+v", d.Groups)
	}
	if WidgetData(Widget{Type: TypeStats}, s).Error == "" {
		t.Error("built-in panels have no task data")
	}
}

func TestValidateAndStore(t *testing.T) {
	bad := []Widget{
		{Type: "nope"},
		{Type: TypeDue},
		{Type: TypeDue, Range: "someday"},
		{Type: TypeFilter},
		{Type: TypeCustom},
		{Type: TypeCustom, Query: &model.TaskQuery{DueFrom: "10/9"}},
		{Type: TypeCustom, Query: &model.TaskQuery{}, Display: "pie"},
		{Type: TypeHTML},
		{Type: TypeHTML, HTML: strings.Repeat("x", MaxHTML+1)},
		{Type: TypeStats, Size: "xl"},
		{Type: TypeStats, Title: strings.Repeat("t", 81)},
	}
	for _, w := range bad {
		if err := Validate(&w); err == nil {
			t.Errorf("Validate(%+v) should fail", w)
		}
	}
	w := Widget{Type: TypeStats, HTML: "<script>"}
	if err := Validate(&w); err != nil || w.Size != "m" || w.HTML != "" {
		t.Errorf("defaults: %+v %v", w, err)
	}

	dir := t.TempDir()
	if l := Load(dir); len(l.Widgets) != len(Default().Widgets) {
		t.Fatal("missing file should give the default layout")
	}
	added, err := Upsert(dir, Widget{Type: TypeDue, Range: "next_quarter", Size: "s"})
	if err != nil || !strings.HasPrefix(added.ID, "w-") {
		t.Fatalf("upsert: %+v %v", added, err)
	}
	added.Title = "Next quarter"
	if _, err := Upsert(dir, added); err != nil {
		t.Fatal(err)
	}
	l := Load(dir)
	last := l.Widgets[len(l.Widgets)-1]
	if last.ID != added.ID || last.Title != "Next quarter" || len(l.Widgets) != len(Default().Widgets)+1 {
		t.Fatalf("after update: %+v", last)
	}
	if err := Remove(dir, "stats"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "stats"); err == nil {
		t.Error("removing twice should fail")
	}
	for _, x := range Load(dir).Widgets {
		if x.ID == "stats" {
			t.Error("stats should be gone")
		}
	}
	if _, err := Save(dir, Layout{Widgets: []Widget{{ID: "a", Type: TypeStats}, {ID: "a", Type: TypeFlow}}}); err == nil {
		t.Error("duplicate ids should be refused")
	}
	if l, err := Save(dir, Layout{}); err != nil || len(Load(dir).Widgets) != 0 || l.Widgets == nil {
		t.Errorf("an empty dashboard is allowed: %v", err)
	}
}

func TestImportExport(t *testing.T) {
	q := model.TaskQuery{Priorities: []string{"high"}}
	data, err := Export([]Widget{
		{ID: "a", Type: TypeCustom, Title: "Hot", Query: &q, Display: "count", ProjectID: "elsewhere"},
		{ID: "b", Type: TypeHTML, Title: "Chart", HTML: "<div id=x></div>", Query: &q},
	})
	if err != nil {
		t.Fatal(err)
	}
	ws, warnings, err := ParseImport(data, map[string]bool{"web": true})
	if err != nil || len(ws) != 2 {
		t.Fatalf("import: %v %v", ws, err)
	}
	if ws[0].ID == "a" || ws[0].ProjectID != "" || len(warnings) != 2 || !strings.Contains(warnings[1], "contains code") {
		t.Errorf("fresh ids, cleared unknown project, warnings: %+v %v", ws[0], warnings)
	}
	if ws, _, err := ParseImport([]byte(`{"type":"due","range":"today","title":"Today"}`), nil); err != nil || len(ws) != 1 {
		t.Errorf("single widget object: %v", err)
	}
	for _, bad := range []string{`nope`, `{}`, `{"jumpstartWidget":1,"widgets":[]}`, `{"jumpstartWidget":1,"widgets":[{"type":"due"}]}`} {
		if _, _, err := ParseImport([]byte(bad), nil); err == nil {
			t.Errorf("ParseImport(%s) should fail", bad)
		}
	}
}
