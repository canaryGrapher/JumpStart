package main

import (
	"time"

	"devdeck/internal/model"
)

// Kanban and planning events (taxonomy 4.5).
//
// The frontend saves the whole task list on every change, so these are
// derived by diffing the old list against the new one. That is deliberate:
// a diff cannot forget to fire when a new UI path starts editing tasks, and
// it produces task_moved with a real age rather than one the UI guesses at.

// trackTaskChanges emits task_created and task_moved for one UpdateTasks
// call. A save that changed neither emits nothing, which keeps drag
// reorders and text edits out of the event stream.
func (a *App) trackTaskChanges(projectID string, before, after []model.Task) {
	old := make(map[string]model.Task, len(before))
	for _, t := range before {
		old[t.ID] = t
	}

	ref := a.ref(projectID)
	for _, t := range after {
		prev, existed := old[t.ID]
		if !existed {
			a.track("task_created", map[string]any{
				"project_ref":   ref,
				"column":        column(t.Status),
				"kind":          taskKind(t.Type),
				"has_sprint":    t.SprintID != "",
				"ai_enriched":   len(t.Acceptance) > 0 || len(t.Subtasks) > 0,
				"has_estimate":  t.StoryPoints > 0,
				"subtask_count": len(t.Subtasks),
				"labelled":      len(t.Labels) > 0,
				"is_child":      t.ParentID != "",
				"priority":      priority(t.Priority),
				"title_length":  len(t.Title),
				"has_body":      t.Description != "",
			})
			continue
		}
		if column(prev.Status) == column(t.Status) {
			continue
		}
		// age_hours turns the board into a cycle-time measurement, which is
		// the only number that shows the kanban is used for real work
		// rather than tried once and abandoned.
		a.track("task_moved", map[string]any{
			"project_ref": ref,
			"from_column": column(prev.Status),
			"to_column":   column(t.Status),
			"kind":        taskKind(t.Type),
			"age_hours":   ageHours(t.CreatedAt),
		})
	}
}

// trackSprintChanges emits sprint_created for sprints that are new in this
// save.
func (a *App) trackSprintChanges(projectID string, before, after []model.Sprint) {
	old := make(map[string]bool, len(before))
	for _, s := range before {
		old[s.ID] = true
	}
	for _, s := range after {
		if old[s.ID] {
			continue
		}
		a.track("sprint_created", map[string]any{
			"project_ref":   a.ref(projectID),
			"has_goal":      s.Goal != "",
			"has_dates":     s.StartDate != "" && s.EndDate != "",
			"duration_days": sprintDays(s),
			"sprint_index":  s.Order,
		})
	}
}

// column bounds the kanban status. An empty status is the backlog.
func column(status string) string {
	switch status {
	case "todo", "inprogress", "testing", "done", "backlog":
		return status
	case "":
		return "backlog"
	}
	return "other"
}

func taskKind(kind string) string {
	switch kind {
	case "story", "bug":
		return kind
	}
	return "task"
}

func priority(p string) string {
	switch p {
	case "low", "medium", "high":
		return p
	}
	return "none"
}

// ageHours is how long a task existed before this move. A zero or absent
// CreatedAt reports 0 rather than a nonsense age from the unix epoch.
func ageHours(createdAt int64) int64 {
	if createdAt <= 0 {
		return 0
	}
	created := time.UnixMilli(createdAt)
	hours := int64(time.Since(created).Hours())
	if hours < 0 {
		return 0
	}
	return hours
}

// sprintDays is the sprint's planned length in days, or 0 when it has no
// dates. Dates are "YYYY-MM-DD" strings in the model.
func sprintDays(s model.Sprint) int {
	const layout = "2006-01-02"
	start, err := time.Parse(layout, s.StartDate)
	if err != nil {
		return 0
	}
	end, err := time.Parse(layout, s.EndDate)
	if err != nil {
		return 0
	}
	days := int(end.Sub(start).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
