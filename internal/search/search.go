// Package search is JumpStart's global search: projects, tasks (every
// field) and attachment contents, including text read from images by OCR.
//
// Ranking is BM25 over weighted fields with prefix matching. Every word
// must match somewhere in a result (AND), so a stray common word does not
// flood the list. Dates in the query ("this week", "Oct 9", "10/9/2026")
// become a due-date filter rather than text, and qualifiers such as
// status:, @assignee, #label and has:file narrow the results.
package search

import (
	"math"
	"sort"
	"strings"
	"time"

	"devdeck/internal/daterange"
	"devdeck/internal/ghsync"
	"devdeck/internal/model"
)

// Text states for attachment contents.
const (
	TextReady   = "ready"   // text was extracted (or the file is text)
	TextPending = "pending" // OCR/extraction has not run yet
	TextNone    = "none"    // nothing searchable (OCR off, unsupported type)
)

// TextSource returns the searchable text of an attachment and its state.
type TextSource func(projectID, taskID string, a model.Attachment) (text, state string)

// Options control one search.
type Options struct {
	ProjectID      string // limit to one project; empty searches everything
	HideDone       bool
	Limit          int // default 50
	Today          time.Time
	DateOrder      string
	GlobalQuarters []model.QuarterRange
	Text           TextSource
}

// Result is one hit.
type Result struct {
	Kind         string  `json:"kind"` // project | task | file
	ProjectID    string  `json:"projectId"`
	ProjectName  string  `json:"projectName"`
	TaskID       string  `json:"taskId,omitempty"`
	Title        string  `json:"title"`
	Type         string  `json:"type,omitempty"`
	Status       string  `json:"status,omitempty"`
	StatusLabel  string  `json:"statusLabel,omitempty"`
	Priority     string  `json:"priority,omitempty"`
	DueDate      string  `json:"dueDate,omitempty"`
	Done         bool    `json:"done,omitempty"`
	PastDue      bool    `json:"pastDue,omitempty"`
	AttachmentID string  `json:"attachmentId,omitempty"`
	FileName     string  `json:"fileName,omitempty"`
	TaskTitle    string  `json:"taskTitle,omitempty"` // for files: the task they belong to
	Field        string  `json:"field,omitempty"`     // where the words matched
	Snippet      string  `json:"snippet,omitempty"`
	Score        float64 `json:"score"`
}

// Response is the result list plus how the query was understood.
type Response struct {
	Results []Result `json:"results"`
	Total   int      `json:"total"`
	Terms   []string `json:"terms"`  // words that were searched, for highlighting
	Notes   []string `json:"notes"`  // e.g. "due Oct 5 – Oct 11, 2026"
	Errors  []string `json:"errors"` // qualifiers that were not understood
	// Pending counts images and PDFs whose text is still being read.
	Pending int `json:"pending"`
}

type field struct {
	name   string
	text   string
	weight float64
	tokens []string
}

type doc struct {
	res    Result
	fields []field
	length int
	task   *model.Task
}

func newField(name, text string, weight float64) field {
	return field{name: name, text: text, weight: weight, tokens: tokenize(text)}
}

func (d *doc) add(f field) {
	if strings.TrimSpace(f.text) == "" {
		return
	}
	d.fields = append(d.fields, f)
	d.length += len(f.tokens)
}

func checklistText(items []model.Subtask) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Title)
		b.WriteString("\n")
	}
	return b.String()
}

func customFieldText(t model.Task) string {
	var b strings.Builder
	for _, f := range t.Fields {
		for _, v := range []string{f.Display, f.Text, f.OptionKey, f.Date, strings.Join(f.Users, " "), strings.Join(f.Labels, " ")} {
			if v != "" {
				b.WriteString(v)
				b.WriteString(" ")
			}
		}
	}
	return b.String()
}

