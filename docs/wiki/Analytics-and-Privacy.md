# Analytics and Privacy

Canonical user-facing policy: [`docs/privacy.md`](https://github.com/canaryGrapher/JumpStart/blob/main/docs/privacy.md).

## Design decisions

1. **Ingest from Go**, not the webview. Wails is not a browser; a JS SDK would need CSP exemptions, miss `Startup`/`Shutdown`, and fail offline.
2. **Consent in `settings.json`**, not `config.json` (projects are a bare array).
3. **On by default** in product builds; **no-op** when `PostHogAPIKey` is empty (local/fork builds).
4. **Sanitize at the boundary** (`internal/analytics/redact.go`) so one bad call site cannot leak paths or free text.

## Package layout (`internal/analytics`)

| File | Role |
|------|------|
| `client.go` | Track, batch, flush |
| `consent.go` | Load/save enabled flag |
| `identity.go` | install UUID + HMAC `project_ref` |
| `redact.go` | Property sanitiser |
| `enums.go` | Bounded failure reasons, runtimes, model family parse |
| `queue.go` | Offline NDJSON queue |
| `transport.go` | HTTP to PostHog |
| `props.go` | Global properties |
| `osinfo_*.go` | OS version strings |

## App wiring

- ldflags: `main.PostHogAPIKey`, `main.PostHogHost` (see Build-and-Release)
- Helpers in `analytics.go`, `analytics_ops.go`, `analytics_kanban.go`, `analytics_lifecycle.go`
- Frontend bridge: `frontend/src/analytics.js` → `TrackEvent*` bindings

## Rules for new events

1. Name: `object_verb_past_tense` snake_case (e.g. `process_started`).
2. Props: counts, booleans, short enums — **never** paths, names, commands, messages, tokens.
3. Fallible ops: use `outcome(start, err, extra)` so `failure_reason` is always mapped.
4. Reach metrics: `trackOnce` / `TrackEventOnce` (e.g. `panel_opened`).
5. Add/adjust tests in `redact_test.go` / `enums_test.go` when introducing new string shapes.
6. Update `docs/privacy.md` if the public description of collection changes.

## Desktop vs landing

| Surface | Tools |
|---------|-------|
| Desktop app | PostHog via Go only |
| Landing site | GA4, Clarity, Vercel Analytics |

Datasets are **never joined**.

## Turning off

Settings → Privacy → toggle. Disabling stops collection and deletes `analytics_queue.ndjson`.
