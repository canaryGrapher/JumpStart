# GA4 + Clarity Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace desktop PostHog ingest with GA4 Measurement Protocol from Go, keep landing GA4+Clarity as-is, and remove PostHog from CI/docs/scripts — per `docs/superpowers/specs/2026-08-09-ga4-clarity-migration-design.md`.

**Architecture:** Transport-swap inside `internal/analytics`. Keep consent, categories, redaction, offline queue, and JumpStart event names. Stamp `DESKTOP_GA_MEASUREMENT_ID` + `DESKTOP_GA_API_SECRET` via ldflags. Clarity remains landing-only (`VITE_GA_ID` / `VITE_CLARITY_ID`).

**Tech Stack:** Go (Wails), hand-rolled GA4 MP HTTP client, React frontend bridge, Vite landing analytics, GitHub Actions.

## Global Constraints

- Do not invent, reuse, or substitute analytics IDs; empty credentials → full no-op.
- Desktop: GA4 Measurement Protocol from Go only — no gtag/Clarity in the Wails webview.
- Desktop Clarity: none (`DESKTOP_CLARITY_ID` not used).
- Keep JumpStart event names; add `is_key_event` mapping only (hybrid taxonomy).
- Landing env names stay `VITE_GA_ID` / `VITE_CLARITY_ID`.
- No WordShield work (N/A).
- Never send paths, tokens, emails, prompts, stack traces, or raw error strings.
- Do not over-engineer a new `analytics/{ga,clarity}` tree.
- Prefer GitHub **variable** for Measurement ID; **secret** for API secret (document both in wiki).
- Historical `RELEASE_NOTES.md` entries may still mention PostHog; update active wiki/README/privacy.

## File map

| File | Responsibility after migration |
|------|--------------------------------|
| `internal/analytics/analytics.go` | Options: `MeasurementID`, `APISecret`; package docs |
| `internal/analytics/transport.go` | GA4 MP `POST /mp/collect` sender |
| `internal/analytics/key_events.go` | Key-event predicate + `is_key_event` param |
| `internal/analytics/key_events_test.go` | Tests for key-event mapping |
| `internal/analytics/client.go` | Gate on both credentials; build events for GA4 |
| `internal/analytics/props.go` | Globals without PostHog `$lib*`; add `app=desktop` |
| `internal/analytics/client_test.go` | httptest asserts GA4 payload shape |
| `analytics.go` (main) | `GAMeasurementID`, `GAAPISecret` ldflag vars |
| `.github/workflows/build.yml` | Stamp desktop GA credentials |
| `frontend/src/analytics.js` | Neutral bridge comments (no PostHog) |
| `scripts/setup-posthog-dashboards.sh` | **Delete** |
| Docs (wiki, privacy, README, frontend/.env.example) | Vendor + CI var updates |

---

### Task 1: Options + configured gate (Measurement ID + API secret)

**Files:**
- Modify: `internal/analytics/analytics.go`
- Modify: `internal/analytics/client.go` (`New`, `Configured`, `Track`)
- Test: `internal/analytics/client_test.go` (update constructors; add empty-secret no-op test)

**Interfaces:**
- Consumes: nothing new
- Produces:
  - `type Options struct { MeasurementID string; APISecret string; Version string; Channel string; Dir string }`
  - `func (c *Client) Configured() bool` — true only when both `MeasurementID` and `APISecret` are non-empty
  - `Track` no-ops when either credential is empty
  - Remove `APIKey`, `Host`, and `DefaultHost` from this package

- [ ] **Step 1: Write the failing test for dual-credential gate**

Add to `internal/analytics/client_test.go`:

