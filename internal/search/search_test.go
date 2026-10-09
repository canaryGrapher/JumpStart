package search

import (
	"strings"
	"testing"
	"time"

	"devdeck/internal/model"
)

// Friday 9 October 2026.
var today = time.Date(2026, 10, 9, 15, 4, 0, 0, time.Local)

func ctx(order string) DateContext { return DateContext{Today: today, Order: order} }

func TestParseDateFormats(t *testing.T) {
	cases := []struct {
		in, order string
		strict    bool
		from, to  string
	}{
		{"2026-10-09", "mdy", true, "2026-10-09", "2026-10-09"},
		{"2026/10/9", "mdy", true, "2026-10-09", "2026-10-09"},
		{"20261009", "mdy", true, "2026-10-09", "2026-10-09"},
		{"10/9/2026", "mdy", true, "2026-10-09", "2026-10-09"},
		{"10/9/26", "mdy", true, "2026-10-09", "2026-10-09"},
		{"10/9", "mdy", true, "2026-10-09", "2026-10-09"},
		{"10/9", "dmy", true, "2026-09-10", "2026-09-10"},
		{"9.10.2026", "dmy", true, "2026-10-09", "2026-10-09"},
		{"25/12", "mdy", true, "2026-12-25", "2026-12-25"}, // impossible month flips the order
		{"Oct 9", "mdy", true, "2026-10-09", "2026-10-09"},
		{"oct 9th", "mdy", true, "2026-10-09", "2026-10-09"},
		{"October 9, 2026", "mdy", true, "2026-10-09", "2026-10-09"},
		{"9 Oct", "mdy", true, "2026-10-09", "2026-10-09"},
		{"9th of October 2026", "mdy", true, "2026-10-09", "2026-10-09"},
		{"Sept 30", "mdy", true, "2026-09-30", "2026-09-30"},
		{"Oct 2026", "mdy", true, "2026-10-01", "2026-10-31"},
		{"february", "mdy", false, "2026-02-01", "2026-02-28"},
		{"today", "mdy", true, "2026-10-09", "2026-10-09"},
		{"tomorrow", "mdy", true, "2026-10-10", "2026-10-10"},
		{"yesterday", "mdy", true, "2026-10-08", "2026-10-08"},
		{"this week", "mdy", true, "2026-10-05", "2026-10-11"},
		{"next week", "mdy", true, "2026-10-12", "2026-10-18"},
		{"last week", "mdy", true, "2026-09-28", "2026-10-04"},
		{"this month", "mdy", true, "2026-10-01", "2026-10-31"},
		{"next month", "mdy", true, "2026-11-01", "2026-11-30"},
		{"last month", "mdy", true, "2026-09-01", "2026-09-30"},
		{"this quarter", "mdy", true, "2026-10-01", "2026-12-31"},
		{"next quarter", "mdy", true, "2027-01-01", "2027-03-31"},
		{"last quarter", "mdy", true, "2026-07-01", "2026-09-30"},
		{"this year", "mdy", true, "2026-01-01", "2026-12-31"},
		{"next year", "mdy", true, "2027-01-01", "2027-12-31"},
		{"q2", "mdy", false, "2026-04-01", "2026-06-30"},
		{"in 3 days", "mdy", true, "2026-10-12", "2026-10-12"},
		{"in 2 weeks", "mdy", true, "2026-10-23", "2026-10-23"},
		{"next 7 days", "mdy", true, "2026-10-09", "2026-10-16"},
		{"3 days ago", "mdy", true, "2026-10-06", "2026-10-06"},
		{"this friday", "mdy", true, "2026-10-09", "2026-10-09"},
		{"next friday", "mdy", true, "2026-10-16", "2026-10-16"},
		{"last friday", "mdy", true, "2026-10-02", "2026-10-02"},
		{"next monday", "mdy", true, "2026-10-12", "2026-10-12"},
		{"friday", "mdy", false, "2026-10-09", "2026-10-09"},
		{"mon", "mdy", false, "2026-10-12", "2026-10-12"},
	}
	for _, c := range cases {
		r, ok := ParseDate(c.in, ctx(c.order), c.strict)
		if !ok || r.From != c.from || r.To != c.to {
			t.Errorf("ParseDate(%q, %s, strict=%v) = %v %v, want %s..%s", c.in, c.order, c.strict, r, ok, c.from, c.to)
		}
	}
}

