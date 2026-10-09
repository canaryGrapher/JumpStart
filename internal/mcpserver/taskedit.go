package mcpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"devdeck/internal/attachments"
	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

// ChecklistItem is one subtask or acceptance criterion in a tool call.
// Done is a pointer so omitting it keeps an existing item's checked state.
type ChecklistItem struct {
	ID    string `json:"id,omitempty" jsonschema:"Existing item id to keep; omit to match by title or create"`
	Title string `json:"title" jsonschema:"Item text"`
	Done  *bool  `json:"done,omitempty" jsonschema:"Checked state; omit to keep the existing state (new items start unchecked)"`
}

// LinkItem is one hyperlink in a tool call.
type LinkItem struct {
	ID    string `json:"id,omitempty" jsonschema:"Existing link id to keep; omit to match by URL or create"`
	Title string `json:"title,omitempty" jsonschema:"Link label"`
	URL   string `json:"url" jsonschema:"http, https, or mailto URL"`
}

// clearable lists the field names accepted by upsert_task's clear argument.
var clearable = map[string]bool{
	"assignee": true, "description": true, "priority": true, "sprintId": true,
	"parentId": true, "storyPoints": true, "dueDate": true, "milestone": true,
	"labels": true, "subtasks": true, "acceptance": true, "links": true,
}

func validateClear(fields []string) error {
	for _, f := range fields {
		if !clearable[f] {
			return fmt.Errorf("cannot clear %q; allowed: assignee, description, priority, sprintId, parentId, storyPoints, dueDate, milestone, labels, subtasks, acceptance, links", f)
		}
	}
	return nil
}

func applyClear(t *model.Task, fields []string) {
	for _, f := range fields {
		switch f {
		case "assignee":
			t.Assignee = ""
		case "description":
			t.Description = ""
		case "priority":
			t.Priority = ""
		case "sprintId":
			t.SprintID = ""
		case "parentId":
			t.ParentID = ""
		case "storyPoints":
			t.StoryPoints = 0
		case "dueDate":
			t.DueDate = ""
		case "milestone":
			t.Milestone = ""
		case "labels":
			t.Labels = nil
		case "subtasks":
			t.Subtasks = nil
		case "acceptance":
			t.Acceptance = nil
		case "links":
			t.Links = nil
		}
	}
}

// validateDueDate accepts "" (no change) or a real YYYY-MM-DD date.
func validateDueDate(s string) error {
	if s == "" {
		return nil
	}
	if _, err := daterange.ParseDate(s); err != nil {
		return fmt.Errorf("dueDate must be YYYY-MM-DD, got %q", s)
	}
	return nil
}

// mergeChecklist builds the new checklist from the requested items,
// preserving the id and checked state of items that already exist. Items
// match by id first, then by case-insensitive title.
func mergeChecklist(existing []model.Subtask, items []ChecklistItem) ([]model.Subtask, error) {
	byID := map[string]model.Subtask{}
	byTitle := map[string]model.Subtask{}
	for _, e := range existing {
		byID[e.ID] = e
		key := strings.ToLower(strings.TrimSpace(e.Title))
		if _, dup := byTitle[key]; !dup {
			byTitle[key] = e
		}
	}
	out := make([]model.Subtask, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			return nil, fmt.Errorf("checklist items need a non-empty title")
		}
		var base model.Subtask
		if prev, ok := byID[it.ID]; ok && it.ID != "" {
			base = prev
		} else if prev, ok := byTitle[strings.ToLower(title)]; ok && !seen[prev.ID] {
			base = prev
		} else {
			base = model.Subtask{ID: uuid.NewString()}
		}
		seen[base.ID] = true
		base.Title = title
		if it.Done != nil {
			base.Done = *it.Done
		}
		out = append(out, base)
	}
	return out, nil
}

// mergeLinks validates and normalizes link URLs, preserving ids of links
// that already exist (matched by id, then by normalized URL).
func mergeLinks(existing []model.TaskLink, items []LinkItem) ([]model.TaskLink, error) {
	byID := map[string]model.TaskLink{}
	byURL := map[string]model.TaskLink{}
	for _, e := range existing {
		byID[e.ID] = e
		byURL[e.URL] = e
	}
	out := make([]model.TaskLink, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		u, err := attachments.NormalizeURL(it.URL)
		if err != nil {
			return nil, fmt.Errorf("link %q: %w", it.URL, err)
		}
		var base model.TaskLink
		if prev, ok := byID[it.ID]; ok && it.ID != "" {
			base = prev
		} else if prev, ok := byURL[u]; ok && !seen[prev.ID] {
			base = prev
		} else {
			base = model.TaskLink{ID: uuid.NewString()}
		}
		seen[base.ID] = true
		base.URL = u
		base.Title = strings.TrimSpace(it.Title)
		out = append(out, base)
	}
	return out, nil
}