```go
func TestTrackIsNoOpWhenAPISecretMissing(t *testing.T) {
	rec := newRecorder(http.StatusOK)
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	defer srv.Close()

	c := New(Options{
		MeasurementID: "G-TEST",
		APISecret:     "", // missing
		Version:       "1.0.0",
		Dir:           t.TempDir(),
		CollectURL:    srv.URL, // added in Task 2; if not yet present, skip URL override and assert EventCount==0 + Configured==false only
	})
	if c.Configured() {
		t.Fatal("Configured should be false without API secret")
	}
	c.Track("app_launched", nil)
	c.Close(time.Second)
	if c.EventCount() != 0 {
		t.Fatalf("counted %d events without API secret", c.EventCount())
	}
}
```

If `CollectURL` does not exist yet, assert only `!Configured()` and `EventCount()==0` without httptest.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/analytics/ -run TestTrackIsNoOpWhenAPISecretMissing -v`  
Expected: FAIL (Options still use `APIKey` / compile error or Configured still key-only)

- [ ] **Step 3: Update Options and client gate**

In `internal/analytics/analytics.go`, replace package header + Options:

```go
// Package analytics reports anonymous product usage to Google Analytics 4
// via the Measurement Protocol from the Go process (not the Wails webview).
package analytics

// Options configures a Client.
type Options struct {
	// MeasurementID is the GA4 Measurement ID (G-…). Empty disables the client.
	MeasurementID string
	// APISecret is the GA4 Measurement Protocol API secret. Empty disables the client.
	APISecret string
	// Version is the running build's version, e.g. "1.4.2" or "dev".
	Version string
	// Channel is the update channel: "stable" or "beta".
	Channel string
	// Dir is the app data directory (~/.jumpstart). Empty resolves it.
	Dir string
	// CollectURL overrides the GA4 MP endpoint for tests. Empty uses DefaultCollectURL.
	CollectURL string
}

// DefaultCollectURL is Google's GA4 Measurement Protocol collect endpoint.
const DefaultCollectURL = "https://www.google-analytics.com/mp/collect"
```

Remove `DefaultHost` and PostHog comments.

In `client.go` `New`:

```go
func New(opts Options) *Client {
	if opts.Dir == "" {
		opts.Dir = DataDir()
	}
	id, first := installID(opts.Dir)
	consent := LoadConsent(opts.Dir)
	c := &Client{
		opts:        opts,
		distinct:    id,
		session:     randomID(),
		events:      make(chan Event, bufferSize),
		queue:       newQueue(filepath.Join(opts.Dir, "analytics_queue.ndjson")),
		sender:      newSender(opts), // Task 2 will redefine newSender
		done:        make(chan struct{}),
		detailLevel: consent.DetailLevel,
		categories:  MergeCategories(consent.Categories),
	}
	c.global = globalProps(opts, first, c.session)
	c.enabled.Store(consent.Enabled)
	return c
}
```

Temporarily, if `newSender` still expects old args, use a stub that compiles — Task 2 completes sender.

```go
func (c *Client) Configured() bool {
	return c != nil && c.opts.MeasurementID != "" && c.opts.APISecret != ""
}

func (c *Client) Track(name string, props map[string]any) {
	if c == nil || name == "" || !c.enabled.Load() || !c.Configured() || c.closed.Load() {
		return
	}
	// ... rest unchanged
}
```

Update all tests that construct `Options{APIKey: ...}` to `Options{MeasurementID: "G-TEST", APISecret: "secret", CollectURL: url}` (CollectURL wired in Task 2; until then pass Host-equivalent via temporary field only if needed — prefer finishing Task 1+2 in one worker session).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/analytics/ -count=1`  
Expected: compile may still fail until Task 2 updates `newSender` — if so, continue immediately to Task 2 in the same session without committing a broken tree.

- [ ] **Step 5: Commit** (only after Task 2 leaves `./internal/analytics` green, or squash Tasks 1–3 into one commit)

```bash
git add internal/analytics/analytics.go internal/analytics/client.go internal/analytics/client_test.go
git commit -m "$(cat <<'EOF'
refactor(analytics): gate client on GA4 measurement ID and API secret

EOF
)"
```

---

### Task 2: GA4 Measurement Protocol transport

