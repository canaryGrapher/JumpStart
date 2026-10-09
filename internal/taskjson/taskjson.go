// Package taskjson exports a whole project (fields, sprints, columns,
// quarters, tasks with every field and status) as JSON and merges edited
// JSON back in. Imports are previewed before they are applied.
package taskjson

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"devdeck/internal/attachments"
	"devdeck/internal/daterange"
	"devdeck/internal/ghsync"
	"devdeck/internal/model"
)

// Format identifies JumpStart project exports.
const Format = "jumpstart-project"

// Redacted replaces environment values in exports unless the caller opts in.
const Redacted = "<redacted>"

// Document is the export envelope.
type Document struct {
	Format     string        `json:"format"`
	Version    int           `json:"version"`
	ExportedAt string        `json:"exportedAt"`
	Project    model.Project `json:"project"`
	// Notes explains what the export does not carry.
	Notes []string `json:"notes,omitempty"`
}

// Export renders a project. Process environment values are replaced with
// Redacted unless includeEnv is true, because they often hold secrets.
func Export(p model.Project, includeEnv bool, now time.Time) ([]byte, error) {
	cp := p
	cp.Processes = make([]model.Process, len(p.Processes))
	for i, proc := range p.Processes {
		c := proc
		if !includeEnv && len(proc.Env) > 0 {
			c.Env = make(map[string]string, len(proc.Env))
			for k := range proc.Env {
				c.Env[k] = Redacted
			}
		}
		cp.Processes[i] = c
	}
	doc := Document{
		Format: Format, Version: 1, ExportedAt: now.UTC().Format(time.RFC3339), Project: cp,
		Notes: []string{
			"Attachment entries list files; the files themselves are not included.",
			"On import, root, processes, icon and GitHub settings are left unchanged. Edit columns with Edit board.",
		},
	}
	// Plain characters (not \u003c escapes) keep the file easy to read and edit.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Mode controls how an import merges.
type Mode string

const (
	// ModeMerge updates tasks matched by id and adds new ones.
	ModeMerge Mode = "merge"
	// ModeReplace also removes tasks that are absent from the import.
	ModeReplace Mode = "replace"
)

// Change describes one updated task in a preview.
type Change struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
}

// Preview summarizes what an import would do.
type Preview struct {
	Added     []string `json:"added"`
	Updated   []Change `json:"updated"`
	Removed   []string `json:"removed"`
	Unchanged int      `json:"unchanged"`
	Project   []string `json:"project"` // project-level fields that change
	Sprints   []string `json:"sprints"` // sprints that would be created
	Warnings  []string `json:"warnings"`
}

// incoming accepts a full export, a bare project, or {"tasks":[...]}.
type incoming struct {
	Format  string         `json:"format"`
	Project *model.Project `json:"project"`
	Tasks   []model.Task   `json:"tasks"`
	Name    *string        `json:"name"`
}

func decode(data []byte) (model.Project, bool, error) {
	var in incoming
	if err := json.Unmarshal(data, &in); err != nil {
		return model.Project{}, false, fmt.Errorf("not valid JSON: %w", err)
	}
	if in.Project != nil {
		return *in.Project, true, nil
	}
	// A bare project object or just a task list.
	var bare model.Project
	if err := json.Unmarshal(data, &bare); err != nil {
		return model.Project{}, false, err
	}
	hasProject := in.Name != nil
	if bare.Tasks == nil {
		bare.Tasks = in.Tasks
	}
	return bare, hasProject, nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]), hex.EncodeToString(b[10:16]))
}

