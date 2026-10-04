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
	"sprint",
	"done",
	"subtasks",
	"acceptance",
	"createdAt",
	"updatedAt",
	"milestone",
	"issueType",
	"parentKey",
	"reviewers",
	"linkedPrs",
}

// Mode controls how Apply merges CSV rows with the existing board.
type Mode string

const (
	// ModeAdd upserts by id and leaves tasks missing from the CSV alone
	// (incremental / additive import).
	ModeAdd Mode = "add"
	// ModeReplace upserts by id, then drops any existing task that was not
	// mentioned in the CSV — the spreadsheet becomes the full board.
	ModeReplace Mode = "replace"

	// SprintScopeAll leaves sprint membership to the CSV sprint/sprintId
	// columns. Any other value (including "") forces imported rows onto
	// that sprint id; "" is the backlog. Replace mode then only removes
	// untouched tasks inside that same sprint board.
	SprintScopeAll = "__all__"
)

// ParseMode maps a UI/API string onto Mode. Unknown values default to Add.
func ParseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(ModeReplace), "replace-all", "overwrite":
		return ModeReplace
	default:
		return ModeAdd
	}
}

// Result summarises one Apply pass.
type Result struct {
	Updated        int `json:"updated"`
	Created        int `json:"created"`
	Removed        int `json:"removed"`        // ModeReplace only: existing tasks dropped
	Skipped        int `json:"skipped"`        // blank title rows
	Total          int `json:"total"`          // data rows in the file
	SprintsCreated int `json:"sprintsCreated"` // new local sprints minted from sprint names
}

// Progress reports how far Apply has got through the CSV rows.
type Progress func(done, total int)

