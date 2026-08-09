package analytics

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type ga4Body struct {
	ClientID           string `json:"client_id"`
	TimestampMicros    int64  `json:"timestamp_micros"`
	NonPersonalizedAds bool   `json:"non_personalized_ads"`
	Events             []struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params"`
	} `json:"events"`
}

type recorder struct {
	mu         sync.Mutex
	bodies     []ga4Body
	queries    []string
	status     int
	seen       chan struct{}
}

func newRecorder(status int) *recorder {
	return &recorder{status: status, seen: make(chan struct{}, 32)}
}

func (r *recorder) handler(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	var payload ga4Body
	_ = json.Unmarshal(body, &payload)
	r.mu.Lock()
	r.bodies = append(r.bodies, payload)
	r.queries = append(r.queries, req.URL.RawQuery)
	r.mu.Unlock()

	w.WriteHeader(r.status)
	select {
	case r.seen <- struct{}{}:
	default:
	}
}

func (r *recorder) eventNames() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var names []string
	for _, b := range r.bodies {
		for _, e := range b.Events {
			names = append(names, e.Name)
		}
	}
	return names
}

func (r *recorder) firstParams() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.bodies) == 0 || len(r.bodies[0].Events) == 0 {
		return nil
	}
	return r.bodies[0].Events[0].Params
}

func (r *recorder) firstQuery() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queries) == 0 {
		return ""
	}
	return r.queries[0]
}

func (r *recorder) wait(t *testing.T) {
	t.Helper()
	select {
	case <-r.seen:
	case <-time.After(3 * time.Second):
		t.Fatal("no batch reached the server")
	}
}

func newTestClient(t *testing.T, collectURL string) *Client {
	t.Helper()
	return New(Options{
		MeasurementID: "G-TEST",
		APISecret:     "test-secret",
		Version:       "1.2.3",
		Dir:           t.TempDir(),
		CollectURL:    collectURL,
	})
}

func TestTrackDeliversEventWithGlobalProps(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.Track("process_started", map[string]any{"runtime": "node", "succeeded": true})
	c.Close(3 * time.Second)
	rec.wait(t)

	names := rec.eventNames()
	if len(names) != 1 {
		t.Fatalf("got %d events, want 1", len(names))
	}
	if names[0] != "process_started" {
		t.Errorf("event name = %q", names[0])
	}

	rec.mu.Lock()
	clientID := rec.bodies[0].ClientID
	rec.mu.Unlock()
	if clientID == "" {
		t.Error("client_id is empty")
	}

	params := rec.firstParams()
	for _, key := range []string{"app", "platform", "app_version", "os", "arch", "session_id", "is_first_session", "update_channel", "runtime"} {
		if _, ok := params[key]; !ok {
			t.Errorf("global property %q missing", key)
		}
	}
	if _, ok := params["$lib"]; ok {
		t.Error("PostHog $lib must not be sent to GA4")
	}
	if params["app"] != "desktop" {
		t.Errorf("app=%v", params["app"])
	}
	if params["platform"] != runtime.GOOS {
		t.Errorf("platform=%v, want %s", params["platform"], runtime.GOOS)
	}
	if params["runtime"] != "node" {
		t.Errorf("caller property lost: %v", params["runtime"])
	}
	if params["is_key_event"] != true {
		t.Errorf("is_key_event = %v, want true for process_started", params["is_key_event"])
	}

	q, _ := url.ParseQuery(rec.firstQuery())
	if q.Get("measurement_id") != "G-TEST" {
		t.Errorf("measurement_id = %q", q.Get("measurement_id"))
	}
	if q.Get("api_secret") != "test-secret" {
		t.Errorf("api_secret = %q", q.Get("api_secret"))
	}
}

func TestTrackIsNoOpWhenAPISecretMissing(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := New(Options{
		MeasurementID: "G-TEST",
		APISecret:     "",
		Version:       "1.0.0",
		Dir:           t.TempDir(),
		CollectURL:    srv.URL,
	})
	if c.Configured() {
		t.Fatal("Configured should be false without API secret")
	}
	c.Track("app_launched", nil)
	c.Close(time.Second)
	if c.EventCount() != 0 {
		t.Fatalf("counted %d events without API secret", c.EventCount())
	}
	if n := len(rec.eventNames()); n != 0 {
		t.Fatalf("sent %d events without API secret", n)
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

	if got := rec.firstParams()["root"]; got != "redacted" {
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

	if n := len(rec.eventNames()); n != 0 {
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

	if n := len(rec.eventNames()); n != 1 {
		t.Fatalf("got %d events, want 1", n)
	}
}

// A dev laptop on a plane must not lose its session.
func TestUndeliverableEventsLandInTheQueue(t *testing.T) {
	rec := newRecorder(http.StatusInternalServerError)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	dir := t.TempDir()
	c := New(Options{
		MeasurementID: "G-TEST",
		APISecret:     "test-secret",
		Version:       "1.0.0",
		Dir:           dir,
		CollectURL:    srv.URL,
	})
	c.Track("app_launched", nil)
	c.Close(3 * time.Second)

	data, err := os.ReadFile(filepath.Join(dir, "analytics_queue.ndjson"))
	if err != nil {
		t.Fatalf("queue file not written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("queue file is empty")
	}
	if !strings.Contains(string(data), `"name":"app_launched"`) {
		t.Fatalf("queue entry missing event name: %s", data)
	}
}

func TestTurningAnalyticsOffPurgesTheQueue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analytics_queue.ndjson")
	if err := os.WriteFile(path, []byte(`{"name":"app_launched"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New(Options{MeasurementID: "G-TEST", APISecret: "test-secret", Dir: dir})
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
