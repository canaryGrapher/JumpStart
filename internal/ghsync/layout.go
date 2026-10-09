package ghsync

import (
	"fmt"
	"strings"

	"devdeck/internal/model"
)

// ApplyBoardLayout replaces a project's Kanban columns in one step:
// new order and labels, columns added (blank id = derive one from the
// label), and columns removed. Tasks in a removed column move to
// moves[removedID]; a removed column that still holds tasks must have a
// destination. GitHub status mappings for removed columns are dropped.
// It returns the number of tasks moved.
func ApplyBoardLayout(p *model.Project, cols []model.BoardColumn, moves map[string]string) (int, error) {
	existing := make([]string, 0, len(cols))
	for _, c := range cols {
		if strings.TrimSpace(c.ID) != "" {
			existing = append(existing, strings.ToLower(strings.TrimSpace(c.ID)))
		}
	}
	for i := range cols {
		if strings.TrimSpace(cols[i].ID) == "" {
			cols[i].ID = ColumnIDFromLabel(cols[i].Label, existing)
			existing = append(existing, cols[i].ID)
		}
	}
	normalized, err := NormalizeColumns(cols)
	if err != nil {
		return 0, err
	}
	keep := map[string]bool{}
	for _, c := range normalized {
		keep[c.ID] = true
	}

	before := EffectiveColumns(p)
	removed := map[string]bool{}
	for _, c := range before {
		if !keep[c.ID] {
			removed[c.ID] = true
		}
	}

	// Validate every move before touching anything.
	counts := map[string]int{}
	for _, t := range p.Tasks {
		if removed[t.Status] {
			counts[t.Status]++
		}
	}
	for id, n := range counts {
		dest := strings.ToLower(strings.TrimSpace(moves[id]))
		if dest == "" {
			return 0, fmt.Errorf("column %q still has %d task(s); choose where to move them", id, n)
		}
		if !keep[dest] {
			return 0, fmt.Errorf("cannot move tasks from %q to %q: that column is not on the board", id, dest)
		}
	}

	moved := 0
	for i := range p.Tasks {
		if !removed[p.Tasks[i].Status] {
			continue
		}
		dest := strings.ToLower(strings.TrimSpace(moves[p.Tasks[i].Status]))
		p.Tasks[i].Status = dest
		p.Tasks[i].Done = dest == "done"
		moved++
	}
	if p.GitHub != nil && p.GitHub.StatusMap != nil {
		for id := range removed {
			delete(p.GitHub.StatusMap, id)
		}
	}
	p.Columns = normalized
	return moved, nil
}
