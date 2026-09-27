package ghsync

import (
	"strings"
	"testing"

	"devdeck/internal/model"
)

func TestComposeBodyEmpty(t *testing.T) {
	if got := ComposeBody("", nil, nil); got != "" {
		t.Errorf("empty compose = %q", got)
	}
	if got := ComposeBody("Just prose", nil, nil); got != "Just prose" {
		t.Errorf("prose-only = %q", got)
	}
}

func TestComposeAndParseRoundTrip(t *testing.T) {
	body := ComposeBody("Ship the feature",
		[]model.Subtask{{Title: "Login works", Done: true}, {Title: "Logout works"}},
		[]model.Subtask{{Title: "Wire the API"}, {Title: "Add tests", Done: true}},
	)
	desc, acc, subs := ParseBody(body)
	if desc != "Ship the feature" {
		t.Errorf("desc = %q", desc)
	}
	if !sameChecklist(acc, []model.Subtask{{Title: "Login works", Done: true}, {Title: "Logout works"}}) {
		t.Errorf("acceptance = %+v", acc)
	}
	if !sameChecklist(subs, []model.Subtask{{Title: "Wire the API"}, {Title: "Add tests", Done: true}}) {
		t.Errorf("subtasks = %+v", subs)
	}
}

func TestParseBodyLeavesUnmarkedChecklistsInDescription(t *testing.T) {
	body := "Notes\n\n## Acceptance criteria\n- [ ] hand written\n"
	desc, acc, subs := ParseBody(body)
	if desc != strings.TrimSpace(body) {
		t.Errorf("unmarked body should stay in description, got %q", desc)
	}
	if len(acc) != 0 || len(subs) != 0 {
		t.Errorf("unmarked lists must not be parsed: acc=%+v subs=%+v", acc, subs)
	}
}

func TestParseBodyStripsMarkedSections(t *testing.T) {
	body := "Intro\n\n" +
		acceptanceStart + "\n## Acceptance criteria\n- [x] A\n" + acceptanceEnd +
		"\n\nTail note"
	desc, acc, _ := ParseBody(body)
	if !strings.Contains(desc, "Intro") || !strings.Contains(desc, "Tail note") {
		t.Errorf("desc should keep prose around markers, got %q", desc)
	}
	if strings.Contains(desc, acceptanceStart) || strings.Contains(desc, "- [x] A") {
		t.Errorf("marked section leaked into description: %q", desc)
	}
	if len(acc) != 1 || acc[0].Title != "A" || !acc[0].Done {
		t.Errorf("acceptance = %+v", acc)
	}
}

func TestAdoptChecklistReusesIDs(t *testing.T) {
	local := []model.Subtask{{ID: "keep", Title: "Wire the API", Done: false}}
	remote := []model.Subtask{{Title: "Wire the API", Done: true}, {Title: "New one"}}
	got := adoptChecklist(local, remote, func() string { return "new-id" })
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ID != "keep" || !got[0].Done {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].ID != "new-id" || got[1].Title != "New one" {
		t.Errorf("second = %+v", got[1])
	}
}
