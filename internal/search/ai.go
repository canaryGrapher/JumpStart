package search

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/ghsync"
	"devdeck/internal/model"
	"devdeck/internal/query"
)

// AI search asks the local model to turn a question into a TaskQuery, then
// JumpStart runs that filter itself. The model never produces results, so
// it cannot invent tasks, and the filter it chose is shown to the user.

// Vocab is what the model may filter on, gathered from the real data.
type Vocab struct {
	Projects   []string
	Statuses   map[string]string // id -> label
	Assignees  []string
	Labels     []string
	Priorities []string
	Types      []string
}

// BuildVocab collects the values in use across projects.
func BuildVocab(projects []model.Project) Vocab {
	v := Vocab{Statuses: map[string]string{}, Priorities: []string{"low", "medium", "high"}, Types: []string{"story", "task", "bug"}}
	assignees, labels := map[string]bool{}, map[string]bool{}
	for i := range projects {
		p := &projects[i]
		v.Projects = append(v.Projects, p.Name)
		for _, c := range ghsync.EffectiveColumns(p) {
			v.Statuses[c.ID] = c.Label
		}
		for _, t := range p.Tasks {
			for _, a := range strings.Split(t.Assignee, ",") {
				if a = strings.TrimSpace(a); a != "" {
					assignees[a] = true
				}
			}
			for _, l := range t.Labels {
				labels[l] = true
			}
		}
	}
	for a := range assignees {
		v.Assignees = append(v.Assignees, a)
	}
	for l := range labels {
		v.Labels = append(v.Labels, l)
	}
	sort.Strings(v.Assignees)
	sort.Strings(v.Labels)
	return v
}

// AIPrompt returns the system and user messages for a question.
func AIPrompt(question string, v Vocab, today time.Time) (system, user string) {
	var statuses []string
	for id, label := range v.Statuses {
		statuses = append(statuses, fmt.Sprintf("%s (%s)", id, label))
	}
	sort.Strings(statuses)
	cap40 := func(list []string) string {
		if len(list) > 40 {
			list = list[:40]
		}
		return strings.Join(list, ", ")
	}
	system = `You convert a question about tasks into a JSON filter for the JumpStart task board.
Reply with one JSON object and nothing else:
{"query": {
   "duePreset": one of ` + strings.Join(daterange.Presets, ", ") + ` or "",
   "dueFrom": "YYYY-MM-DD" or "", "dueTo": "YYYY-MM-DD" or "",
   "noDueDate": bool, "overdue": bool,
   "statuses": [status ids], "priorities": [low|medium|high], "types": [story|task|bug],
   "assignees": [names], "labels": [labels],
   "acceptance": "has"|"none"|"", "subtasks": "has"|"none"|"",
   "text": words that must appear in the task, or ""},
 "projects": [project names to search, empty for all],
 "explain": "one short sentence describing the filter"}
Use only values from the lists below. Leave a field empty rather than guessing.
Use dueFrom/dueTo for specific dates and duePreset for named ranges. Weeks run Monday to Sunday.
"Open" or "not done" means every status except done. The board's done status id is "done".`
	user = fmt.Sprintf(`Today is %s.
Projects: %s
Statuses: %s
Assignees: %s
Labels: %s

Question: %s`, today.Format("Monday 2006-01-02"), cap40(v.Projects), strings.Join(statuses, ", "),
		cap40(v.Assignees), cap40(v.Labels), question)
	return system, user
}

// AIPlan is a validated model reply.
type AIPlan struct {
	Query    model.TaskQuery `json:"query"`
	Projects []string        `json:"projects"`
	Explain  string          `json:"explain"`
	Warnings []string        `json:"warnings"`
}

func pick(want []string, allowed func(string) (string, bool), what string, warn *[]string) []string {
	var out []string
	for _, w := range want {
		if c, ok := allowed(strings.TrimSpace(w)); ok {
			out = append(out, c)
		} else if strings.TrimSpace(w) != "" {
			*warn = append(*warn, fmt.Sprintf("ignored unknown %s %q", what, w))
		}
	}
	return out
}

func foldIn(list []string) func(string) (string, bool) {
	return func(v string) (string, bool) {
		for _, x := range list {
			if strings.EqualFold(x, v) {
				return x, true
			}
		}
		return "", false
	}
}

