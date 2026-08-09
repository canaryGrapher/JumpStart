package analytics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEventCategoryKnownEvents(t *testing.T) {
	cases := map[string]Category{
		"app_launched":         CatLifecycle,
		"consent_decided":      CatLifecycle,
		"project_created":      CatOnboarding,
		"processes_accepted":   CatOnboarding,
		"process_started":      CatProcesses,
		"logs_opened":          CatProcesses,
		"ports_viewed":         CatProcesses,
		"env_file_edited":      CatProcesses,
		"git_action_performed": CatGitDocker,
		"task_created":         CatKanban,
		"roadmap_opened":       CatKanban,
		"ai_chat_message_sent": CatAI,
		"update_checked":       CatUpdates,
		"panel_opened":         CatUIPanels,
		"unknown_thing":        CatUIPanels,
	}
	for name, want := range cases {
		if got := EventCategory(name); got != want {
			t.Errorf("EventCategory(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCategoriesForLevel(t *testing.T) {
	full := CategoriesForLevel(LevelFull)
	for _, c := range AllCategories {
		if !full[string(c)] {
			t.Fatalf("full missing %s", c)
		}
	}

	bal := CategoriesForLevel(LevelBalanced)
	if bal[string(CatUIPanels)] {
		t.Fatal("balanced should disable ui_panels")
	}
	if !bal[string(CatProcesses)] {
		t.Fatal("balanced should keep processes")
	}

	min := CategoriesForLevel(LevelMinimal)
	if !min[string(CatLifecycle)] || !min[string(CatUpdates)] {
		t.Fatal("minimal should keep lifecycle and updates")
	}
	if min[string(CatAI)] || min[string(CatProcesses)] {
		t.Fatal("minimal should drop ai and processes")
	}
}

func TestInferDetailLevel(t *testing.T) {
	if InferDetailLevel(CategoriesForLevel(LevelFull)) != LevelFull {
		t.Fatal("expected full")
	}
	if InferDetailLevel(CategoriesForLevel(LevelBalanced)) != LevelBalanced {
		t.Fatal("expected balanced")
	}
	if InferDetailLevel(CategoriesForLevel(LevelMinimal)) != LevelMinimal {
		t.Fatal("expected minimal")
	}
	custom := CategoriesForLevel(LevelFull)
	custom[string(CatAI)] = false
	if InferDetailLevel(custom) != LevelCustom {
		t.Fatal("expected custom")
	}
}

func TestEnvKeyDiff(t *testing.T) {
	before := map[string]string{"A": "1", "B": "2"}
	after := map[string]string{"B": "9", "C": "3"}
	n, added, removed := EnvKeyDiff(before, after)
	if n != 2 || added != 1 || removed != 1 {
		t.Fatalf("got varCount=%d added=%d removed=%d", n, added, removed)
	}
}

func TestTrackRespectsCategoryGate(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if err := c.SetDetailLevel(LevelMinimal); err != nil {
		t.Fatal(err)
	}
	c.Track("process_started", map[string]any{"runtime": "node"})
	c.Track("app_launched", map[string]any{"project_count": 1})
	c.Close(3 * time.Second)
	rec.wait(t)

	names := rec.eventNames()
	if len(names) != 1 || names[0] != "app_launched" {
		t.Fatalf("got %#v, want only app_launched", names)
	}
}