**Files:**
- Modify: `internal/analytics/transport.go`
- Modify: `internal/analytics/client.go` (`build`, comments)
- Modify: `internal/analytics/queue.go` (comment only: “next launch” not “PostHog”)
- Modify: `internal/analytics/client_test.go` (`recorder` parses GA4 body)
- Modify: `internal/analytics/analytics.go` (`batchSize` must stay ≤ 25)

**Interfaces:**
- Consumes: `Options.MeasurementID`, `Options.APISecret`, `Options.CollectURL`
- Produces:
  - Internal queued `Event` shape (JSON):
    ```go
    type Event struct {
    	Name      string         `json:"name"`
    	ClientID  string         `json:"client_id"`
    	Params    map[string]any `json:"params"`
    	Timestamp string         `json:"timestamp"` // RFC3339; converted to timestamp_micros on send
    }
    ```
  - `func newSender(opts Options) *sender`
  - `func (s *sender) send(batch []Event) error` posts one or more MP requests (≤25 events each) to  
    `{CollectURL}?measurement_id={id}&api_secret={secret}`  
    Body:
    ```json
    {
      "client_id": "<install-uuid>",
      "timestamp_micros": 0,
      "non_personalized_ads": true,
      "events": [
        { "name": "process_started", "params": { "...": "..." } }
      ]
    }
    ```
  - Group batch by `ClientID` if mixed (normally one install). Use first event’s timestamp for request-level `timestamp_micros` or per-event if splitting to one event per request when timestamps differ — simplest correct approach: **one MP HTTP request per Event** OR chunk ≤25 with shared `client_id` and omit request-level timestamp (events carry params only). Prefer: single request per flush batch when all share `ClientID`, map each Event → `{name, params}`; convert RFC3339 → `timestamp_micros` on each event via GA4’s per-event support if available — GA4 MP events don’t all support per-event timestamp in all docs; use request `timestamp_micros` from the **oldest** event in the batch when sending one client_id batch.
  - 5xx / 429 → return error (queue). Other 4xx → treat as success (do not retry forever). Empty credentials → no-op success.

- [ ] **Step 1: Rewrite failing recorder tests for GA4 payload**

Replace `recorder` in `client_test.go`:

```go
type ga4Body struct {
	ClientID            string `json:"client_id"`
	TimestampMicros     int64  `json:"timestamp_micros"`
	NonPersonalizedAds  bool   `json:"non_personalized_ads"`
	Events              []struct {
		Name   string         `json:"name"`
		Params map[string]any `json:"params"`
	} `json:"events"`
}

type recorder struct {
	mu     sync.Mutex
	bodies []ga4Body
	status int
	seen   chan struct{}
}

func (r *recorder) handler(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	var payload ga4Body
	_ = json.Unmarshal(body, &payload)
	r.mu.Lock()
	r.bodies = append(r.bodies, payload)
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
```

Update `TestTrackDeliversEventWithGlobalProps` to assert:
- `eventNames()[0] == "process_started"`
- `bodies[0].ClientID != ""`
- params contain `app_version`, `os`, `arch`, `session_id`, `is_first_session`, `update_channel`, `runtime`
- query string on request includes `measurement_id` and `api_secret` (capture `req.URL.RawQuery` in recorder if useful)

Update `newTestClient`:

```go
func newTestClient(t *testing.T, url string) *Client {
	t.Helper()
	return New(Options{
		MeasurementID: "G-TEST",
		APISecret:     "test-secret",
		Version:       "1.2.3",
		Dir:           t.TempDir(),
		CollectURL:    url,
	})
}
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/analytics/ -run TestTrackDeliversEventWithGlobalProps -v`  
Expected: FAIL (still PostHog wire format)

- [ ] **Step 3: Implement GA4 transport + Event shape**

Replace `internal/analytics/transport.go` with:

