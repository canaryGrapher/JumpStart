package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/ghsync"
	"devdeck/internal/model"
	"devdeck/internal/query"
	"devdeck/internal/search"
)

// MaxTasks caps how many tasks a widget's data carries (Total has the count).
const MaxTasks = 50

// Group is one bucket of a grouped widget.
type Group struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Data is what a task widget shows.
type Data struct {
	WidgetID string          `json:"widgetId"`
	Title    string          `json:"title"`
	Query    model.TaskQuery `json:"query"`
	Total    int             `json:"total"`
	Tasks    []search.Result `json:"tasks"`
	Groups   []Group         `json:"groups,omitempty"`
	Error    string          `json:"error,omitempty"`
}

var dueTitles = map[string]string{
	"overdue": "Past due", "today": "Due today", "tomorrow": "Due tomorrow", "this_week": "Due this week",
	"next_week": "Due next week", "this_month": "Due this month", "next_month": "Due next month",
	"this_quarter": "Due this quarter", "next_quarter": "Due next quarter", "this_year": "Due this year",
	"no_date": "No due date",
}

var builtinTitles = map[string]string{
	TypeStats: "Overview", TypeFlow: "Task flow", TypeDonut: "Completion mix", TypeRecent: "Recent projects",
	TypeActivity: "Activity by project", TypePorts: "Live ports", TypeImport: "Config import",
	TypeCustom: "Custom widget", TypeHTML: "Custom widget", TypeFilter: "Saved filter",
}

// IsTaskWidget reports whether a widget's content comes from a task query.
func IsTaskWidget(w Widget) bool {
	return w.Type == TypeDue || w.Type == TypeFilter || w.Type == TypeCustom || w.Type == TypeHTML
}

// TitleOf is the widget's title, or a sensible default.
func TitleOf(w Widget) string {
	if w.Title != "" {
		return w.Title
	}
	if w.Type == TypeDue {
		return dueTitles[w.Range]
	}
	return builtinTitles[w.Type]
}

// Sources is everything widget data is computed from.
type Sources struct {
	Projects       []model.Project
	GlobalQuarters []model.QuarterRange
	AppFilters     []model.SavedFilter
	Today          time.Time
}

func (s Sources) findFilter(w Widget) (model.SavedFilter, bool) {
	for _, p := range s.Projects {
		if w.ProjectID != "" && p.ID != w.ProjectID {
			continue
		}
		for _, f := range p.SavedFilters {
			if f.ID == w.FilterID {
				return f, true
			}
		}
	}
	for _, f := range s.AppFilters {
		if f.ID == w.FilterID {
			return f, true
		}
	}
	return model.SavedFilter{}, false
}

// QueryOf resolves the task query behind a widget.
func QueryOf(w Widget, s Sources) (model.TaskQuery, string, error) {
	title := TitleOf(w)
	switch w.Type {
	case TypeDue:
		return DueQuery(w.Range), title, nil
	case TypeFilter:
		f, ok := s.findFilter(w)
		if !ok {
			return model.TaskQuery{}, title, fmt.Errorf("the saved filter behind this widget no longer exists")
		}
		if w.Title == "" {
			title = f.Name
		}
		return f.Query, title, nil
	case TypeCustom, TypeHTML:
		if w.Query == nil {
			return model.TaskQuery{}, title, nil
		}
		return *w.Query, title, nil
	}
	return model.TaskQuery{}, title, fmt.Errorf("%s widgets have no task data", w.Type)
}

