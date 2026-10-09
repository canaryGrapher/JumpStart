// Package taskmerge applies task edits made against an older copy of the
// board without overwriting changes made elsewhere in the meantime (by MCP,
// agents, sync, or another window). Each edit carries the version of the
// task it started from (its base); only fields the editor actually changed
// are written over the stored task.
package taskmerge

import (
	"bytes"
	"encoding/json"

	"devdeck/internal/model"
)

// Change is one edited task. Base is the task as the editor last saw it;
// nil means the task is new.
type Change struct {
	Base *model.Task `json:"base"`
	Task model.Task  `json:"task"`
}

func fields(t model.Task) map[string]json.RawMessage {
	data, _ := json.Marshal(t)
	m := map[string]json.RawMessage{}
	_ = json.Unmarshal(data, &m)
	return m
}

func same(a, b json.RawMessage) bool {
	// Absent and empty values (omitempty) compare equal.
	empty := func(x json.RawMessage) bool {
		s := string(bytes.TrimSpace(x))
		return s == "" || s == "null" || s == `""` || s == "[]" || s == "{}" || s == "0" || s == "false"
	}
	if empty(a) && empty(b) {
		return true
	}
	return bytes.Equal(bytes.TrimSpace(a), bytes.TrimSpace(b))
}

// ThreeWay merges an edit: for every field, the editor's value wins if the
// editor changed it relative to base; otherwise the stored value wins.
func ThreeWay(base, mine, stored model.Task) model.Task {
	b, m, s := fields(base), fields(mine), fields(stored)
	out := map[string]json.RawMessage{}
	for k, v := range s {
		out[k] = v
	}
	keys := map[string]bool{}
	for k := range b {
		keys[k] = true
	}
	for k := range m {
		keys[k] = true
	}
	for k := range keys {
		if k == "updatedAt" {
			continue
		}
		if !same(m[k], b[k]) {
			if v, ok := m[k]; ok {
				out[k] = v
			} else {
				delete(out, k) // the editor cleared it
			}
		}
	}
	data, _ := json.Marshal(out)
	var t model.Task
	_ = json.Unmarshal(data, &t)
	if mine.UpdatedAt > t.UpdatedAt {
		t.UpdatedAt = mine.UpdatedAt
	}
	return t
}

// Apply merges changes and deletions into the stored task list and returns
// the new list. Order is preserved; new tasks are appended.
//   - A change whose task no longer exists is re-added only if it is new
//     (Base == nil); an edit to a task deleted elsewhere is dropped.
//   - A deletion of a task that is already gone is ignored.
func Apply(stored []model.Task, changes []Change, deletes []string) []model.Task {
	gone := map[string]bool{}
	for _, id := range deletes {
		gone[id] = true
	}
	index := map[string]int{}
	out := make([]model.Task, 0, len(stored)+len(changes))
	for _, t := range stored {
		if gone[t.ID] {
			continue
		}
		index[t.ID] = len(out)
		out = append(out, t)
	}
	for _, c := range changes {
		if gone[c.Task.ID] {
			continue
		}
		i, exists := index[c.Task.ID]
		switch {
		case exists && c.Base != nil:
			out[i] = ThreeWay(*c.Base, c.Task, out[i])
		case exists && c.Base == nil:
			out[i] = c.Task // same id created twice: the latest write wins
		case c.Base == nil:
			index[c.Task.ID] = len(out)
			out = append(out, c.Task)
		}
	}
	return out
}
