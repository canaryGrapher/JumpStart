package search

import (
	"strings"
	"testing"
)

func TestAIPromptCarriesVocabularyAndDate(t *testing.T) {
	v := BuildVocab(fixture())
	sys, user := AIPrompt("what is sam doing this week", v, today)
	if !strings.Contains(sys, "this_quarter") || !strings.Contains(user, "Friday 2026-10-09") ||
		!strings.Contains(user, "Sam") || !strings.Contains(user, "inprogress (In Progress)") || !strings.Contains(user, "Web Store") {
		t.Fatalf("prompt missing context:\n%s\n%s", sys, user)
	}
}

func TestParseAIPlanValidates(t *testing.T) {
	v := BuildVocab(fixture())
	plan, err := ParseAIPlan("Sure! ```json\n"+`{"query":{"duePreset":"this_week","assignees":["sam","Nobody"],"statuses":["To Do"],"labels":["AUTH"]},"projects":["web store","Mars"],"explain":"Sam's work this week"}`+"\n```", v)
	if err != nil {
		t.Fatal(err)
	}
	q := plan.Query
	if q.DuePreset != "this_week" || strings.Join(q.Assignees, ",") != "Sam" || strings.Join(q.Statuses, ",") != "todo" ||
		strings.Join(q.Labels, ",") != "auth" || strings.Join(plan.Projects, ",") != "Web Store" {
		t.Fatalf("plan %+v", plan)
	}
	if len(plan.Warnings) != 2 {
		t.Errorf("warnings %v", plan.Warnings)
	}
	res, err := RunPlan(fixture(), plan, "", nil, today)
	if err != nil || len(res) != 1 || res[0].Title != "Fix login bug" {
		t.Fatalf("run: %+v %v", res, err)
	}

	for _, bad := range []string{
		"no json here",
		`{"query":{"dueFrom":"10/9/2026"}}`,
		`{"query":{"acceptance":"maybe"}}`,
		`{"query":{"assignees":["Nobody"]}}`, // everything unknown -> empty filter
		`{"query":{}}`,
	} {
		if _, err := ParseAIPlan(bad, v); err == nil {
			t.Errorf("ParseAIPlan(%s) should fail", bad)
		}
	}
	plan, err = ParseAIPlan(`{"query":{"duePreset":"someday","overdue":true}}`, v)
	if err != nil || plan.Query.DuePreset != "" || !plan.Query.Overdue || len(plan.Warnings) != 1 {
		t.Errorf("unknown preset should be dropped: %+v %v", plan, err)
	}
}