// validateTask checks the same rules as the MCP tools and normalizes links.
func validateTask(t *model.Task, columns map[string]bool) error {
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("a task has no title")
	}
	if t.Status == "" {
		t.Status = "todo"
	}
	if !columns[t.Status] {
		known := make([]string, 0, len(columns))
		for c := range columns {
			known = append(known, c)
		}
		sort.Strings(known)
		return fmt.Errorf("task %q has status %q, which is not a column on this board (%s)", t.Title, t.Status, strings.Join(known, ", "))
	}
	if t.DueDate != "" {
		if _, err := daterange.ParseDate(t.DueDate); err != nil {
			return fmt.Errorf("task %q: dueDate must be YYYY-MM-DD, got %q", t.Title, t.DueDate)
		}
	}
	for i := range t.Links {
		u, err := attachments.NormalizeURL(t.Links[i].URL)
		if err != nil {
			return fmt.Errorf("task %q: link %q: %w", t.Title, t.Links[i].URL, err)
		}
		t.Links[i].URL = u
		if t.Links[i].ID == "" {
			t.Links[i].ID = newID()
		}
	}
	for _, list := range []*[]model.Subtask{&t.Subtasks, &t.Acceptance} {
		for i := range *list {
			if strings.TrimSpace((*list)[i].Title) == "" {
				return fmt.Errorf("task %q has an empty checklist item", t.Title)
			}
			if (*list)[i].ID == "" {
				(*list)[i].ID = newID()
			}
		}
	}
	if t.Type == "" {
		t.Type = "task"
	}
	t.Done = t.Status == "done"
	return nil
}

// diffFields names the task fields that differ, ignoring bookkeeping.
func diffFields(a, b model.Task) []string {
	ignore := map[string]bool{"UpdatedAt": true, "CreatedAt": true, "GitHub": true, "Attachments": true, "Fields": true}
	var out []string
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	tt := va.Type()
	for i := 0; i < tt.NumField(); i++ {
		f := tt.Field(i)
		if ignore[f.Name] {
			continue
		}
		x, y := va.Field(i).Interface(), vb.Field(i).Interface()
		if isEmpty(x) && isEmpty(y) {
			continue
		}
		if !reflect.DeepEqual(x, y) {
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			out = append(out, name)
		}
	}
	return out
}

func isEmpty(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	}
	return rv.IsZero()
}

// normalized returns the preview with empty (never nil) lists, so callers
// and JSON consumers can rely on arrays.
func (pv Preview) normalized() Preview {
	for _, l := range []*[]string{&pv.Added, &pv.Removed, &pv.Project, &pv.Sprints, &pv.Warnings} {
		if *l == nil {
			*l = []string{}
		}
	}
	if pv.Updated == nil {
		pv.Updated = []Change{}
	}
	return pv
}

// Plan merges data into a copy of current and reports the changes. It does
// not modify current. now stamps updated and created tasks.
func Plan(current model.Project, data []byte, mode Mode, now time.Time) (model.Project, Preview, error) {
	out, pv, err := plan(current, data, mode, now)
	return out, pv.normalized(), err
}