```go
package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Event is the offline-queue and in-memory batch unit.
type Event struct {
	Name      string         `json:"name"`
	ClientID  string         `json:"client_id"`
	Params    map[string]any `json:"params"`
	Timestamp string         `json:"timestamp"`
}

type sender struct {
	measurementID string
	apiSecret     string
	baseURL       string
	client        *http.Client
}

func newSender(opts Options) *sender {
	base := opts.CollectURL
	if base == "" {
		base = DefaultCollectURL
	}
	return &sender{
		measurementID: opts.MeasurementID,
		apiSecret:     opts.APISecret,
		baseURL:       strings.TrimRight(base, "/"),
		client:        &http.Client{Timeout: sendTimeout},
	}
}

type mpEvent struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

type mpPayload struct {
	ClientID           string    `json:"client_id"`
	TimestampMicros    int64     `json:"timestamp_micros,omitempty"`
	NonPersonalizedAds bool      `json:"non_personalized_ads"`
	Events             []mpEvent `json:"events"`
}

func (s *sender) send(batch []Event) error {
	if len(batch) == 0 || s.measurementID == "" || s.apiSecret == "" {
		return nil
	}
	// GA4 allows max 25 events per request.
	for i := 0; i < len(batch); i += 25 {
		end := i + 25
		if end > len(batch) {
			end = len(batch)
		}
		if err := s.sendChunk(batch[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *sender) sendChunk(batch []Event) error {
	clientID := batch[0].ClientID
	events := make([]mpEvent, 0, len(batch))
	var oldestMicros int64
	for _, ev := range batch {
		if ev.ClientID != "" {
			clientID = ev.ClientID
		}
		events = append(events, mpEvent{Name: ev.Name, Params: ev.Params})
		if micros := rfc3339ToMicros(ev.Timestamp); micros > 0 && (oldestMicros == 0 || micros < oldestMicros) {
			oldestMicros = micros
		}
	}
	body, err := json.Marshal(mpPayload{
		ClientID:           clientID,
		TimestampMicros:    oldestMicros,
		NonPersonalizedAds: true,
		Events:             events,
	})
	if err != nil {
		return nil
	}

	u, err := url.Parse(s.baseURL)
	if err != nil {
		return nil
	}
	q := u.Query()
	q.Set("measurement_id", s.measurementID)
	q.Set("api_secret", s.apiSecret)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "jumpstart-analytics/1")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("ga4 mp: status %d", resp.StatusCode)
	}
	return nil
}

func rfc3339ToMicros(ts string) int64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return t.UnixMicro()
}

func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// keep strconv import used if you stringify debug; otherwise omit strconv.
var _ = strconv.Itoa
```

Remove unused `strconv` if not needed.

Update `build` in `client.go`:

```go
func (c *Client) build(name string, props map[string]any) Event {
	c.globalMu.RLock()
	merged := make(map[string]any, len(c.global)+len(props)+2)
	for k, v := range c.global {
		merged[k] = v
	}
	c.globalMu.RUnlock()
	for k, v := range Sanitize(props) {
		merged[k] = v
	}
	if IsKeyEvent(name, merged) {
		merged["is_key_event"] = true
	}
	return Event{
		Name:      name,
		ClientID:  c.distinct,
		Params:    merged,
		Timestamp: nowTimestamp(),
	}
}
```

(`IsKeyEvent` lands in Task 3 — until then omit the `if IsKeyEvent` block or add a stub `func IsKeyEvent(string, map[string]any) bool { return false }`.)

- [ ] **Step 4: Run all analytics package tests**

Run: `go test ./internal/analytics/ -count=1`  
Expected: PASS (after Task 3 stub or full key_events)

- [ ] **Step 5: Commit** (with Task 1 if not yet committed)

```bash
git add internal/analytics/
git commit -m "$(cat <<'EOF'
feat(analytics): send events via GA4 Measurement Protocol

Replace PostHog batch transport with mp/collect while keeping consent, queue, and redaction.
EOF
)"
```

---

### Task 3: Key-event mapping (`is_key_event`)

**Files:**
- Create: `internal/analytics/key_events.go`
- Create: `internal/analytics/key_events_test.go`
- Modify: `internal/analytics/client.go` (`build` calls `IsKeyEvent`)

