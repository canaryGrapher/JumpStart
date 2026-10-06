package ghsync

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"devdeck/internal/model"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// DefaultBoardColumns is the built-in five-column layout used when a
// project has never customized its board.
func DefaultBoardColumns() []model.BoardColumn {
	out := make([]model.BoardColumn, len(localColumns))
	labels := map[string]string{
		"backlog":    "Backlog",
		"todo":       "To Do",
		"inprogress": "In Progress",
		"testing":    "Testing",
		"done":       "Done",
	}
	for i, id := range localColumns {
		out[i] = model.BoardColumn{ID: id, Label: labels[id], Order: i}
	}
	return out
}

// EffectiveColumns returns the project's Kanban columns, falling back to
// the built-in set when Columns is empty (older projects).
func EffectiveColumns(p *model.Project) []model.BoardColumn {
	if p != nil && len(p.Columns) > 0 {
		out := append([]model.BoardColumn(nil), p.Columns...)
		for i := range out {
			out[i].Order = i
		}
		return out
	}
	return DefaultBoardColumns()
}

// EffectiveColumnIDs is the id list EffectiveColumns would return.
func EffectiveColumnIDs(p *model.Project) []string {
	cols := EffectiveColumns(p)
	ids := make([]string, len(cols))
	for i, c := range cols {
		ids[i] = c.ID
	}
	return ids
}

// KnownColumn reports whether id is a valid column on the project
// (custom layout or the built-in defaults).
func KnownColumn(p *model.Project, id string) bool {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return false
	}
	for _, c := range EffectiveColumnIDs(p) {
		if c == id {
			return true
		}
	}
	return false
}

// NormalizeColumns assigns Order from slice position and drops empty ids.
func NormalizeColumns(cols []model.BoardColumn) ([]model.BoardColumn, error) {
	if len(cols) == 0 {
		return nil, fmt.Errorf("at least one column is required")
	}
	seen := map[string]bool{}
	out := make([]model.BoardColumn, 0, len(cols))
	for i, c := range cols {
		id := strings.TrimSpace(strings.ToLower(c.ID))
		label := strings.TrimSpace(c.Label)
		if id == "" {
			return nil, fmt.Errorf("column %d is missing an id", i+1)
		}
		if label == "" {
			return nil, fmt.Errorf("column %q needs a label", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate column id %q", id)
		}
		seen[id] = true
		out = append(out, model.BoardColumn{ID: id, Label: label, Order: len(out)})
	}
	return out, nil
}

// ColumnIDFromLabel turns a display name into a stable column id, avoiding
// collisions with existing ids by appending a numeric suffix.
func ColumnIDFromLabel(label string, existing []string) string {
	base := slugify(label)
	if base == "" {
		base = "column"
	}
	used := map[string]bool{}
	for _, id := range existing {
		used[id] = true
	}
	if !used[base] {
		return base
	}
	for n := 2; n < 1000; n++ {
		cand := fmt.Sprintf("%s%d", base, n)
		if !used[cand] {
			return cand
		}
	}
	return fmt.Sprintf("%s%d", base, len(existing)+1)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			b.WriteByte('-')
		}
	}
	out := nonSlug.ReplaceAllString(b.String(), "")
	// Prefer the compact style of built-ins (inprogress, not in-progress).
	out = strings.ReplaceAll(out, "-", "")
	return out
}