func TestParseDateRejectsNonDates(t *testing.T) {
	for _, in := range []string{
		"may", "march", "friday", "q3", "2026", "9", "1-2", "v1.2", "13/13", "2/30/2026", "20261399",
		"oct 9 2026 deploy", "login", "", "3.14", "404",
	} {
		if r, ok := ParseDate(in, ctx("mdy"), true); ok {
			t.Errorf("strict ParseDate(%q) = %v, want no date", in, r)
		}
	}
}

func TestParseQualifiersAndDates(t *testing.T) {
	q := Parse(`webhook status:todo,"in progress" @sam #stripe due:next week has:file is:open project:web`, ctx("mdy"))
	if strings.Join(q.Terms, ",") != "webhook" {
		t.Errorf("terms = %v", q.Terms)
	}
	if q.Due == nil || q.Due.From != "2026-10-12" || q.Due.To != "2026-10-18" {
		t.Errorf("due = %v", q.Due)
	}
	if len(q.Assignees) != 1 || q.Assignees[0] != "sam" || q.Labels[0] != "stripe" || q.Has[0] != "file" || *q.Done || q.Projects[0] != "web" {
		t.Errorf("qualifiers = %+v", q)
	}

	q = Parse("deploy notes Oct 9", ctx("mdy"))
	if strings.Join(q.Terms, ",") != "deploy,notes" || q.Due == nil || q.Due.From != "2026-10-09" {
		t.Errorf("free-text date: %+v", q)
	}
	q = Parse("tasks due friday", ctx("mdy"))
	if len(q.Terms) != 0 || q.Due == nil || q.Due.From != "2026-10-09" {
		t.Errorf("due friday: %+v", q)
	}
	q = Parse("friday release", ctx("mdy")) // bare weekday is text
	if strings.Join(q.Terms, ",") != "friday,release" || q.Due != nil {
		t.Errorf("bare weekday: %+v", q)
	}
	q = Parse("re: TODO: https://x.example/a:b", ctx("mdy"))
	if q.HasFilters() || len(q.Errors) != 0 {
		t.Errorf("colons in text should stay text: %+v", q)
	}
	q = Parse("due:someday is:weird", ctx("mdy"))
	if len(q.Errors) != 2 {
		t.Errorf("errors = %v", q.Errors)
	}
}

func fixture() []model.Project {
	return []model.Project{
		{
			ID: "web", Name: "Web Store", Description: "Checkout and payments",
			Tasks: []model.Task{
				{ID: "t1", Title: "Fix login bug", Type: "bug", Status: "todo", Priority: "high", Assignee: "Sam", DueDate: "2026-10-09", Labels: []string{"auth"},
					Description: "Users see a 401 after the token rotation on Friday."},
				{ID: "t2", Title: "Payment retries", Status: "inprogress", DueDate: "2026-10-14",
					Acceptance: []model.Subtask{{Title: "Retries the same charge without duplicates"}},
					Links:      []model.TaskLink{{Title: "Stripe charge docs", URL: "https://docs.stripe.com/api/charges"}}},
				{ID: "t3", Title: "Old login cleanup", Status: "done", Done: true, DueDate: "2026-09-01"},
				{ID: "t4", Title: "Release notes", Status: "todo", Attachments: []model.Attachment{
					{ID: "a1", Name: "whiteboard.png", Mime: "image/png"},
					{ID: "a2", Name: "plan.txt", Mime: "text/plain"},
				}},
			},
		},
		{ID: "ops", Name: "Ops", Description: "Login infrastructure", Tasks: []model.Task{
			{ID: "o1", Title: "Rotate certificates", Status: "todo", DueDate: "2026-08-01"},
		}},
	}
}

func text(_, _ string, a model.Attachment) (string, string) {
	switch a.ID {
	case "a1":
		return "Q4 roadmap: migrate billing to the new ledger", TextReady
	case "a2":
		return "", TextPending
	}
	return "", TextNone
}

func run(q string, o ...func(*Options)) Response {
	opts := Options{Today: today, DateOrder: "mdy", Text: text}
	for _, f := range o {
		f(&opts)
	}
	return Search(fixture(), q, opts)
}