**Interfaces:**
- Consumes: event name + merged params
- Produces: `func IsKeyEvent(name string, props map[string]any) bool`

Key events per design:

| name | condition |
|------|-----------|
| `consent_decided` | `props["granted"] == true` |
| `project_created` | always |
| `process_started` | always |
| `processes_accepted` | always |
| `update_installed` | always |

- [ ] **Step 1: Write failing tests**

```go
package analytics

import "testing"

func TestIsKeyEvent(t *testing.T) {
	cases := []struct {
		name  string
		props map[string]any
		want  bool
	}{
		{"project_created", nil, true},
		{"process_started", nil, true},
		{"processes_accepted", nil, true},
		{"update_installed", nil, true},
		{"consent_decided", map[string]any{"granted": true}, true},
		{"consent_decided", map[string]any{"granted": false}, false},
		{"panel_opened", nil, false},
		{"app_launched", nil, false},
	}
	for _, tc := range cases {
		if got := IsKeyEvent(tc.name, tc.props); got != tc.want {
			t.Errorf("IsKeyEvent(%q)=%v want %v", tc.name, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./internal/analytics/ -run TestIsKeyEvent -v`  
Expected: FAIL (`IsKeyEvent` undefined)

- [ ] **Step 3: Implement**

```go
package analytics

// IsKeyEvent reports whether this event should be treated as a GA4 key /
// conversion candidate. Operators still mark key events in GA4 Admin; this
// flag keeps DebugView and explorations consistent.
func IsKeyEvent(name string, props map[string]any) bool {
	switch name {
	case "project_created", "process_started", "processes_accepted", "update_installed":
		return true
	case "consent_decided":
		g, _ := props["granted"].(bool)
		return g
	}
	return false
}
```

Wire into `build` as shown in Task 2.

Optional integration assert in `TestTrackDeliversEventWithGlobalProps` variant: track `project_created` → params `is_key_event==true`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/analytics/ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/analytics/key_events.go internal/analytics/key_events_test.go internal/analytics/client.go
git commit -m "$(cat <<'EOF'
feat(analytics): mark GA4 key-event candidates with is_key_event

EOF
)"
```

---

### Task 4: Global properties cleanup for GA4

**Files:**
- Modify: `internal/analytics/props.go`
- Modify: `internal/analytics/client_test.go` (assert new globals; no `$lib`)

**Interfaces:**
- Consumes: `Options`, first-session bool, session id
- Produces: globals including `app=desktop`, `platform` (= `runtime.GOOS`), existing version/os/arch/locale/session fields; **remove** `$lib` and `$lib_version`

- [ ] **Step 1: Update test expectations**

In `TestTrackDeliversEventWithGlobalProps`, require `app` and `platform`; fail if `$lib` present:

```go
if _, ok := params["$lib"]; ok {
	t.Error("PostHog $lib must not be sent to GA4")
}
if params["app"] != "desktop" {
	t.Errorf("app=%v", params["app"])
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test ./internal/analytics/ -run TestTrackDeliversEventWithGlobalProps -v`

- [ ] **Step 3: Update `globalProps`**

```go
func globalProps(opts Options, first bool, session string) map[string]any {
	return map[string]any{
		"app":              "desktop",
		"platform":         runtime.GOOS,
		"app_version":      version(opts.Version),
		"update_channel":   channel(opts.Channel),
		"os":               runtime.GOOS,
		"os_version":       osVersion(),
		"arch":             runtime.GOARCH,
		"locale":           locale(),
		"install_age_days": installAgeDays(opts.Dir),
		"session_id":       session,
		"is_first_session": first,
	}
}
```

- [ ] **Step 4: Run package tests**

Run: `go test ./internal/analytics/ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/analytics/props.go internal/analytics/client_test.go
git commit -m "$(cat <<'EOF'
refactor(analytics): use GA4-oriented global event parameters

EOF
)"
```

---

### Task 5: Wire main package ldflags

**Files:**
- Modify: `analytics.go` (package main)
- Grep: replace remaining `PostHogAPIKey` / `PostHogHost` references in Go call sites / comments (`analytics_ops.go` comment, etc.)

**Interfaces:**
- Consumes: `analytics.New`, `analytics.Options`
- Produces:
  ```go
  var (
  	GAMeasurementID = ""
  	GAAPISecret     = ""
  )

  func (a *App) initAnalytics() {
  	a.analytics = analytics.New(analytics.Options{
  		MeasurementID: GAMeasurementID,
  		APISecret:     GAAPISecret,
  		Version:       Version,
  		Dir:           analytics.DataDir(),
  	})
  }
  ```

- [ ] **Step 1: Replace ldflag vars and initAnalytics**

Delete `PostHogAPIKey` / `PostHogHost`. Update comments to document:

```text
-ldflags "-X main.GAMeasurementID=G-XXX -X main.GAAPISecret=..."
```

Fix `outcome` comment (“reaching PostHog” → “leaving the machine” / “reaching GA4”).

- [ ] **Step 2: Compile**

Run: `go build -o /dev/null .`  
Expected: success

- [ ] **Step 3: Commit**

```bash
git add analytics.go analytics_ops.go
git commit -m "$(cat <<'EOF'
feat(analytics): stamp GA4 measurement ID and API secret via ldflags