// withStatus sets the status and keeps the legacy Done flag in step, the
// same rule the board applies when a card is dragged.
func withStatus(t *model.Task, status string) {
	t.Status = status
	t.Done = status == "done"
}

// appendNote adds a timestamped note to a task description.
func appendNote(t *model.Task, note string, now time.Time) {
	entry := fmt.Sprintf("Note (%s): %s", now.UTC().Format(time.RFC3339), strings.TrimSpace(note))
	if strings.TrimSpace(t.Description) == "" {
		t.Description = entry
		return
	}
	t.Description = strings.TrimRight(t.Description, "\n") + "\n\n" + entry
}

// TaskFilter selects tasks for list_tasks. Comma-separated values match any.
type TaskFilter struct {
	Status         string
	Priority       string
	Type           string
	SprintID       string
	Assignee       string
	Label          string
	ParentID       string
	HasAcceptance  *bool
	HasSubtasks    *bool
	DuePreset      string
	DueFrom        string
	DueTo          string
	Overdue        *bool
	NoDueDate      *bool
	HasLinks       *bool
	HasAttachments *bool
}

func csvSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
			out[p] = true
		}
	}
	return out
}

// Match applies the filter. due is the resolved date range, or nil when no
// due-date range was requested.
func (f TaskFilter) Match(t model.Task, due *daterange.Range, today time.Time) bool {
	if set := csvSet(f.Status); len(set) > 0 && !set[strings.ToLower(t.Status)] {
		return false
	}
	if set := csvSet(f.Priority); len(set) > 0 && !set[strings.ToLower(t.Priority)] {
		return false
	}
	if set := csvSet(f.Type); len(set) > 0 {
		ty := strings.ToLower(t.Type)
		if ty == "" {
			ty = "task"
		}
		if !set[ty] {
			return false
		}
	}
	if f.SprintID != "" {
		want := f.SprintID
		if want == "backlog" {
			want = ""
		}
		if t.SprintID != want {
			return false
		}
	}
	if f.ParentID != "" && t.ParentID != f.ParentID {
		return false
	}
	if set := csvSet(f.Assignee); len(set) > 0 && !anyIn(set, strings.Split(t.Assignee, ",")) {
		return false
	}
	if set := csvSet(f.Label); len(set) > 0 && !anyIn(set, t.Labels) {
		return false
	}
	if f.HasAcceptance != nil && (len(t.Acceptance) > 0) != *f.HasAcceptance {
		return false
	}
	if f.HasSubtasks != nil && (len(t.Subtasks) > 0) != *f.HasSubtasks {
		return false
	}
	if f.HasLinks != nil && (len(t.Links) > 0) != *f.HasLinks {
		return false
	}
	if f.HasAttachments != nil && (len(t.Attachments) > 0) != *f.HasAttachments {
		return false
	}
	if f.NoDueDate != nil && (t.DueDate == "") != *f.NoDueDate {
		return false
	}
	if f.Overdue != nil && daterange.IsPastDue(t.DueDate, t.Done || t.Status == "done", today) != *f.Overdue {
		return false
	}
	if due != nil && !due.Contains(t.DueDate) {
		return false
	}
	return true
}

func anyIn(set map[string]bool, values []string) bool {
	for _, v := range values {
		if set[strings.ToLower(strings.TrimSpace(v))] {
			return true
		}
	}
	return false
}

// DueRange combines the preset and explicit bounds. Explicit dueFrom/dueTo
// narrow a preset; either may be given alone.
func (f TaskFilter) DueRange(today time.Time, project, global []model.QuarterRange) (*daterange.Range, error) {
	if f.DuePreset == "" && f.DueFrom == "" && f.DueTo == "" {
		return nil, nil
	}
	r := daterange.Range{}
	if f.DuePreset != "" {
		got, err := daterange.Resolve(f.DuePreset, today, daterange.Effective(project, global))
		if err != nil {
			return nil, err
		}
		r = got
	}
	for _, d := range []string{f.DueFrom, f.DueTo} {
		if err := validateDueDate(d); err != nil {
			return nil, err
		}
	}
	if f.DueFrom != "" && (r.From == "" || f.DueFrom > r.From) {
		r.From = f.DueFrom
	}
	if f.DueTo != "" && (r.To == "" || f.DueTo < r.To) {
		r.To = f.DueTo
	}
	return &r, nil
}