// ParseAIPlan validates the model's JSON against the vocabulary. Unknown
// values are dropped with a warning; malformed JSON or invalid dates are
// errors, so nothing runs on a broken filter.
func ParseAIPlan(reply string, v Vocab) (AIPlan, error) {
	reply = strings.TrimSpace(reply)
	if i := strings.Index(reply, "{"); i > 0 {
		reply = reply[i:]
	}
	if j := strings.LastIndex(reply, "}"); j >= 0 && j < len(reply)-1 {
		reply = reply[:j+1]
	}
	var raw AIPlan
	if err := json.Unmarshal([]byte(reply), &raw); err != nil {
		return AIPlan{}, fmt.Errorf("the model did not return a usable filter (%v)", err)
	}
	var plan AIPlan
	plan.Explain = strings.TrimSpace(raw.Explain)
	q := raw.Query
	statusOK := func(s string) (string, bool) {
		for id, label := range v.Statuses {
			if strings.EqualFold(id, s) || strings.EqualFold(label, s) {
				return id, true
			}
		}
		return "", false
	}
	q.Statuses = pick(q.Statuses, statusOK, "status", &plan.Warnings)
	q.Priorities = pick(q.Priorities, foldIn(v.Priorities), "priority", &plan.Warnings)
	q.Types = pick(q.Types, foldIn(v.Types), "type", &plan.Warnings)
	q.Assignees = pick(q.Assignees, foldIn(v.Assignees), "assignee", &plan.Warnings)
	q.Labels = pick(q.Labels, foldIn(v.Labels), "label", &plan.Warnings)
	plan.Projects = pick(raw.Projects, foldIn(v.Projects), "project", &plan.Warnings)
	q.Text = strings.TrimSpace(q.Text)
	q.Sprints = nil // sprint ids differ per project and are not offered
	if q.DuePreset != "" && !isPreset(q.DuePreset) {
		plan.Warnings = append(plan.Warnings, fmt.Sprintf("ignored unknown date range %q", q.DuePreset))
		q.DuePreset = ""
	}
	if err := query.ValidateQuery(q); err != nil {
		return AIPlan{}, fmt.Errorf("the model's filter was invalid: %v", err)
	}
	if query.Empty(q) && len(plan.Projects) == 0 {
		// An empty filter would list every task, which answers nothing.
		return AIPlan{}, fmt.Errorf("the model could not turn that into a filter; try naming a date, person, label or status")
	}
	plan.Query = q
	return plan, nil
}

// TaskResult renders one task as a search result.
func TaskResult(p *model.Project, t *model.Task, today time.Time) Result {
	done := t.Done || t.Status == "done"
	label := t.Status
	for _, c := range ghsync.EffectiveColumns(p) {
		if c.ID == t.Status {
			label = c.Label
		}
	}
	return Result{
		Kind: "task", ProjectID: p.ID, ProjectName: p.Name, TaskID: t.ID, Title: t.Title,
		Type: firstNonEmpty(t.Type, "task"), Status: t.Status, StatusLabel: label, Priority: t.Priority,
		DueDate: t.DueDate, Done: done, PastDue: daterange.IsPastDue(t.DueDate, done, day(today)),
	}
}

// RunPlan executes a validated plan over the real projects (optionally one).
func RunPlan(projects []model.Project, plan AIPlan, onlyProject string, global []model.QuarterRange, today time.Time) ([]Result, error) {
	var out []Result
	for i := range projects {
		p := &projects[i]
		if onlyProject != "" && p.ID != onlyProject {
			continue
		}
		if len(plan.Projects) > 0 && !containsFold(plan.Projects, p.Name) {
			continue
		}
		tasks, err := query.Run(plan.Query, *p, global, today)
		if err != nil {
			return nil, err
		}
		for ti := range tasks {
			out = append(out, TaskResult(p, &tasks[ti], today))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.DueDate == "") != (b.DueDate == "") {
			return a.DueDate != ""
		}
		if a.DueDate != b.DueDate {
			return a.DueDate < b.DueDate
		}
		return strings.ToLower(a.Title) < strings.ToLower(b.Title)
	})
	return out, nil
}