func titles(r Response) string {
	var out []string
	for _, x := range r.Results {
		out = append(out, x.Kind+":"+x.Title)
	}
	return strings.Join(out, " | ")
}

func TestSearchRanksAndRequiresAllWords(t *testing.T) {
	r := run("login")
	got := titles(r)
	// Exact title words first, the done task ranked below, the project too.
	if !strings.HasPrefix(got, "task:Fix login bug") || !strings.Contains(got, "task:Old login cleanup") || !strings.Contains(got, "project:Ops") {
		t.Fatalf("login: %s", got)
	}
	if strings.Index(got, "Old login cleanup") < strings.Index(got, "Fix login bug") {
		t.Errorf("done task should rank lower: %s", got)
	}
	if got := titles(run("login payments")); got != "" {
		t.Errorf("AND: no single result has both words, got %s", got)
	}
	if got := titles(run("log")); !strings.Contains(got, "Fix login bug") {
		t.Errorf("prefix match: %s", got)
	}
	if got := titles(run("lo")); strings.Contains(got, "login") {
		t.Errorf("two-letter prefix should not match: %s", got)
	}
	if got := titles(run("login", func(o *Options) { o.HideDone = true })); strings.Contains(got, "Old login") {
		t.Errorf("hide done: %s", got)
	}
}

func TestSearchFieldsAndSnippets(t *testing.T) {
	r := run("rotation")
	if len(r.Results) != 1 || r.Results[0].Field != "description" || !strings.Contains(r.Results[0].Snippet, "token rotation") {
		t.Fatalf("description: %+v", r.Results)
	}
	if got := titles(run("duplicates")); got != "task:Payment retries" {
		t.Errorf("criteria: %s", got)
	}
	if got := titles(run("stripe docs")); got != "task:Payment retries" {
		t.Errorf("links: %s", got)
	}
	r = run("ledger")
	if len(r.Results) != 1 || r.Results[0].Kind != "file" || r.Results[0].FileName != "whiteboard.png" || r.Results[0].TaskTitle != "Release notes" {
		t.Fatalf("OCR text: %+v", r.Results)
	}
	if r.Pending != 1 {
		t.Errorf("pending = %d", r.Pending)
	}
	if got := titles(run("whiteboard")); !strings.Contains(got, "task:Release notes") || !strings.Contains(got, "file:whiteboard.png") {
		t.Errorf("file name: %s", got)
	}
}

func TestSearchDatesAndFilters(t *testing.T) {
	if got := titles(run("today")); got != "task:Fix login bug" {
		t.Errorf("today: %s", got)
	}
	if got := titles(run("Oct 14")); got != "task:Payment retries" {
		t.Errorf("Oct 14: %s", got)
	}
	if got := titles(run("10/14/2026")); got != "task:Payment retries" {
		t.Errorf("10/14/2026: %s", got)
	}
	if got := titles(run("this week")); got != "task:Fix login bug" {
		t.Errorf("this week (Oct 5-11): %s", got)
	}
	if got := titles(run("next 7 days")); got != "task:Fix login bug | task:Payment retries" {
		t.Errorf("next 7 days sorted by due: %s", got)
	}
	if got := titles(run("is:overdue")); got != "task:Rotate certificates" {
		t.Errorf("overdue: %s", got)
	}
	if got := titles(run("login this week")); got != "task:Fix login bug" {
		t.Errorf("text + date: %s", got)
	}
	if got := titles(run("@sam priority:high type:bug")); got != "task:Fix login bug" {
		t.Errorf("qualifiers: %s", got)
	}
	if got := titles(run("status:\"in progress\"")); got != "task:Payment retries" {
		t.Errorf("status label: %s", got)
	}
	if got := titles(run("has:file")); got != "task:Release notes" {
		t.Errorf("has:file: %s", got)
	}
	if got := titles(run("project:ops")); got != "task:Rotate certificates" {
		t.Errorf("project: %s", got)
	}
	if got := titles(run("login", func(o *Options) { o.ProjectID = "ops" })); got != "project:Ops" {
		t.Errorf("scope: %s", got)
	}
	if got := titles(run("")); got != "" {
		t.Errorf("empty: %s", got)
	}
	r := run("due:nope")
	if len(r.Errors) != 1 || len(r.Results) != 0 {
		t.Errorf("bad qualifier: %+v", r)
	}
}
