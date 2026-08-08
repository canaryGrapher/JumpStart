package analytics

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// recorder is a stand-in PostHog that captures whatever the client posts.
type recorder struct {
	mu     sync.Mutex
	events []Event
	status int
	seen   chan struct{}
}

func newRecorder(status int) *recorder {
	return &recorder{status: status, seen: make(chan struct{}, 32)}
}

func (r *recorder) handler(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	var payload batchPayload
	_ = json.Unmarshal(body, &payload)

	r.mu.Lock()
	r.events = append(r.events, payload.Batch...)
	r.mu.Unlock()

	w.WriteHeader(r.status)
	select {
	case r.seen <- struct{}{}:
	default:
	}
}

func (r *recorder) captured() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

func (r *recorder) wait(t *testing.T) {
	t.Helper()
	select {
	case <-r.seen:
	case <-time.After(3 * time.Second):
		t.Fatal("no batch reached the server")
	}
}

func newTestClient(t *testing.T, url string) *Client {
	t.Helper()
	return New(Options{APIKey: "phc_test", Host: url, Version: "1.2.3", Dir: t.TempDir()})
}

func TestTrackDeliversEventWithGlobalProps(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.Track("process_started", map[string]any{"runtime": "node", "succeeded": true})
	c.Close(3 * time.Second)
	rec.wait(t)

	events := rec.captured()
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	ev := events[0]
	if ev.Event != "process_started" {
		t.Errorf("event name = %q", ev.Event)
	}
	if ev.DistinctID == "" {
		t.Error("distinct_id is empty")
	}
	for _, key := range []string{"app_version", "os", "arch", "session_id", "is_first_session", "update_channel"} {
		if _, ok := ev.Properties[key]; !ok {
			t.Errorf("global property %q missing", key)
		}
	}
	if ev.Properties["runtime"] != "node" {
		t.Errorf("caller property lost: %v", ev.Properties["runtime"])
	}
}

// The emitter, not the caller, is the last place a path can be stopped.
func TestTrackSanitizesCallerProperties(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.Track("project_created", map[string]any{"root": "/Users/yash/code/acme-api"})
	c.Close(3 * time.Second)
	rec.wait(t)

	if got := rec.captured()[0].Properties["root"]; got != "redacted" {
		t.Errorf("path reached the wire as %v", got)
	}
}

func TestTrackIsNoOpWhenConsentIsOff(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if err := c.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	c.Track("process_started", nil)
	c.Close(time.Second)

	if n := len(rec.captured()); n != 0 {
		t.Fatalf("sent %d events with consent off", n)
	}
	if c.EventCount() != 0 {
		t.Fatalf("counted %d events with consent off", c.EventCount())
	}
}

func TestConsentRoundTripsThroughDisk(t *testing.T) {
	dir := t.TempDir()
	if !LoadConsent(dir).Enabled {
		t.Fatal("a fresh install should default to enabled")
	}
	if err := SaveConsent(dir, false); err != nil {
		t.Fatalf("SaveConsent: %v", err)
	}
	state := LoadConsent(dir)
	if state.Enabled {
		t.Error("consent did not persist")
	}
	if state.DecidedAt == 0 {
		t.Error("DecidedAt not stamped")
	}
}

func TestNilClientIsSafe(t *testing.T) {
	var c *Client
	c.Track("app_launched", map[string]any{"a": 1})
	c.TrackOnce("k", "panel_opened", nil)
	c.Close(time.Second)
	if c.Enabled() || c.Configured() || c.SessionID() != "" || c.EventCount() != 0 {
		t.Error("nil client reported state")
	}
}

func TestTrackOnceDeduplicatesPerSession(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	for i := 0; i < 5; i++ {
		c.TrackOnce("panel:git", "panel_opened", map[string]any{"panel": "git"})
	}
	c.Close(3 * time.Second)
	rec.wait(t)

	if n := len(rec.captured()); n != 1 {
		t.Fatalf("got %d events, want 1", n)
	}
}

// A dev laptop on a plane must not lose its session.
func TestUndeliverableEventsLandInTheQueue(t *testing.T) {
	rec := newRecorder(http.StatusInternalServerError)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	dir := t.TempDir()
	c := New(Options{APIKey: "phc_test", Host: srv.URL, Version: "1.0.0", Dir: dir})
	c.Track("app_launched", nil)
	c.Close(3 * time.Second)

	data, err := os.ReadFile(filepath.Join(dir, "analytics_queue.ndjson"))
	if err != nil {
		t.Fatalf("queue file not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("queue file is empty")
	}
}

func TestTurningAnalyticsOffPurgesTheQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analytics_queue.ndjson")
	if err := os.WriteFile(path, []byte(`{"event":"app_launched"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Options{APIKey: "phc_test", Dir: dir})
	if err := c.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("queued events survived an opt-out")
	}
}

func TestRefIsStableAndOpaque(t *testing.T) {
	a := Ref("install-1", "project-abc")
	if a == "" || a == "project-abc" || len(a) != 12 {
		t.Fatalf("Ref = %q", a)
	}
	if Ref("install-1", "project-abc") != a {
		t.Error("Ref is not stable")
	}
	if Ref("install-2", "project-abc") == a {
		t.Error("Ref is not scoped to the install")
	}
}
