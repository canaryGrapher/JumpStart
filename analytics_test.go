package main

import (
	"errors"
	"testing"
	"time"

	"devdeck/internal/analytics"
	"devdeck/internal/model"
)

// Every bounded helper in the main package feeds a property that goes
// through analytics.Sanitize on its way out. A helper that returns a value
// Sanitize rejects produces a chart full of "redacted", which looks like
// data until you try to use it.
func TestBoundedHelpersSurviveSanitize(t *testing.T) {
	values := map[string]string{
		"themeMode(dark)":       themeMode("dark"),
		"themeMode(garbage)":    themeMode("something else"),
		"taskKind(story)":       taskKind("story"),
		"taskKind(empty)":       taskKind(""),
		"column(inprogress)":    column("inprogress"),
		"column(empty)":         column(""),
		"column(unknown)":       column("archived"),
		"priority(high)":        priority("high"),
		"priority(empty)":       priority(""),
		"testScope(project)":    testScope(""),
		"testScope(process)":    testScope("proc-1"),
		"topScoreBucket(empty)": topScoreBucket(nil),
	}
	for name, value := range values {
		out := analytics.Sanitize(map[string]any{"v": value})
		if out["v"] != value {
			t.Errorf("%s = %q, which Sanitize rejected as %v", name, value, out["v"])
		}
	}
}

func TestOutcomeReportsBoundedFailureReason(t *testing.T) {
	start := time.Now().Add(-250 * time.Millisecond)
	props := outcome(start, errors.New("open /Users/yash/api: permission denied"), map[string]any{
		"action": "push",
	})

	if props["succeeded"] != false {
		t.Errorf("succeeded = %v, want false", props["succeeded"])
	}
	if props["failure_reason"] != analytics.ReasonPermission {
		t.Errorf("failure_reason = %v, want %q", props["failure_reason"], analytics.ReasonPermission)
	}
	if props["action"] != "push" {
		t.Errorf("extra property dropped: %v", props["action"])
	}
	ms, ok := props["duration_ms"].(int64)
	if !ok || ms < 200 {
		t.Errorf("duration_ms = %v, want a plausible int64", props["duration_ms"])
	}

	// The raw error must not survive the sanitiser under any key.
	for key, value := range analytics.Sanitize(props) {
		if s, ok := value.(string); ok && len(s) > 32 {
			t.Errorf("property %s leaked a long string: %q", key, s)
		}
	}
}

func TestOutcomeOnSuccess(t *testing.T) {
	props := outcome(time.Now(), nil, nil)
	if props["succeeded"] != true {
		t.Errorf("succeeded = %v, want true", props["succeeded"])
	}
	if props["failure_reason"] != analytics.ReasonNone {
		t.Errorf("failure_reason = %q, want empty", props["failure_reason"])
	}
}

func TestSprintDays(t *testing.T) {
	cases := []struct {
		sprint model.Sprint
		want   int
	}{
		{model.Sprint{StartDate: "2026-08-01", EndDate: "2026-08-15"}, 14},
		{model.Sprint{StartDate: "2026-08-01", EndDate: "2026-08-01"}, 0},
		{model.Sprint{}, 0},
		{model.Sprint{StartDate: "2026-08-15", EndDate: "2026-08-01"}, 0}, // reversed
		{model.Sprint{StartDate: "not a date", EndDate: "2026-08-01"}, 0},
	}
	for _, c := range cases {
		if got := sprintDays(c.sprint); got != c.want {
			t.Errorf("sprintDays(%+v) = %d, want %d", c.sprint, got, c.want)
		}
	}
}

func TestAgeHoursIgnoresMissingTimestamps(t *testing.T) {
	if got := ageHours(0); got != 0 {
		t.Errorf("ageHours(0) = %d, want 0 (not an epoch-sized number)", got)
	}
	if got := ageHours(time.Now().Add(time.Hour).UnixMilli()); got != 0 {
		t.Errorf("ageHours(future) = %d, want 0", got)
	}
	if got := ageHours(time.Now().Add(-3 * time.Hour).UnixMilli()); got != 3 {
		t.Errorf("ageHours(-3h) = %d, want 3", got)
	}
}

// A nil analytics client is the state every App is in before Startup runs,
// and the one dev builds stay in. Nothing may panic.
func TestTrackHelpersTolerateNilClient(t *testing.T) {
	a := NewApp()
	a.track("app_launched", map[string]any{"project_count": 3})
	a.trackOnce("k", "panel_opened", map[string]any{"panel": "git"})
	a.trackImport("file", 2, nil)
	a.trackDepsInstall("npm", time.Now(), nil)
	a.trackProcessStopped("proc-1", "single", nil)
	a.trackTaskChanges("p1", nil, []model.Task{{ID: "t1", Title: "x"}})
	a.trackSprintChanges("p1", nil, []model.Sprint{{ID: "s1"}})
	if ref := a.ref("project-1"); ref != "" {
		t.Errorf("ref on a nil client = %q, want empty", ref)
	}
	if a.GetAnalyticsSettings().Enabled {
		t.Error("a nil client should not report analytics as enabled")
	}
}
