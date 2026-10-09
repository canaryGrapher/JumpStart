package search

import (
	"strings"
	"unicode"

	"devdeck/internal/daterange"
)

// Query is a parsed search: free-text terms plus structured qualifiers.
//
//	login bug status:todo @sam #stripe project:web due:this week has:file is:overdue
type Query struct {
	Terms      []string // lowercased words that must all appear
	Due        *daterange.Range
	Created    *daterange.Range
	Updated    *daterange.Range
	NoDue      bool
	Statuses   []string
	Priorities []string
	Types      []string
	Assignees  []string
	Labels     []string
	Projects   []string
	Has        []string // file, link, criteria, subtasks, due, assignee
	No         []string
	Overdue    bool
	Done       *bool // nil = either
	// Notes explain how the query was read, e.g. `due Oct 5 – Oct 11, 2026`.
	Notes []string
	// Errors are qualifiers that could not be understood.
	Errors []string
}

// HasFilters reports whether anything other than free text was given.
func (q Query) HasFilters() bool {
	return q.Due != nil || q.Created != nil || q.Updated != nil || q.NoDue || q.Overdue || q.Done != nil ||
		len(q.Statuses)+len(q.Priorities)+len(q.Types)+len(q.Assignees)+len(q.Labels)+len(q.Has)+len(q.No) > 0
}

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "of": true, "to": true,
	"in": true, "on": true, "for": true, "with": true, "at": true, "by": true, "is": true,
	"due": true, "task": true, "tasks": true,
}

// "due" and "task(s)" are dropped as free text so "tasks due friday" works;
// they still match through the due date itself.

func boolPtr(b bool) *bool { return &b }

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(strings.ToLower(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Parse reads a search string. Unknown qualifiers are reported in Errors
// and otherwise ignored, rather than being searched as text.
func Parse(input string, c DateContext) Query {
	var q Query
	words := strings.Fields(input)
	var free []string
	for i := 0; i < len(words); i++ {
		w := words[i]
		lw := strings.ToLower(w)
		switch {
		case strings.HasPrefix(w, "@") && len(w) > 1:
			q.Assignees = append(q.Assignees, strings.ToLower(w[1:]))
			continue
		case strings.HasPrefix(w, "#") && len(w) > 1:
			q.Labels = append(q.Labels, strings.ToLower(w[1:]))
			continue
		}
		key, val, ok := strings.Cut(lw, ":")
		if !ok || key == "" || strings.Contains(key, "/") || strings.HasPrefix(val, "//") {
			free = append(free, w)
			continue
		}
		// Quoted or multi-word values: due:"next week" or due:next week.
		// Also status:todo,"in progress".
		for strings.Count(val, "\"")%2 == 1 && i+1 < len(words) {
			i++
			val += " " + strings.ToLower(words[i])
		}
		val = strings.ReplaceAll(val, "\"", "")
		switch key {
		case "due", "created", "updated":
			if key == "due" && (val == "none" || val == "never") {
				q.NoDue = true
				q.Notes = append(q.Notes, "no due date")
				continue
			}
			if key == "due" && (val == "overdue" || val == "past") {
				q.Overdue = true
				q.Notes = append(q.Notes, "past due")
				continue
			}
			// Take the longest following phrase that parses: due:this week.
			var r daterange.Range
			found := false
			used := 0
			for n := minInt(3, len(words)-i-1); n >= 0 && !found; n-- {
				phrase := strings.Join(append([]string{val}, words[i+1:i+1+n]...), " ")
				if rr, ok := ParseDate(phrase, c, false); ok {
					r, found, used = rr, true, n
				}
			}
			if !found {
				q.Errors = append(q.Errors, w+": not a date")
				continue
			}
			i += used
			rr := r
			switch key {
			case "due":
				q.Due = &rr
			case "created":
				q.Created = &rr
			default:
				q.Updated = &rr
			}
			q.Notes = append(q.Notes, key+" "+Describe(r))
		case "status", "s":
			q.Statuses = append(q.Statuses, splitList(val)...)
		case "priority", "p":
			q.Priorities = append(q.Priorities, splitList(val)...)
		case "type":
			q.Types = append(q.Types, splitList(val)...)
		case "assignee", "owner":
			q.Assignees = append(q.Assignees, splitList(val)...)
		case "label", "tag":
			q.Labels = append(q.Labels, splitList(val)...)
		case "project", "in":
			q.Projects = append(q.Projects, splitList(val)...)
		case "has":
			q.Has = append(q.Has, normalizeHas(splitList(val))...)
		case "no":
			for _, v := range normalizeHas(splitList(val)) {
				if v == "due" {
					q.NoDue = true
				} else {
					q.No = append(q.No, v)
				}
			}
		case "is":
			switch val {
			case "overdue", "late", "pastdue":
				q.Overdue = true
				q.Notes = append(q.Notes, "past due")
			case "done", "closed", "complete", "completed":
				q.Done = boolPtr(true)
			case "open", "active", "todo":
				q.Done = boolPtr(false)
			case "story", "bug", "task":
				q.Types = append(q.Types, val)
			default:
				q.Errors = append(q.Errors, w+": unknown")
			}
		default:
			// Something like "re:" or "TODO:" is ordinary text.
			free = append(free, w)
		}
	}

	ranges, phrases, rest := FindDates(free, c)
	if len(ranges) > 0 && q.Due == nil {
		r := ranges[0]
		for _, x := range ranges[1:] { // "oct 1 oct 15" spans both
			if x.From < r.From {
				r.From = x.From
			}
			if x.To > r.To {
				r.To = x.To
			}
		}
		q.Due = &r
		q.Notes = append(q.Notes, "due "+Describe(r)+" (from “"+strings.Join(phrases, "”, “")+"”)")
	} else {
		rest = free
	}
	for _, w := range rest {
		for _, t := range tokenize(w) {
			if !stopwords[t] {
				q.Terms = append(q.Terms, t)
			}
		}
	}
	q.Terms = dedupe(q.Terms)
	return q
}

func normalizeHas(vals []string) []string {
	alias := map[string]string{
		"file": "file", "files": "file", "attachment": "file", "attachments": "file", "image": "image", "images": "image",
		"link": "link", "links": "link", "url": "link",
		"criteria": "criteria", "acceptance": "criteria", "ac": "criteria",
		"subtasks": "subtasks", "subtask": "subtasks", "checklist": "subtasks",
		"due": "due", "date": "due", "duedate": "due", "assignee": "assignee", "owner": "assignee",
	}
	var out []string
	for _, v := range vals {
		if a, ok := alias[v]; ok {
			out = append(out, a)
		}
	}
	return out
}

// tokenize lowercases and splits on anything that is not a letter or digit.
// Single characters are dropped except digits ("v2" stays "v2").
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if len([]rune(f)) >= 2 || unicode.IsDigit([]rune(f)[0]) {
			if len(f) <= 60 {
				out = append(out, f)
			}
		}
	}
	return out
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