EOF
)"
```

---

### Task 6: GitHub Actions build workflow

**Files:**
- Modify: `.github/workflows/build.yml`

**Interfaces:**
- Consumes: `vars.DESKTOP_GA_MEASUREMENT_ID`, `secrets.DESKTOP_GA_API_SECRET` (secret preferred for API secret)
- Produces: ldflags `-X main.GAMeasurementID=…` `-X main.GAAPISecret=…`
- Removes: `POSTHOG_API_KEY`, `POSTHOG_HOST`, `PostHog*` ldflags

- [ ] **Step 1: Update workflow env + comments + three platform build steps**

```yaml
    # Desktop analytics: GA4 Measurement Protocol credentials stamped into the
    # Go binary (not the frontend). Empty values disable analytics (local/fork).
    #   DESKTOP_GA_MEASUREMENT_ID — repo Variable (G-…)
    #   DESKTOP_GA_API_SECRET     — repo Secret (MP API secret)
    env:
      DESKTOP_GA_MEASUREMENT_ID: ${{ vars.DESKTOP_GA_MEASUREMENT_ID }}
      DESKTOP_GA_API_SECRET: ${{ secrets.DESKTOP_GA_API_SECRET }}
      TAG: ${{ inputs.tag }}
```

Each `wails build` ldflags block:

```text
-X main.GAMeasurementID=$DESKTOP_GA_MEASUREMENT_ID
-X main.GAAPISecret=$DESKTOP_GA_API_SECRET
```

- [ ] **Step 2: Sanity-check YAML**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/build.yml'))"`  
(or `actionlint` if installed). Expected: parse OK.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/build.yml
git commit -m "$(cat <<'EOF'
ci: inject desktop GA4 MP credentials instead of PostHog

EOF
)"
```

---

### Task 7: Frontend bridge comment cleanup

**Files:**
- Modify: `frontend/src/analytics.js`
- Modify: `frontend/.env.example` (remove PostHog / obsolete GA-in-app notes; state Go GA4 MP)

**Interfaces:**
- Consumes: existing `TrackEvent` / `TrackEventOnce` Wails bindings
- Produces: same `capture` / `captureOnce` API; comments say “analytics bridge to Go”, not PostHog

- [ ] **Step 1: Edit comments only (no behavior change)**

- [ ] **Step 2: Commit**

```bash
git add frontend/src/analytics.js frontend/.env.example
git commit -m "$(cat <<'EOF'
docs(frontend): describe analytics bridge without PostHog