// Search runs input against projects.
func Search(projects []model.Project, input string, o Options) Response {
	if o.Limit <= 0 {
		o.Limit = 50
	}
	if o.Today.IsZero() {
		o.Today = time.Now()
	}
	today := day(o.Today)
	q := Parse(input, DateContext{Today: today, Order: o.DateOrder, Quarters: o.GlobalQuarters})
	resp := Response{Terms: q.Terms, Notes: q.Notes, Errors: q.Errors, Results: []Result{}}
	if len(q.Terms) == 0 && !q.HasFilters() && len(q.Projects) == 0 {
		return resp
	}

	var docs []*doc
	for pi := range projects {
		p := &projects[pi]
		if o.ProjectID != "" && p.ID != o.ProjectID {
			continue
		}
		if !projectMatches(q.Projects, p) {
			continue
		}
		cols := ghsync.EffectiveColumns(p)
		label := func(id string) string {
			for _, c := range cols {
				if c.ID == id {
					return c.Label
				}
			}
			return id
		}
		// A project only matches its own name/description, and only for a
		// plain text search: filters are about tasks.
		if len(q.Terms) > 0 && !q.HasFilters() {
			d := &doc{res: Result{Kind: "project", ProjectID: p.ID, ProjectName: p.Name, Title: p.Name}}
			d.add(newField("name", p.Name, 3))
			d.add(newField("description", p.Description, 1))
			docs = append(docs, d)
		}
		for ti := range p.Tasks {
			t := &p.Tasks[ti]
			done := t.Done || t.Status == "done"
			if o.HideDone && done {
				continue
			}
			if !taskPasses(q, p, t, label, today) {
				continue
			}
			base := Result{
				ProjectID: p.ID, ProjectName: p.Name, TaskID: t.ID, Type: firstNonEmpty(t.Type, "task"),
				Status: t.Status, StatusLabel: label(t.Status), Priority: t.Priority, DueDate: t.DueDate,
				Done: done, PastDue: daterange.IsPastDue(t.DueDate, done, today),
			}
			td := &doc{res: base, task: t}
			td.res.Kind, td.res.Title = "task", t.Title
			td.add(newField("title", t.Title, 3))
			td.add(newField("description", t.Description, 1))
			td.add(newField("labels", strings.Join(t.Labels, " "), 2))
			td.add(newField("assignee", t.Assignee, 1.5))
			td.add(newField("acceptance criteria", checklistText(t.Acceptance), 1))
			td.add(newField("subtasks", checklistText(t.Subtasks), 1))
			var links strings.Builder
			for _, l := range t.Links {
				links.WriteString(l.Title + " " + l.URL + "\n")
			}
			td.add(newField("links", links.String(), 1))
			td.add(newField("milestone", t.Milestone, 1))
			var names strings.Builder
			for _, a := range t.Attachments {
				names.WriteString(a.Name + "\n")
			}
			td.add(newField("file names", names.String(), 1.5))
			td.add(newField("fields", customFieldText(*t), 1))
			docs = append(docs, td)

			for _, a := range t.Attachments {
				text, state := "", TextNone
				if o.Text != nil {
					text, state = o.Text(p.ID, t.ID, a)
				}
				if state == TextPending {
					resp.Pending++
				}
				if len(q.Terms) == 0 {
					continue // a filter-only search lists tasks, not their files
				}
				fd := &doc{res: base, task: t}
				fd.res.Kind, fd.res.Title = "file", a.Name
				fd.res.AttachmentID, fd.res.FileName, fd.res.TaskTitle = a.ID, a.Name, t.Title
				fd.add(newField("file name", a.Name, 2))
				fd.add(newField("file text", text, 1))
				docs = append(docs, fd)
			}
		}
	}

	var hits []Result
	if len(q.Terms) == 0 {
		for _, d := range docs {
			if d.res.Kind == "task" {
				hits = append(hits, d.res)
			}
		}
		sort.SliceStable(hits, func(i, j int) bool {
			a, b := hits[i], hits[j]
			if (a.DueDate == "") != (b.DueDate == "") {
				return a.DueDate != ""
			}
			if a.DueDate != b.DueDate {
				return a.DueDate < b.DueDate
			}
			return strings.ToLower(a.Title) < strings.ToLower(b.Title)
		})
	} else {
		hits = rank(docs, q.Terms)
	}
	resp.Total = len(hits)
	if len(hits) > o.Limit {
		hits = hits[:o.Limit]
	}
	resp.Results = append(resp.Results, hits...)
	return resp
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// termHit reports how a field matches a term: 1 for an exact word, 0.7 for
// a word that starts with it (terms of 3+ characters only), 0 otherwise.
func termHit(tokens []string, term string) float64 {
	best := 0.0
	for _, tok := range tokens {
		if tok == term {
			return 1
		}
		if len(term) >= 3 && strings.HasPrefix(tok, term) {
			best = 0.7
		}
	}
	return best
}

func termFreq(tokens []string, term string) float64 {
	n := 0.0
	for _, tok := range tokens {
		if tok == term {
			n++
		} else if len(term) >= 3 && strings.HasPrefix(tok, term) {
			n += 0.7
		}
	}
	return n
}

const (
	k1 = 1.2
	b  = 0.75
)

func rank(docs []*doc, terms []string) []Result {
	if len(docs) == 0 {
		return nil
	}
	df := make(map[string]int, len(terms))
	total := 0
	for _, d := range docs {
		total += d.length
		for _, t := range terms {
			for _, f := range d.fields {
				if termHit(f.tokens, t) > 0 {
					df[t]++
					break
				}
			}
		}
	}
	n := float64(len(docs))
	avg := math.Max(1, float64(total)/n)

	var out []Result
	for _, d := range docs {
		score := 0.0
		bestField, bestContribution := "", 0.0
		all := true
		for _, t := range terms {
			tf := 0.0
			for _, f := range d.fields {
				c := termFreq(f.tokens, t) * f.weight
				tf += c
				if c > bestContribution && f.name != "title" && f.name != "name" {
					bestField, bestContribution = f.name, c
				}
			}
			if tf == 0 {
				all = false
				break
			}
			idf := math.Log(1 + (n-float64(df[t])+0.5)/(float64(df[t])+0.5))
			score += idf * tf * (k1 + 1) / (tf + k1*(1-b+b*float64(d.length)/avg))
		}
		if !all {
			continue
		}
		r := d.res
		if r.Done {
			score *= 0.6
		}
		title := strings.ToLower(r.Title)
		if strings.Contains(title, strings.Join(terms, " ")) {
			score *= 1.5
		}
		r.Score = math.Round(score*1000) / 1000
		// Show where it matched unless the title alone explains it.
		titleHasAll := true
		for _, t := range terms {
			if termHit(tokenize(r.Title), t) == 0 {
				titleHasAll = false
				break
			}
		}
		if bestField != "" && !titleHasAll {
			for _, f := range d.fields {
				if f.name == bestField {
					r.Field = f.name
					r.Snippet = snippet(f.text, terms)
					break
				}
			}
		} else if r.Kind == "task" && d.task != nil && d.task.Description != "" {
			r.Snippet = snippet(d.task.Description, nil)
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	return out
}

// snippet returns about 140 characters of text around the first matched
// term (or the start of text), with whitespace collapsed.
func snippet(text string, terms []string) string {
	flat := []rune(strings.Join(strings.Fields(text), " "))
	lower := []rune(strings.ToLower(string(flat)))
	at := 0
	for _, t := range terms {
		if i := indexWordStart(lower, []rune(t)); i >= 0 {
			at = i
			break
		}
	}
	start := at - 50
	if start < 0 {
		start = 0
	}
	end := start + 140
	if end > len(flat) {
		end = len(flat)
	}
	s := strings.TrimSpace(string(flat[start:end]))
	if start > 0 {
		s = "…" + s
	}
	if end < len(flat) {
		s += "…"
	}
	return s
}

func isWordRune(r rune) bool {
	return r == '_' || ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') || r > 127
}

func indexWordStart(hay, needle []rune) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if i > 0 && isWordRune(hay[i-1]) {
			continue
		}
		if string(hay[i:i+len(needle)]) == string(needle) {
			return i
		}
	}
	return -1
}

func projectMatches(want []string, p *model.Project) bool {
	if len(want) == 0 {
		return true
	}
	name := strings.ToLower(p.Name)
	for _, w := range want {
		if w == strings.ToLower(p.ID) || strings.Contains(name, w) {
			return true
		}
	}
	return false
}

func squash(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), ""))
}