func plan(current model.Project, data []byte, mode Mode, now time.Time) (model.Project, Preview, error) {
	var pv Preview
	if mode == "" {
		mode = ModeMerge
	}
	if mode != ModeMerge && mode != ModeReplace {
		return current, pv, fmt.Errorf("mode must be merge or replace")
	}
	in, hasProject, err := decode(data)
	if err != nil {
		return current, pv, err
	}
	if in.ID != "" && current.ID != "" && in.ID != current.ID {
		pv.Warnings = append(pv.Warnings, fmt.Sprintf("The JSON is from another project (%s); its tasks are merged into this one.", in.Name))
	}

	out := current
	out.Tasks = append([]model.Task(nil), current.Tasks...)
	out.Sprints = append([]model.Sprint(nil), current.Sprints...)

	// Project-level fields that are safe to edit from JSON.
	if hasProject {
		if strings.TrimSpace(in.Name) != "" && in.Name != current.Name {
			out.Name = in.Name
			pv.Project = append(pv.Project, "name")
		}
		if in.Description != current.Description {
			out.Description = in.Description
			pv.Project = append(pv.Project, "description")
		}
		if len(in.Quarters) > 0 && !reflect.DeepEqual(in.Quarters, current.Quarters) {
			if err := daterange.ValidateQuarters(in.Quarters); err != nil {
				return current, pv, fmt.Errorf("quarters: %w", err)
			}
			out.Quarters = in.Quarters
			pv.Project = append(pv.Project, "quarters")
		}
		if in.Root != "" && in.Root != current.Root {
			pv.Warnings = append(pv.Warnings, "Root folder in the JSON differs; it was not changed.")
		}
		if len(in.Columns) > 0 && !reflect.DeepEqual(in.Columns, current.Columns) {
			pv.Warnings = append(pv.Warnings, "Columns in the JSON were not applied; use Edit board to change columns.")
		}
	}

	// Sprints: match by id, then by name; create the rest.
	sprintByID := map[string]string{}
	byName := map[string]string{}
	for _, s := range out.Sprints {
		sprintByID[s.ID] = s.ID
		byName[strings.ToLower(s.Name)] = s.ID
	}
	for _, s := range in.Sprints {
		if _, ok := sprintByID[s.ID]; ok && s.ID != "" {
			continue
		}
		if id, ok := byName[strings.ToLower(s.Name)]; ok {
			sprintByID[s.ID] = id
			continue
		}
		if strings.TrimSpace(s.Name) == "" {
			continue
		}
		ns := s
		if ns.ID == "" {
			ns.ID = newID()
		}
		ns.Order = len(out.Sprints)
		if ns.CreatedAt == 0 {
			ns.CreatedAt = now.UnixMilli()
		}
		out.Sprints = append(out.Sprints, ns)
		sprintByID[s.ID] = ns.ID
		byName[strings.ToLower(ns.Name)] = ns.ID
		pv.Sprints = append(pv.Sprints, ns.Name)
	}

	columns := map[string]bool{}
	for _, c := range ghsync.EffectiveColumns(&out) {
		columns[c.ID] = true
	}
	index := map[string]int{}
	for i, t := range out.Tasks {
		index[t.ID] = i
	}
	seen := map[string]bool{}
	for _, t := range in.Tasks {
		t := t
		if t.SprintID != "" {
			if id, ok := sprintByID[t.SprintID]; ok {
				t.SprintID = id
			} else {
				pv.Warnings = append(pv.Warnings, fmt.Sprintf("Task %q points to an unknown sprint; it was put in the backlog.", t.Title))
				t.SprintID = ""
			}
		}
		if err := validateTask(&t, columns); err != nil {
			return current, pv, err
		}
		if i, ok := index[t.ID]; ok && t.ID != "" {
			if seen[t.ID] {
				return current, pv, fmt.Errorf("task id %q appears twice", t.ID)
			}
			seen[t.ID] = true
			prev := out.Tasks[i]
			// Files and GitHub links can't come from JSON; keep the stored ones.
			t.Attachments = prev.Attachments
			t.GitHub = prev.GitHub
			if t.Fields == nil {
				t.Fields = prev.Fields
			}
			t.CreatedAt = prev.CreatedAt
			fields := diffFields(prev, t)
			if len(fields) == 0 {
				pv.Unchanged++
				continue
			}
			t.UpdatedAt = now.UnixMilli()
			out.Tasks[i] = t
			pv.Updated = append(pv.Updated, Change{ID: t.ID, Title: t.Title, Fields: fields})
			continue
		}
		if t.ID == "" || seen[t.ID] {
			t.ID = newID()
		}
		seen[t.ID] = true
		t.Attachments = nil // files are not part of JSON
		t.GitHub = nil
		if t.CreatedAt == 0 {
			t.CreatedAt = now.UnixMilli()
		}
		t.UpdatedAt = now.UnixMilli()
		out.Tasks = append(out.Tasks, t)
		index[t.ID] = len(out.Tasks) - 1
		pv.Added = append(pv.Added, t.Title)
	}

	if mode == ModeReplace {
		kept := out.Tasks[:0]
		for _, t := range out.Tasks {
			if seen[t.ID] {
				kept = append(kept, t)
				continue
			}
			pv.Removed = append(pv.Removed, t.Title)
		}
		out.Tasks = kept
	}

	// Parent links must point at a task that still exists.
	ids := map[string]bool{}
	for _, t := range out.Tasks {
		ids[t.ID] = true
	}
	for i := range out.Tasks {
		if p := out.Tasks[i].ParentID; p != "" && !ids[p] {
			pv.Warnings = append(pv.Warnings, fmt.Sprintf("Task %q pointed to a missing parent; it is now top-level.", out.Tasks[i].Title))
			out.Tasks[i].ParentID = ""
		}
	}
	return out, pv, nil
}
