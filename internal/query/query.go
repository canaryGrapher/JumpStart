// Package query defines the one task filter shape shared by the board's
// filter panel, the sheet view, saved filters, dashboard widgets, MCP and
// Raycast, and evaluates it. Field names match the frontend (dueDates.js).
package query

import (
	"strings"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

// TaskQuery is the shared filter shape (defined in model so projects can
// store saved filters).
type TaskQuery = model.TaskQuery

// Empty reports whether the query constrains nothing.
func Empty(q TaskQuery) bool {
	return q.DuePreset == "" && q.DueFrom == "" && q.DueTo == "" && !q.NoDueDate && !q.Overdue &&
		len(q.Statuses) == 0 && len(q.Priorities) == 0 && len(q.Types) == 0 && len(q.Sprints) == 0 &&
		len(q.Assignees) == 0 && len(q.Labels) == 0 && q.Acceptance == "" && q.Subtasks == "" && strings.TrimSpace(q.Text) == ""
}

// DueRange resolves the preset and explicit bounds (explicit bounds narrow a
// preset). It returns nil when no date range applies.
func DueRange(q TaskQuery, today time.Time, quarters []model.QuarterRange) (*daterange.Range, error) {
	if q.DuePreset == "" && q.DueFrom == "" && q.DueTo == "" {
		return nil, nil
	}
	r := daterange.Range{}
	if q.DuePreset != "" {
		got, err := daterange.Resolve(q.DuePreset, today, quarters)
		if err != nil {
			return nil, err
		}
		r = got
	}
	for _, d := range []string{q.DueFrom, q.DueTo} {
		if d == "" {
			continue
		}
		if _, err := daterange.ParseDate(d); err != nil {
			return nil, err
		}
	}
	if q.DueFrom != "" && (r.From == "" || q.DueFrom > r.From) {
		r.From = q.DueFrom
	}
	if q.DueTo != "" && (r.To == "" || q.DueTo < r.To) {
		r.To = q.DueTo
	}
	return &r, nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func lowerAll(list []string) []string {
	out := make([]string, len(list))
	for i, v := range list {
		out[i] = strings.ToLower(strings.TrimSpace(v))
	}
	return out
}

// Match evaluates one task. due is the resolved range (nil for none).
func Match(q TaskQuery, t model.Task, due *daterange.Range, today time.Time) bool {
	done := t.Done || t.Status == "done"
	if len(q.Statuses) > 0 && !contains(q.Statuses, t.Status) {
		return false
	}
	if len(q.Priorities) > 0 {
		p := t.Priority
		if p == "" {
			p = "none"
		}
		if !contains(q.Priorities, p) {
			return false
		}
	}
	if len(q.Types) > 0 {
		ty := t.Type
		if ty == "" {
			ty = "task"
		}
		if !contains(q.Types, ty) {
			return false
		}
	}
	if len(q.Sprints) > 0 && !contains(q.Sprints, t.SprintID) {
		return false
	}
	if len(q.Assignees) > 0 {
		have := []string{}
		for _, a := range strings.Split(t.Assignee, ",") {
			if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
				have = append(have, a)
			}
		}
		hit := false
		for _, w := range lowerAll(q.Assignees) {
			if (w == "__none__" && len(have) == 0) || contains(have, w) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if len(q.Labels) > 0 {
		have := lowerAll(t.Labels)
		hit := false
		for _, w := range lowerAll(q.Labels) {
			if contains(have, w) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if (q.Acceptance == "has" && len(t.Acceptance) == 0) || (q.Acceptance == "none" && len(t.Acceptance) > 0) {
		return false
	}
	if (q.Subtasks == "has" && len(t.Subtasks) == 0) || (q.Subtasks == "none" && len(t.Subtasks) > 0) {
		return false
	}
	if q.NoDueDate && t.DueDate != "" {
		return false
	}
	if q.Overdue && !daterange.IsPastDue(t.DueDate, done, today) {
		return false
	}
	if due != nil && !due.Contains(t.DueDate) {
		return false
	}
	if text := strings.ToLower(strings.TrimSpace(q.Text)); text != "" {
		if !strings.Contains(strings.ToLower(t.Title), text) && !strings.Contains(strings.ToLower(t.Description), text) {
			return false
		}
	}
	return true
}

// Run filters a project's tasks, applying the project's effective quarters.
func Run(q TaskQuery, p model.Project, globalQuarters []model.QuarterRange, today time.Time) ([]model.Task, error) {
	due, err := DueRange(q, today, daterange.Effective(p.Quarters, globalQuarters))
	if err != nil {
		return nil, err
	}
	out := []model.Task{}
	for _, t := range p.Tasks {
		if Match(q, t, due, today) {
			out = append(out, t)
		}
	}
	return out, nil
}
