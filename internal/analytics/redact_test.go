package analytics

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// leaky is the load-bearing table: realistic values that must never reach
// GA4, whatever a careless call site does with them. If a future change
// widens Sanitize, this test is what catches it.
var leaky = []string{
	"/Users/yash/code/acme-api",
	"/home/yash/work/client-project/backend",
	`C:\Users\yash\Projects\secret-startup`,
	"git@github.com:acme/private-repo.git",
	"https://github.com/acme/private-repo",
	"https://gitlab.internal.acme.com/team/api",
	"feat: add billing to the checkout flow",
	"yash@example.com",
	"ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789",
	"glpat-xxxxxxxxxxxxxxxxxxxx",
	"npm run dev --workspace=@acme/web",
	"postgres://user:password@localhost:5432/acme",
	"../../etc/passwd",
	"DATABASE_URL=postgres://localhost/acme",
	"Why is my checkout flow returning a 500 on staging?",
}

func TestSanitizeDropsIdentifyingValues(t *testing.T) {
	for _, value := range leaky {
		props := Sanitize(map[string]any{"leaked": value})
		got, _ := props["leaked"].(string)
		if got != "redacted" {
			t.Errorf("Sanitize kept %q as %q, want \"redacted\"", value, got)
		}
	}
}

func TestSanitizeDropsIdentifyingValuesInSlices(t *testing.T) {
	props := Sanitize(map[string]any{"runtimes": []string{"node", "/Users/yash/api"}})
	got, ok := props["runtimes"].(string)
	if !ok {
		t.Fatalf("Sanitize returned %#v, want a comma-joined string", props["runtimes"])
	}
	if got != "node,redacted" {
		t.Errorf("got %q, want \"node,redacted\"", got)
	}
}

func TestSanitizeJoinsStringSlicesForGA4(t *testing.T) {
	props := Sanitize(map[string]any{"runtimes": []string{"node", "go", "python"}})
	got, ok := props["runtimes"].(string)
	if !ok {
		t.Fatalf("Sanitize returned %#v, want string", props["runtimes"])
	}
	if got != "node,go,python" {
		t.Errorf("got %q, want \"node,go,python\"", got)
	}
	if props := Sanitize(map[string]any{"runtimes": []string{}}); props["runtimes"] != "" {
		t.Errorf("empty slice = %#v, want \"\"", props["runtimes"])
	}
}

func TestSanitizeKeepsBoundedValues(t *testing.T) {
	in := map[string]any{
		"runtime":        "node",
		"manager":        "go",
		"project_ref":    "9f2c1ab73de4",
		"failure_reason": "port_in_use",
		"panel":          "git",
		"count":          12,
		"duration_ms":    int64(1450),
		"succeeded":      true,
		"score":          0.42,
		"nothing":        nil,
	}
	out := Sanitize(in)
	for key, want := range in {
		if out[key] != want {
			t.Errorf("Sanitize(%s) = %v, want %v", key, out[key], want)
		}
	}
}

func TestSanitizeConvertsErrorsToBoundedReasons(t *testing.T) {
	out := Sanitize(map[string]any{
		"err": errors.New("open /Users/yash/api/.env: permission denied"),
	})
	if out["err"] != ReasonPermission {
		t.Errorf("error value = %v, want %q", out["err"], ReasonPermission)
	}
}

func TestSanitizeRejectsLongValues(t *testing.T) {
	long := strings.Repeat("a", maxValueLen+1)
	if got := Sanitize(map[string]any{"v": long})["v"]; got != "redacted" {
		t.Errorf("long value survived: %v", got)
	}
}

func TestFailureReasonIsBounded(t *testing.T) {
	cases := map[string]string{
		"": ReasonNone,
		`exec: "pnpm": executable file not found in $PATH`:  ReasonCommandNotFound,
		"listen tcp :3000: bind: address already in use":    ReasonPortInUse,
		"open /Users/yash/x: permission denied":             ReasonPermission,
		"chdir /Users/yash/gone: no such file or directory": ReasonMissingPath,
		"exit status 1":                                         ReasonNonZeroExit,
		"GitHub API 401: Bad credentials":                       ReasonAuth,
		"dial tcp 127.0.0.1:11434: connect: connection refused": ReasonNetwork,
		"context deadline exceeded":                             ReasonTimeout,
		"no access token configured — add one in Preferences":   ReasonNotConfigured,
		"something nobody has seen before":                      ReasonOther,
	}
	for msg, want := range cases {
		var err error
		if msg != "" {
			err = errors.New(msg)
		}
		if got := FailureReason(err); got != want {
			t.Errorf("FailureReason(%q) = %q, want %q", msg, got, want)
		}
	}
}

// Errors are the likeliest accidental leak: they routinely embed the path,
// command or remote that failed. FailureReason must collapse every one of
// them onto the bounded set, whatever they wrap.
func TestFailureReasonNeverLeaksTheMessage(t *testing.T) {
	for _, value := range leaky {
		err := fmt.Errorf("operation failed: %s", value)
		reason := FailureReason(err)
		if !isKnownReason(reason) {
			t.Errorf("FailureReason(%q) produced unbounded value %q", value, reason)
		}
		if strings.Contains(err.Error(), reason) && reason == ReasonOther {
			t.Errorf("FailureReason(%q) echoed the message", value)
		}
	}
}

func isKnownReason(r string) bool {
	for _, known := range []string{
		ReasonNone, ReasonCommandNotFound, ReasonPortInUse, ReasonPermission,
		ReasonMissingPath, ReasonNonZeroExit, ReasonNotFound, ReasonAuth,
		ReasonNetwork, ReasonTimeout, ReasonConflict, ReasonNotConfigured,
		ReasonParse, ReasonOther,
	} {
		if r == known {
			return true
		}
	}
	return false
}

func TestPathDepthReplacesThePath(t *testing.T) {
	cases := map[string]int{
		"":                           0,
		"/":                          0,
		"/Users/yash/code/acme-api":  4,
		"/home/yash/work":            3,
		"relative/path/here":         3,
		`C:\Users\yash\Projects\api`: 4,
	}
	for path, want := range cases {
		if got := PathDepth(path); got != want {
			t.Errorf("PathDepth(%q) = %d, want %d", path, got, want)
		}
	}
}

func TestBucket(t *testing.T) {
	cases := map[float64]string{-1: "none", 0: "none", 0.1: "low", 0.5: "medium", 0.9: "high"}
	for score, want := range cases {
		if got := Bucket(score); got != want {
			t.Errorf("Bucket(%v) = %q, want %q", score, got, want)
		}
	}
}
