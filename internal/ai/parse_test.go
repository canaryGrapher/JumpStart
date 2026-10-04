package ai

import "testing"

func TestParseEnrichToleratesShapeDrift(t *testing.T) {
	raw := `{
	  "description": "Fix login",
	  "acceptance": "User lands on dashboard",
	  "subtasks": ["Wire redirect", {"title": "Add tests"}],
	  "priority": "HIGH",
	  "labels": "bug, auth"
	}`
	got, err := ParseEnrich(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "Fix login" || got.Priority != "high" {
		t.Fatalf("got %+v", got)
	}
	if len(got.Acceptance) != 1 || got.Acceptance[0] != "User lands on dashboard" {
		t.Fatalf("acceptance = %v", got.Acceptance)
	}
	if len(got.Subtasks) != 2 {
		t.Fatalf("subtasks = %v", got.Subtasks)
	}
	if len(got.Labels) != 2 {
		t.Fatalf("labels = %v", got.Labels)
	}
}

func TestParseEnrichRejectsBoolAcceptanceAlone(t *testing.T) {
	// Bool acceptance is ignored; with no other fields this is empty.
	_, err := ParseEnrich(`{"acceptance": true}`)
	if err == nil {
		t.Fatal("expected error for empty usable payload")
	}
}

func TestParseChatRecoversStoriesFromDrift(t *testing.T) {
	raw := `{
	  "reply": "Here is a plan",
	  "stories": [{
	    "title": "Login",
	    "acceptance": true,
	    "priority": "Medium",
	    "tasks": [
	      {"description": "Build the form"},
	      {"title": "Add session cookie"}
	    ]
	  }]
	}`
	got := ParseChat(raw)
	if got.Reply != "Here is a plan" {
		t.Fatalf("reply = %q", got.Reply)
	}
	if len(got.Stories) != 1 {
		t.Fatalf("stories = %d", len(got.Stories))
	}
	s := got.Stories[0]
	if s.Priority != "medium" {
		t.Fatalf("priority = %q", s.Priority)
	}
	if len(s.Tasks) != 2 || s.Tasks[0].Title != "Build the form" || s.Tasks[1].Title != "Add session cookie" {
		t.Fatalf("tasks = %+v", s.Tasks)
	}
}

func TestExtractJSONStripsFences(t *testing.T) {
	raw := "```json\n{\"reply\":\"hi\",\"stories\":[]}\n```"
	if got := ExtractJSON(raw); got != `{"reply":"hi","stories":[]}` {
		t.Fatalf("got %q", got)
	}
}
