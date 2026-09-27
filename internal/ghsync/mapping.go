// Package ghsync reconciles a JumpStart project's Kanban board with a
// GitHub Projects v2 board. It owns the mapping between the two data
// models, the conflict rules, and the polling loop that keeps them close
// to live without a webhook endpoint.
package ghsync

import (
	crand "crypto/rand"
	"fmt"
	"strings"
	"time"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

// columnAliases maps each local Kanban column to the Status option names
// GitHub boards commonly use, so a fresh link picks sensible defaults
// without the user wiring four dropdowns by hand.
var columnAliases = map[string][]string{
	"backlog":    {"backlog", "icebox", "triage", "no status"},
	"todo":       {"todo", "to do", "ready", "up next", "planned"},
	"inprogress": {"in progress", "inprogress", "doing", "started", "active"},
	"done":       {"done", "closed", "complete", "completed", "shipped", "resolved", "fixed"},
}

// BuildStatusMap pairs local column ids with option ids on the board's
// Status field. Options that match nothing are left unmapped, and the
// user can correct any pairing in the link panel.
func BuildStatusMap(f github.Field) map[string]string {
	out := map[string]string{}
	used := map[string]bool{}
	for col, aliases := range columnAliases {
		for _, opt := range f.Options {
			if used[opt.ID] {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(opt.Name))
			for _, a := range aliases {
				if name == a {
					out[col] = opt.ID
					used[opt.ID] = true
					break
				}
			}
			if out[col] != "" {
				break
			}
		}
	}
	return out
}

// FindStatusField returns the single-select field the columns map to,
// preferring one literally named "Status".
func FindStatusField(fields []github.Field) (github.Field, bool) {
	var fallback github.Field
	found := false
	for _, f := range fields {
		if f.DataType != github.FieldSingleSelect {
			continue
		}
		if strings.EqualFold(f.Name, "Status") {
			return f, true
		}
		if !found {
			fallback, found = f, true
		}
	}
	return fallback, found
}

// columnFor reverses the status map: given the option id on an item, it
// returns the local column that option belongs to.
func columnFor(statusMap map[string]string, optionID string) string {
	if optionID == "" {
		return ""
	}
	for col, id := range statusMap {
		if id == optionID {
			return col
		}
	}
	return ""
}

// applyRemote copies a GitHub item onto a local task, leaving purely
// local concepts (sprint membership, parent story) untouched.
// Acceptance criteria and subtasks that JumpStart encoded into the
// issue body are pulled back; unmarked body prose stays in Description.
// It returns true when anything actually changed.
func applyRemote(task *model.Task, item github.Item, cfg *model.GitHubSync) bool {
	changed := false

	if item.Title != "" && task.Title != item.Title {
		task.Title = item.Title
		changed = true
	}

	desc, acceptance, subtasks := ParseBody(item.Body)
	if task.Description != desc {
		task.Description = desc
		changed = true
	}
	// Only replace checklists when the body carried our markers.
	// Unmarked bodies leave local checklists alone so a hand-edited
	// GitHub issue cannot wipe JumpStart-only lists.
	if bodyHasChecklistMarkers(item.Body) {
		nextAcc := adoptChecklist(task.Acceptance, acceptance, newChecklistID)
		if !sameChecklist(task.Acceptance, nextAcc) {
			task.Acceptance = nextAcc
			changed = true
		}
		nextSubs := adoptChecklist(task.Subtasks, subtasks, newChecklistID)
		if !sameChecklist(task.Subtasks, nextSubs) {
			task.Subtasks = nextSubs
			changed = true
		}
	}

	if assignee := github.FormatAssignees(item.Assignees); assignee != task.Assignee {
		task.Assignee = assignee
		changed = true
	}
	if !sameStrings(task.Labels, item.Labels) {
		task.Labels = append([]string(nil), item.Labels...)
		changed = true
	}
	if task.Milestone != item.Milestone {
		task.Milestone = item.Milestone
		changed = true
	}
	if !sameStrings(task.Reviewers, item.Reviewers) {
		task.Reviewers = append([]string(nil), item.Reviewers...)
		changed = true
	}
	if !sameStrings(task.LinkedPRs, item.LinkedPRs) {
		task.LinkedPRs = append([]string(nil), item.LinkedPRs...)
		changed = true
	}
	if task.IssueType != item.IssueType {
		task.IssueType = item.IssueType
		changed = true
	}
	if task.ParentKey != item.ParentTitle {
		task.ParentKey = item.ParentTitle
		changed = true
	}

	// Column comes from the mapped Status option; a closed issue whose
	// status is unmapped still reads as done.
	if cfg != nil && cfg.StatusFieldID != "" {
		if v, ok := item.Values[cfg.StatusFieldID]; ok {
			if col := columnFor(cfg.StatusMap, v.OptionID); col != "" && col != task.Status {
				task.Status = col
				task.Done = col == "done"
				changed = true
			}
		}
	}
	if strings.EqualFold(item.State, "CLOSED") && task.Status != "done" {
		task.Status = "done"
		task.Done = true
		changed = true
	}

	// Every field value on the board lands in Fields, so custom columns
	// survive a round trip even when JumpStart has no native editor.
	next := map[string]model.FieldValue{}
	for id, v := range item.Values {
		next[id] = model.FieldValue{
			FieldID:   v.FieldID,
			Name:      v.FieldName,
			DataType:  v.DataType,
			Text:      v.Text,
			Number:    v.Number,
			Date:      v.Date,
			OptionID:  v.OptionID,
			OptionKey: v.OptionName,
			Iteration: v.IterationID,
			Display:   v.Display,
		}
	}
	if !sameFields(task.Fields, next) {
		task.Fields = next
		changed = true
	}

	if sp, ok := storyPointsFromValues(item.Values); ok && task.StoryPoints != sp {
		task.StoryPoints = sp
		changed = true
	}

	link := task.GitHub
	if link == nil {
		link = &model.GitHubLink{}
		task.GitHub = link
	}
	link.ItemID = item.ID
	link.ContentID = item.ContentID
	link.ContentType = item.ContentType
	link.Number = item.Number
	link.URL = item.URL
	link.Repo = item.Repo
	link.State = item.State
	link.RemoteUpdatedAt = parseTime(item.UpdatedAt)
	return changed
}

func bodyHasChecklistMarkers(body string) bool {
	return strings.Contains(body, acceptanceStart) || strings.Contains(body, subtasksStart)
}

// storyPointsFromValues reads a numeric field that looks like story
// points off the item's field map.
func storyPointsFromValues(values map[string]github.ItemFieldValue) (int, bool) {
	for _, v := range values {
		if v.DataType != github.FieldNumber || v.Number == nil {
			continue
		}
		n := strings.ToLower(v.FieldName)
		if strings.Contains(n, "point") || strings.Contains(n, "estimate") || n == "size" {
			return int(*v.Number), true
		}
	}
	return 0, false
}

func newChecklistID() string {
	// Prefer crypto/rand UUID when available; fall back to a time-based
	// id so tests and constrained environments still work.
	b := make([]byte, 16)
	if _, err := randRead(b); err == nil {
		b[6] = (b[6] & 0x0f) | 0x40
		b[8] = (b[8] & 0x3f) | 0x80
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}
	return fmt.Sprintf("gh-%d", time.Now().UnixNano())
}

// randRead is crypto/rand.Read, stubbed in tests if needed.
var randRead = func(b []byte) (int, error) {
	return crand.Read(b)
}

// storyPointsField finds a numeric field that looks like story points,
// so the local StoryPoints value has somewhere to go.
func storyPointsField(fields []github.Field) (github.Field, bool) {
	for _, f := range fields {
		if f.DataType != github.FieldNumber {
			continue
		}
		n := strings.ToLower(f.Name)
		if strings.Contains(n, "point") || strings.Contains(n, "estimate") || n == "size" {
			return f, true
		}
	}
	return github.Field{}, false
}

func parseTime(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameFields(a, b map[string]model.FieldValue) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || av.Display != bv.Display || av.OptionID != bv.OptionID ||
			av.Text != bv.Text || av.Date != bv.Date || av.Iteration != bv.Iteration {
			return false
		}
	}
	return true
}