// WidgetData runs a task widget's query and groups the result.
func WidgetData(w Widget, s Sources) Data {
	if s.Today.IsZero() {
		s.Today = time.Now()
	}
	q, title, err := QueryOf(w, s)
	d := Data{WidgetID: w.ID, Title: title, Query: q, Tasks: []search.Result{}}
	if err != nil {
		d.Error = err.Error()
		return d
	}
	type hit struct {
		p *model.Project
		t *model.Task
	}
	var hits []hit
	for pi := range s.Projects {
		p := &s.Projects[pi]
		if w.ProjectID != "" && p.ID != w.ProjectID {
			continue
		}
		tasks, err := query.Run(q, *p, s.GlobalQuarters, s.Today)
		if err != nil {
			d.Error = err.Error()
			return d
		}
		for ti := range tasks {
			t := tasks[ti]
			if !w.IncludeDone && (t.Done || t.Status == "done") {
				continue
			}
			hits = append(hits, hit{p, &t})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i].t, hits[j].t
		if (a.DueDate == "") != (b.DueDate == "") {
			return a.DueDate != ""
		}
		if a.DueDate != b.DueDate {
			return a.DueDate < b.DueDate
		}
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
	d.Total = len(hits)
	for i, h := range hits {
		if i >= MaxTasks {
			break
		}
		d.Tasks = append(d.Tasks, search.TaskResult(h.p, h.t, s.Today))
	}
	if w.GroupBy != "" {
		counts := map[string]*Group{}
		var order []string
		add := func(key, label string) {
			g, ok := counts[key]
			if !ok {
				g = &Group{Key: key, Label: label}
				counts[key] = g
				order = append(order, key)
			}
			g.Count++
		}
		for _, h := range hits {
			for _, kv := range groupKeys(w.GroupBy, h.p, h.t, s.Today) {
				add(kv[0], kv[1])
			}
		}
		for _, k := range order {
			d.Groups = append(d.Groups, *counts[k])
		}
		sort.SliceStable(d.Groups, func(i, j int) bool {
			if w.GroupBy == "due" {
				return dueBucketRank[d.Groups[i].Key] < dueBucketRank[d.Groups[j].Key]
			}
			if d.Groups[i].Count != d.Groups[j].Count {
				return d.Groups[i].Count > d.Groups[j].Count
			}
			return d.Groups[i].Label < d.Groups[j].Label
		})
	}
	return d
}

var dueBucketRank = map[string]int{"overdue": 0, "today": 1, "this_week": 2, "later": 3, "none": 4}

func groupKeys(by string, p *model.Project, t *model.Task, today time.Time) [][2]string {
	switch by {
	case "status":
		label := t.Status
		for _, c := range ghsync.EffectiveColumns(p) {
			if c.ID == t.Status {
				label = c.Label
			}
		}
		return [][2]string{{strings.ToLower(label), label}}
	case "priority":
		if t.Priority == "" {
			return [][2]string{{"", "No priority"}}
		}
		return [][2]string{{t.Priority, strings.ToUpper(t.Priority[:1]) + t.Priority[1:]}}
	case "type":
		typ := t.Type
		if typ == "" {
			typ = "task"
		}
		return [][2]string{{typ, strings.ToUpper(typ[:1]) + typ[1:]}}
	case "assignee":
		var out [][2]string
		for _, a := range strings.Split(t.Assignee, ",") {
			if a = strings.TrimSpace(a); a != "" {
				out = append(out, [2]string{strings.ToLower(a), a})
			}
		}
		if len(out) == 0 {
			return [][2]string{{"", "Unassigned"}}
		}
		return out
	case "label":
		if len(t.Labels) == 0 {
			return [][2]string{{"", "No label"}}
		}
		var out [][2]string
		for _, l := range t.Labels {
			out = append(out, [2]string{strings.ToLower(l), l})
		}
		return out
	case "project":
		return [][2]string{{p.ID, p.Name}}
	case "sprint":
		for _, s := range p.Sprints {
			if s.ID == t.SprintID {
				return [][2]string{{p.ID + "/" + s.ID, s.Name}}
			}
		}
		return [][2]string{{"", "Backlog"}}
	case "due":
		done := t.Done || t.Status == "done"
		switch {
		case t.DueDate == "":
			return [][2]string{{"none", "No date"}}
		case daterange.IsPastDue(t.DueDate, done, today):
			return [][2]string{{"overdue", "Past due"}}
		case t.DueDate == today.Format(daterange.DateLayout):
			return [][2]string{{"today", "Today"}}
		}
		if wk, err := daterange.Resolve("this_week", today, nil); err == nil && wk.Contains(t.DueDate) {
			return [][2]string{{"this_week", "This week"}}
		}
		return [][2]string{{"later", "Later"}}
	}
	return nil
}