func taskPasses(q Query, p *model.Project, t *model.Task, label func(string) string, today time.Time) bool {
	done := t.Done || t.Status == "done"
	if q.Done != nil && *q.Done != done {
		return false
	}
	if len(q.Statuses) > 0 {
		ok := false
		for _, s := range q.Statuses {
			if squash(s) == squash(t.Status) || squash(s) == squash(label(t.Status)) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if len(q.Priorities) > 0 && !containsFold(q.Priorities, t.Priority) {
		return false
	}
	if len(q.Types) > 0 && !containsFold(q.Types, firstNonEmpty(t.Type, "task")) {
		return false
	}
	if len(q.Assignees) > 0 {
		have := strings.ToLower(t.Assignee)
		ok := false
		for _, a := range q.Assignees {
			if have != "" && strings.Contains(have, a) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	if len(q.Labels) > 0 {
		ok := false
		for _, l := range q.Labels {
			if containsFold(t.Labels, l) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	for _, h := range q.Has {
		if !has(t, h) {
			return false
		}
	}
	for _, h := range q.No {
		if has(t, h) {
			return false
		}
	}
	if q.NoDue && t.DueDate != "" {
		return false
	}
	if q.Overdue && !daterange.IsPastDue(t.DueDate, done, today) {
		return false
	}
	if q.Due != nil && !q.Due.Contains(t.DueDate) {
		return false
	}
	if q.Created != nil && !q.Created.Contains(msDate(t.CreatedAt)) {
		return false
	}
	if q.Updated != nil && !q.Updated.Contains(msDate(firstNonZero(t.UpdatedAt, t.CreatedAt))) {
		return false
	}
	return true
}

func firstNonZero(a, b int64) int64 {
	if a != 0 {
		return a
	}
	return b
}

func msDate(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).In(time.Local).Format(daterange.DateLayout)
}

func has(t *model.Task, what string) bool {
	switch what {
	case "file":
		return len(t.Attachments) > 0
	case "image":
		for _, a := range t.Attachments {
			if strings.HasPrefix(a.Mime, "image/") {
				return true
			}
		}
		return false
	case "link":
		return len(t.Links) > 0
	case "criteria":
		return len(t.Acceptance) > 0
	case "subtasks":
		return len(t.Subtasks) > 0
	case "due":
		return t.DueDate != ""
	case "assignee":
		return strings.TrimSpace(t.Assignee) != ""
	}
	return false
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}