// Encode writes tasks as a CSV document with Header as the first row.
// sprints supplies human-readable names for the sprint column. onProgress
// may be nil; when set it reports 1-based row progress through the task
// list (header write is done=0).
func Encode(w io.Writer, tasks []model.Task, sprints []model.Sprint, onProgress Progress) error {
	names := make(map[string]string, len(sprints))
	for _, s := range sprints {
		if s.ID != "" {
			names[s.ID] = s.Name
		}
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(Header); err != nil {
		return err
	}
	total := len(tasks)
	if onProgress != nil && total > 0 {
		onProgress(0, total)
	}
	for i, t := range tasks {
		if err := cw.Write(encodeRow(t, names)); err != nil {
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

// Apply merges CSV records into existing tasks according to mode.
//
// ModeAdd (default): rows whose id matches an existing task update that
// task; blank or unknown ids create new ones. Tasks not mentioned in the
// CSV are left alone.
//
// ModeReplace: same upsert rules, then any existing task whose id was not
// present in the CSV is removed so the spreadsheet becomes the board.
// When sprintScope is not SprintScopeAll, only untouched tasks that
// already belong to that sprint board are removed.
//
// sprintScope is SprintScopeAll to honour the CSV sprint/sprintId
// columns, or a sprint id ("" = backlog) to force every imported row
// onto that board. Unknown names still mint new local sprints when
// reading from the CSV. onProgress may be nil.
func Apply(existing []model.Task, sprints []model.Sprint, records [][]string, mode Mode, sprintScope string, newID func() string, onProgress Progress) ([]model.Task, []model.Sprint, Result, error) {
	if mode == "" {
		mode = ModeAdd
	}
	// sprintScope "" is the backlog when forced; only "__all__" defers to CSV.
	forceSprint := sprintScope != SprintScopeAll
	if len(records) == 0 {
		return existing, sprints, Result{}, fmt.Errorf("csv is empty")
	}
	col := columnIndex(records[0])
	if _, ok := col["title"]; !ok {
		return existing, sprints, Result{}, fmt.Errorf("csv is missing a title column")
	}

	byID := make(map[string]int, len(existing))
	out := make([]model.Task, len(existing))
	copy(out, existing)
	for i, t := range out {
		if t.ID != "" {
			byID[t.ID] = i
		}
	}

	sprintOut := make([]model.Sprint, len(sprints))
	copy(sprintOut, sprints)
	sprintByName := make(map[string]string, len(sprintOut))
	sprintByID := make(map[string]bool, len(sprintOut))
	for _, s := range sprintOut {
		if s.ID != "" {
			sprintByID[s.ID] = true
		}
		if name := strings.TrimSpace(s.Name); name != "" {
			sprintByName[strings.ToLower(name)] = s.ID
		}
	}

	res := Result{Total: len(records) - 1}
	now := time.Now().UnixMilli()
	dataRows := records[1:]
	touched := make(map[string]bool, len(dataRows))
	if onProgress != nil && len(dataRows) > 0 {
		onProgress(0, len(dataRows))
	}

	assignSprint := func(row []string) string {
		if forceSprint {
			return sprintScope
		}
		return resolveSprintID(row, col, &sprintOut, sprintByName, sprintByID, newID, now, &res)
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
			t := ensureChecklistIDs(applyRow(out[idx], row, col, now), newID)
			t.SprintID = assignSprint(row)
			out[idx] = t
			touched[out[idx].ID] = true
			res.Updated++
		} else {
			t := model.Task{
				ID:        newID(),
				CreatedAt: now,
			}
			t = ensureChecklistIDs(applyRow(t, row, col, now), newID)
			// Prefer the spreadsheet's id when minting a new card so a
			// round-trip that cleared the board and re-uploaded keeps
			// stable references (parentId / sprintId links).
			if id != "" {
				t.ID = id
			}
			t.SprintID = assignSprint(row)
			byID[t.ID] = len(out)
			touched[t.ID] = true
			out = append(out, t)
			res.Created++
		}
		if onProgress != nil {
			onProgress(i+1, len(dataRows))
		}
	}

	if mode == ModeReplace {
		kept := make([]model.Task, 0, len(out))
		for _, t := range out {
			if touched[t.ID] {
				kept = append(kept, t)
				continue
			}
			// Scoped replace: leave other sprint boards alone.
			if forceSprint && t.SprintID != sprintScope {
				kept = append(kept, t)
				continue
			}
			res.Removed++
		}
		out = kept
	}

	return out, sprintOut, res, nil
}

// resolveSprintID prefers the human-readable sprint column. "Backlog"
// (any case) clears membership. Unknown names create a planned sprint.
// When only sprintId is present, that id is kept as-is.
func resolveSprintID(row []string, col map[string]int, sprints *[]model.Sprint, byName map[string]string, byID map[string]bool, newID func() string, now int64, res *Result) string {
	name := strings.TrimSpace(cell(row, col, "sprint"))
	id := strings.TrimSpace(cell(row, col, "sprintId"))

	if _, hasName := col["sprint"]; hasName {
		if name == "" || strings.EqualFold(name, "backlog") {
			return ""
		}
		key := strings.ToLower(name)
		if existing, ok := byName[key]; ok {
			return existing
		}
		s := model.Sprint{
			ID:        newID(),
			Name:      name,
			Status:    "planned",
			Order:     len(*sprints),
			CreatedAt: now,
		}
		*sprints = append(*sprints, s)
		byName[key] = s.ID
		byID[s.ID] = true
		res.SprintsCreated++
		return s.ID
	}

	if id == "" {
		return ""
	}
	return id
}

func encodeRow(t model.Task, sprintNames map[string]string) []string {
	sprintName := ""
	if t.SprintID != "" {
		sprintName = sprintNames[t.SprintID]
	}
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
		sprintName,
		strconv.FormatBool(t.Done),
		encodeChecklist(t.Subtasks),
		encodeChecklist(t.Acceptance),
		formatMillis(t.CreatedAt),
		formatMillis(t.UpdatedAt),
		t.Milestone,
		t.IssueType,
		t.ParentKey,
		strings.Join(t.Reviewers, ";"),
		strings.Join(t.LinkedPRs, ";"),
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
	// Sprint membership is resolved in Apply (name → id, create-if-missing).
	// Keep a raw sprintId only when the name column is absent so older CSVs
	// still round-trip.
	if _, hasName := col["sprint"]; !hasName {
		t.SprintID = cell(row, col, "sprintId")
	}

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

	if _, ok := col["subtasks"]; ok {
		t.Subtasks = parseChecklist(cell(row, col, "subtasks"), t.Subtasks)
	}
	if _, ok := col["acceptance"]; ok {
		t.Acceptance = parseChecklist(cell(row, col, "acceptance"), t.Acceptance)
	}
	if _, ok := col["createdAt"]; ok {
		if ms, ok := parseMillis(cell(row, col, "createdAt")); ok && t.CreatedAt == 0 {
			t.CreatedAt = ms
		}
	}
	if _, ok := col["updatedAt"]; ok {
		if ms, ok := parseMillis(cell(row, col, "updatedAt")); ok {
			t.UpdatedAt = ms
		}
	} else {
		t.UpdatedAt = now
	}
	if _, ok := col["milestone"]; ok {
		t.Milestone = cell(row, col, "milestone")
	}
	if _, ok := col["issueType"]; ok {
		t.IssueType = cell(row, col, "issueType")
	}
	if _, ok := col["parentKey"]; ok {
		t.ParentKey = cell(row, col, "parentKey")
	}
	if _, ok := col["reviewers"]; ok {
		t.Reviewers = splitLabels(cell(row, col, "reviewers"))
	}
	if _, ok := col["linkedPrs"]; ok {
		t.LinkedPRs = splitLabels(cell(row, col, "linkedPrs"))
	}

	if t.UpdatedAt == 0 {
		t.UpdatedAt = now
	}
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
		case "sprint_id", "sprintid":
			key = "sprintid"
		case "sprint_name", "sprintname":
			key = "sprint"
		case "story_points", "points":
			key = "storypoints"
		case "created", "created_at", "createdat":
			key = "createdat"
		case "updated", "updated_at", "updatedat":
			key = "updatedat"
		case "issue_type", "issuetype":
			key = "issuetype"
		case "parent_key", "parentkey":
			key = "parentkey"
		case "linked_prs", "linkedprs", "prs":
			key = "linkedprs"
		case "acceptance_criteria", "ac":
			key = "acceptance"
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
		case "createdat":
			out["createdAt"] = i
		case "updatedat":
			out["updatedAt"] = i
		case "issuetype":
			out["issueType"] = i
		case "parentkey":
			out["parentKey"] = i
		case "linkedprs":
			out["linkedPrs"] = i
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

// encodeChecklist writes checklist items as "[x] Title; [ ] Title" so the
// cells stay editable in a spreadsheet without JSON escaping.
func encodeChecklist(items []model.Subtask) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, s := range items {
		title := strings.TrimSpace(s.Title)
		if title == "" {
			continue
		}
		if s.Done {
			parts = append(parts, "[x] "+title)
		} else {
			parts = append(parts, "[ ] "+title)
		}
	}
	return strings.Join(parts, "; ")
}

// parseChecklist reads encodeChecklist output (or a plain semicolon list).
// When the cell is present but empty, the checklist is cleared. Existing
// IDs are reused when the title still matches so round-trips stay stable.
func parseChecklist(s string, existing []model.Subtask) []model.Subtask {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	byTitle := make(map[string]string, len(existing))
	for _, e := range existing {
		if e.Title != "" && e.ID != "" {
			byTitle[e.Title] = e.ID
		}
	}
	parts := strings.Split(s, ";")
	out := make([]model.Subtask, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		done := false
		title := p
		lower := strings.ToLower(p)
		switch {
		case strings.HasPrefix(lower, "[x]"):
			done = true
			title = strings.TrimSpace(p[3:])
		case strings.HasPrefix(lower, "[ ]"):
			title = strings.TrimSpace(p[3:])
		case strings.HasPrefix(lower, "[]"):
			title = strings.TrimSpace(p[2:])
		}
		if title == "" {
			continue
		}
		item := model.Subtask{Title: title, Done: done}
		if id := byTitle[title]; id != "" {
			item.ID = id
		}
		out = append(out, item)
	}
	return out
}

func ensureChecklistIDs(t model.Task, newID func() string) model.Task {
	if newID == nil {
		return t
	}
	for i := range t.Subtasks {
		if t.Subtasks[i].ID == "" {
			t.Subtasks[i].ID = newID()
		}
	}
	for i := range t.Acceptance {
		if t.Acceptance[i].ID == "" {
			t.Acceptance[i].ID = newID()
		}
	}
	return t
}

func formatMillis(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return strconv.FormatInt(ms, 10)
}

func parseMillis(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "done":
		return true
	default:
		return false
	}
}