EOF
)"
```

---

### Task 8: Remove PostHog scripts and update active docs

**Files:**
- Delete: `scripts/setup-posthog-dashboards.sh`
- Modify: `docs/wiki/Analytics-and-Privacy.md`
- Modify: `docs/wiki/Build-and-Release.md`
- Modify: `docs/wiki/Landing-Site.md` (desktop = GA4 MP; landing = GA4+Clarity; never join)
- Modify: `docs/wiki/Home.md` (table blurb if PostHog-named)
- Modify: `docs/privacy.md`
- Modify: `README.md` (analytics section)
- Modify: `docs/analytics-plan.md` — add status banner: vendor superseded by GA4 MP design; taxonomy still valid
- Modify: `docs/superpowers/plans/2026-08-09-posthog-analytics-wiring.md` — add superseded banner pointing at this plan / GA4 design
- Do **not** rewrite all historical `RELEASE_NOTES.md` changelog entries

**Doc content must include:**

Desktop CI vars:

| Name | Store as | Purpose |
|------|----------|---------|
| `DESKTOP_GA_MEASUREMENT_ID` | Actions Variable | GA4 Measurement ID |
| `DESKTOP_GA_API_SECRET` | Actions Secret | MP API secret |

Landing: `VITE_GA_ID`, `VITE_CLARITY_ID`  
Manual: mark key events in GA4 Admin; delete old `POSTHOG_*` repo vars.

- [ ] **Step 1: Delete script + update docs**

- [ ] **Step 2: Repo search for operational PostHog**

Run:

```bash
rg -n 'posthog|PostHog|POSTHOG' --glob '!RELEASE_NOTES.md' --glob '!docs/superpowers/specs/2026-08-09-posthog*' --glob '!docs/superpowers/plans/2026-08-09-posthog*' .
```

Expected: only historical/superseded banners, or fix stragglers in active code/docs.

- [ ] **Step 3: Commit**

```bash
git add -A scripts/setup-posthog-dashboards.sh docs/ README.md frontend/.env.example
git commit -m "$(cat <<'EOF'
docs: migrate analytics documentation from PostHog to GA4 + Clarity

EOF
)"
```

---

### Task 9: Full verification + final report

**Files:** none required (report in PR/chat)

- [ ] **Step 1: Run Go tests**

Run: `go test ./...`  
Expected: PASS

- [ ] **Step 2: Confirm landing still builds (optional if deps present)**

Run: `cd landing && pnpm install && pnpm run build`  
Expected: PASS (analytics unchanged)

- [ ] **Step 3: Confirm no PostHog packages in lockfiles**

Run: `rg -n 'posthog' frontend/package.json landing/package.json go.mod`  
Expected: no matches

- [ ] **Step 4: Produce final report** (paste into PR / chat) covering design § “Final report” items 1–16, including:

- WordShield: N/A  
- Vercel: manual verify `VITE_GA_ID` / `VITE_CLARITY_ID`  
- Operator must set `DESKTOP_GA_*` and mark GA4 key events  
- Do not invent IDs  

- [ ] **Step 5: Commit** only if verification fixed anything; otherwise stop.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| GA4 MP from Go | 2, 5 |
| No desktop Clarity | Global + docs Task 8 |
| Dual ldflag credentials | 1, 5, 6 |
| Keep taxonomy + key events | 3 |
| Keep consent/queue/redact | 1–2 (unchanged paths) |
| Landing `VITE_*` unchanged | 8 (docs only) |
| Non-prod no-op | 1 (`Configured`) |
| Remove PostHog CI/docs/script | 6, 8 |
| Final report | 9 |
| WordShield N/A | 8, 9 |
| No invented IDs | Global |

## Placeholder / consistency self-check

- Options fields: `MeasurementID`, `APISecret`, `CollectURL` — consistent across Tasks 1–5.
- ldflag symbols: `main.GAMeasurementID`, `main.GAAPISecret` — consistent across Tasks 5–6.
- Event JSON fields: `name`, `client_id`, `params`, `timestamp` — transport + queue + tests.
- `IsKeyEvent` signature stable between Tasks 2–3.
