package ghsync

import (
	"fmt"
	"strings"

	"devdeck/internal/model"
)

// Stable HTML-comment markers wrap the checklist sections JumpStart
// owns inside an issue/draft body. Plain "## Acceptance criteria"
// headings alone are too ambiguous — a user may write the same words
// by hand — so the markers are what make a round trip safe.
const (
	acceptanceStart = "<!-- jumpstart:acceptance -->"
	acceptanceEnd   = "<!-- /jumpstart:acceptance -->"
	subtasksStart   = "<!-- jumpstart:subtasks -->"
	subtasksEnd     = "<!-- /jumpstart:subtasks -->"
)

// ComposeBody builds the GitHub issue/draft body from the local
// description plus any acceptance criteria and subtasks. Empty
// checklists omit their section entirely.
func ComposeBody(description string, acceptance, subtasks []model.Subtask) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(description, "\n"))

	if section := checklistSection(acceptanceStart, acceptanceEnd, "Acceptance criteria", acceptance); section != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(section)
	}
	if section := checklistSection(subtasksStart, subtasksEnd, "Subtasks", subtasks); section != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(section)
	}
	return b.String()
}

// ParseBody splits a GitHub body into the free-form description and any
// JumpStart-owned checklists. Sections without markers are left in the
// description so hand-written GitHub markdown is never eaten.
func ParseBody(body string) (description string, acceptance, subtasks []model.Subtask) {
	rest := body
	var accBlock, subBlock string
	rest, accBlock = cutMarkedSection(rest, acceptanceStart, acceptanceEnd)
	rest, subBlock = cutMarkedSection(rest, subtasksStart, subtasksEnd)
	description = strings.TrimSpace(rest)
	acceptance = parseChecklistBlock(accBlock)
	subtasks = parseChecklistBlock(subBlock)
	return description, acceptance, subtasks
}

func checklistSection(start, end, heading string, items []model.Subtask) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(start)
	b.WriteByte('\n')
	fmt.Fprintf(&b, "## %s\n", heading)
	for _, it := range items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			continue
		}
		mark := " "
		if it.Done {
			mark = "x"
		}
		fmt.Fprintf(&b, "- [%s] %s\n", mark, title)
	}
	b.WriteString(end)
	return b.String()
}

// cutMarkedSection removes one marked block from s and returns the
// remainder plus the inner text (without the markers). Missing markers
// leave s unchanged and return an empty block.
func cutMarkedSection(s, start, end string) (rest, block string) {
	i := strings.Index(s, start)
	if i < 0 {
		return s, ""
	}
	j := strings.Index(s[i+len(start):], end)
	if j < 0 {
		return s, ""
	}
	j = i + len(start) + j
	block = strings.TrimSpace(s[i+len(start) : j])
	rest = strings.TrimSpace(s[:i] + s[j+len(end):])
	return rest, block
}

func parseChecklistBlock(block string) []model.Subtask {
	if block == "" {
		return nil
	}
	var out []model.Subtask
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		done, title, ok := parseChecklistLine(line)
		if !ok || title == "" {
			continue
		}
		out = append(out, model.Subtask{Title: title, Done: done})
	}
	return out
}

func parseChecklistLine(line string) (done bool, title string, ok bool) {
	// GitHub task-list forms: "- [ ] title", "* [x] title", "1. [X] title"
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
		line = strings.TrimSpace(line[1:])
	} else if i := strings.IndexByte(line, '.'); i > 0 && isDigits(line[:i]) {
		line = strings.TrimSpace(line[i+1:])
	} else {
		return false, "", false
	}
	if len(line) < 3 || line[0] != '[' || line[2] != ']' {
		return false, "", false
	}
	mark := line[1]
	done = mark == 'x' || mark == 'X'
	title = strings.TrimSpace(line[3:])
	return done, title, true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// sameChecklist reports whether two checklists match by title and done
// state, ignoring ids (which only exist locally).
func sameChecklist(a, b []model.Subtask) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Done != b[i].Done || strings.TrimSpace(a[i].Title) != strings.TrimSpace(b[i].Title) {
			return false
		}
	}
	return true
}

// adoptChecklist copies remote checklist items onto the local list,
// reusing an existing id when the title matches so the UI does not
// flicker keys on every pull.
func adoptChecklist(local, remote []model.Subtask, newID func() string) []model.Subtask {
	if len(remote) == 0 {
		if len(local) == 0 {
			return local
		}
		return nil
	}
	byTitle := map[string]string{}
	for _, it := range local {
		key := strings.ToLower(strings.TrimSpace(it.Title))
		if key != "" && it.ID != "" {
			byTitle[key] = it.ID
		}
	}
	out := make([]model.Subtask, 0, len(remote))
	for _, it := range remote {
		title := strings.TrimSpace(it.Title)
		id := byTitle[strings.ToLower(title)]
		if id == "" {
			id = newID()
		}
		out = append(out, model.Subtask{ID: id, Title: title, Done: it.Done})
	}
	return out
}
