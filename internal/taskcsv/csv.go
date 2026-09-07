// Package taskcsv encodes and merges JumpStart kanban tasks as CSV so
// boards can be bulk-edited in a spreadsheet and uploaded back.
package taskcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"devdeck/internal/model"
)

// Header is the stable column order written by Encode and expected by Decode.
// Keep this list in sync with encodeRow / applyRow.
var Header = []string{
	"id",
	"title",
	"type",
	"status",
	"priority",
	"description",
	"assignee",
	"labels",
	"storyPoints",
	"parentId",
	"sprintId",
	"done",
}

// Result summarises one Apply pass.
type Result struct {
	Updated int `json:"updated"`
	Created int `json:"created"`
	Skipped int `json:"skipped"` // blank title rows
	Total   int `json:"total"`   // data rows in the file
}

// Progress reports how far Apply has got through the CSV rows.
type Progress func(done, total int)

// Encode writes tasks as a CSV document with Header as the first row.
// onProgress may be nil; when set it reports 1-based row progress through
// the task list (header write is done=0).
func Encode(w io.Writer, tasks []model.Task, onProgress Progress) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(Header); err != nil {
		return err
	}
	total := len(tasks)
	if onProgress != nil && total > 0 {
		onProgress(0, total)
	}
	for i, t := range tasks {
		if err := cw.Write(encodeRow(t)); err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(i+1, total)
		}
	}
	cw.Flush()
	return cw.Error()
}

// Decode reads a CSV produced by Encode (or a compatible edit of one).
// Extra columns are ignored; missing columns are treated as empty.
func Decode(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // allow short rows from hand edits
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("csv is empty")
	}
	return records, nil
}

// Apply merges CSV records into existing tasks. Rows whose id matches an
// existing task update that task; blank or unknown ids create new ones.
// Tasks not mentioned in the CSV are left alone (upload is additive /
// upsert, never a wipe). onProgress may be nil.
func Apply(existing []model.Task, records [][]string, newID func() string, onProgress Progress) ([]model.Task, Result, error) {
	if len(records) == 0 {
		return existing, Result{}, fmt.Errorf("csv is empty")
	}
	col := columnIndex(records[0])
	if _, ok := col["title"]; !ok {
		return existing, Result{}, fmt.Errorf("csv is missing a title column")
	}

	byID := make(map[string]int, len(existing))
	out := make([]model.Task, len(existing))
	copy(out, existing)
	for i, t := range out {
		if t.ID != "" {
			byID[t.ID] = i
		}
	}

	res := Result{Total: len(records) - 1}
	now := time.Now().UnixMilli()
	dataRows := records[1:]
	if onProgress != nil && len(dataRows) > 0 {
		onProgress(0, len(dataRows))
	}

	for i, row := range dataRows {
		title := cell(row, col, "title")
		if strings.TrimSpace(title) == "" {
			res.Skipped++
			if onProgress != nil {
				onProgress(i+1, len(dataRows))
			}
			continue
		}

		id := strings.TrimSpace(cell(row, col, "id"))
		if idx, ok := byID[id]; ok && id != "" {
			out[idx] = applyRow(out[idx], row, col, now)
			res.Updated++
		} else {
			t := model.Task{
				ID:        newID(),
				CreatedAt: now,
			}
			t = applyRow(t, row, col, now)
			// Prefer the spreadsheet's id when minting a new card so a
			// round-trip that cleared the board and re-uploaded keeps
			// stable references (parentId / sprintId links).
			if id != "" {
				t.ID = id
			}
			byID[t.ID] = len(out)
			out = append(out, t)
			res.Created++
		}
		if onProgress != nil {
			onProgress(i+1, len(dataRows))
		}
	}
	return out, res, nil
}

func encodeRow(t model.Task) []string {
	return []string{
		t.ID,
		t.Title,
		t.Type,
		t.Status,
		t.Priority,
		t.Description,
		t.Assignee,
		strings.Join(t.Labels, ";"),
		strconv.Itoa(t.StoryPoints),
		t.ParentID,
		t.SprintID,
		strconv.FormatBool(t.Done),
	}
}

func applyRow(t model.Task, row []string, col map[string]int, now int64) model.Task {
	t.Title = cell(row, col, "title")
	if v := cell(row, col, "type"); v != "" {
		t.Type = v
	} else if t.Type == "" {
		t.Type = "task"
	}
	if v := cell(row, col, "status"); v != "" {
		t.Status = v
	} else if t.Status == "" {
		t.Status = "todo"
	}
	t.Priority = cell(row, col, "priority")
	t.Description = cell(row, col, "description")
	t.Assignee = cell(row, col, "assignee")
	t.ParentID = cell(row, col, "parentId")
	t.SprintID = cell(row, col, "sprintId")

	if _, ok := col["labels"]; ok {
		t.Labels = splitLabels(cell(row, col, "labels"))
	}
	if _, ok := col["storyPoints"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(cell(row, col, "storyPoints"))); err == nil {
			t.StoryPoints = n
		}
	}
	if _, ok := col["done"]; ok {
		t.Done = parseBool(cell(row, col, "done"))
	} else {
		t.Done = t.Status == "done"
	}
	if t.Status == "done" {
		t.Done = true
	} else if t.Done && t.Status == "" {
		t.Status = "done"
	}
	t.UpdatedAt = now
	return t
}

func columnIndex(header []string) map[string]int {
	out := make(map[string]int, len(header))
	for i, h := range header {
		key := strings.TrimSpace(strings.ToLower(h))
		// Accept a few friendly aliases people type in spreadsheets.
		switch key {
		case "parent", "parent_id":
			key = "parentid"
		case "sprint", "sprint_id":
			key = "sprintid"
		case "story_points", "points":
			key = "storypoints"
		case "storypoints":
			key = "storypoints"
		case "parentid":
			key = "parentid"
		case "sprintid":
			key = "sprintid"
		}
		// Map normalised keys back to the canonical Header names we use
		// in cell().
		switch key {
		case "parentid":
			out["parentId"] = i
		case "sprintid":
			out["sprintId"] = i
		case "storypoints":
			out["storyPoints"] = i
		default:
			out[key] = i
		}
	}
	return out
}

func cell(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

func splitLabels(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ';' || r == '|' || r == ','
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "done":
		return true
	default:
		return false
	}
}
