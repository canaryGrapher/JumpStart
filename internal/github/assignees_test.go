package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestParseAssignees(t *testing.T) {
	got := ParseAssignees(" alice, bob ,alice, ,Carol ")
	want := []string{"alice", "bob", "Carol"}
	if len(got) != len(want) {
		t.Fatalf("ParseAssignees = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestFormatAssigneesStable(t *testing.T) {
	if got := FormatAssignees([]string{"bob", "alice", "bob"}); got != "alice, bob" {
		t.Errorf("FormatAssignees = %q, want sorted unique", got)
	}
}

func TestAssigneesEqual(t *testing.T) {
	if !AssigneesEqual([]string{"a", "b"}, []string{"b", "a"}) {
		t.Error("order should not matter")
	}
	if AssigneesEqual([]string{"a"}, []string{"a", "b"}) {
		t.Error("different sets should not be equal")
	}
}

func TestListAssignableUsers(t *testing.T) {
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"repository":{"assignableUsers":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"id":"U1","login":"alice","name":"Alice","avatarUrl":"https://example/a"},
			{"id":"U2","login":"bob","name":"","avatarUrl":"https://example/b"}
		]}}}}`))
	})
	c := New("tok")
	users, err := c.ListAssignableUsers(context.Background(), "acme/app")
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Login != "alice" || users[0].ID != "U1" {
		t.Fatalf("users = %+v", users)
	}
}

func TestSetIssueAssignees(t *testing.T) {
	var saw map[string]any
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]any `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		saw = body.Variables
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"updateIssue":{"issue":{"id":"I1"}}}}`))
	})
	c := New("tok")
	if err := c.SetIssueAssignees(context.Background(), "I1", []string{"U1", "U2"}); err != nil {
		t.Fatal(err)
	}
	if saw["issueId"] != "I1" {
		t.Errorf("issueId = %v", saw["issueId"])
	}
	ids, _ := saw["assigneeIds"].([]any)
	if len(ids) != 2 {
		t.Fatalf("assigneeIds = %v", saw["assigneeIds"])
	}
}
