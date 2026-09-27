package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestNormalizeLabels(t *testing.T) {
	got := NormalizeLabels([]string{" Bug ", "bug", "feature", ""})
	want := []string{"Bug", "feature"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeLabels = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSetIssueLabels(t *testing.T) {
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
	if err := c.SetIssueLabels(context.Background(), "I1", []string{"L1", "L2"}); err != nil {
		t.Fatal(err)
	}
	if saw["issueId"] != "I1" {
		t.Errorf("issueId = %v", saw["issueId"])
	}
	ids, _ := saw["labelIds"].([]any)
	if len(ids) != 2 {
		t.Fatalf("labelIds = %v", saw["labelIds"])
	}
}

func TestListRepoLabels(t *testing.T) {
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"repository":{"labels":{"pageInfo":{"hasNextPage":false},"nodes":[
			{"id":"L1","name":"bug","color":"d73a4a","description":""},
			{"id":"L2","name":"feature","color":"a2eeef","description":"new"}
		]}}}}`))
	})
	c := New("tok")
	labels, err := c.ListRepoLabels(context.Background(), "acme/app")
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 2 || labels[0].Name != "bug" || labels[0].ID != "L1" {
		t.Fatalf("labels = %+v", labels)
	}
}
