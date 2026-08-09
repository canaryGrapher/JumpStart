# Analytics and Privacy

Canonical user-facing policy: [`docs/privacy.md`](https://github.com/canaryGrapher/JumpStart/blob/main/docs/privacy.md).

## Design decisions

1. **Ingest from Go**, not the webview. Wails is not a browser; a JS SDK that talked to PostHog directly would need CSP exemptions, miss `Startup`/`Shutdown`, and fail offline.
2. **Consent in `settings.json`**, not `config.json` (projects are a bare array).
3. **On by default** in product builds (master + all categories); **no-op** when `PostHogAPIKey` is empty (local/fork builds).
4. **Sanitize at the boundary** (`internal/analytics/redact.go`) so one bad call site cannot leak paths or free text.
5. **Volume control** is category / detail-level gating — not random sampling (funnels stay valid on Full and Balanced).

## GitHub Actions variables (required for release builds)

| Variable | Required | Notes |
|----------|----------|-------|
| `POSTHOG_API_KEY` | **Yes** | PostHog **project** key (`phc_…`). Repo **variable**, not secret. |
| `POSTHOG_HOST` | No | Leave unset for US (`https://us.i.posthog.com`). |

```sh
gh variable set POSTHOG_API_KEY --body 'phc_…'
```

Without the variable, release binaries compile but analytics is a no-op.

## Package layout (`internal/analytics`)

| File | Role |
|------|------|
| `client.go` | Track, batch, flush, category gate |
| `category.go` | Event→category map, presets, `EnvKeyDiff` |
| `consent.go` | Load/save enabled + detailLevel + categories |
| `identity.go` | install UUID + HMAC `project_ref` |
| `redact.go` | Property sanitiser |
| `enums.go` | Bounded failure reasons, runtimes, model family parse |
| `queue.go` | Offline NDJSON queue |
| `transport.go` | HTTP to PostHog |
| `props.go` | Global properties |
| `osinfo_*.go` | OS version strings |

## Categories and detail levels

Settings → Privacy:

- Master: Share anonymous usage data
- Detail level: Full | Balanced | Minimal (Custom when toggles diverge)
- Per-category toggles (all on under Full): lifecycle, onboarding, processes, git_docker, kanban, ai, updates, ui_panels

| Preset | Off |
|--------|-----|
| Full | — |
| Balanced | `ui_panels` |
| Minimal | everything except `lifecycle` + `updates` |

`processes_accepted` (frontend) joins to `processes_detected` (Go) for detection quality.

## App wiring

- ldflags: `main.PostHogAPIKey`, `main.PostHogHost` (see Build-and-Release)
- Helpers in `analytics.go`, `analytics_ops.go`, `analytics_kanban.go`, `analytics_lifecycle.go`
- Frontend bridge: `frontend/src/analytics.js` — PostHog-shaped `capture` / `optIn` / … that only call Go bindings (no `posthog-js`)

## Dashboards

One-shot script (personal API key on your machine only):

```sh
export POSTHOG_PERSONAL_API_KEY=phx_…
export POSTHOG_PROJECT_ID=…
./scripts/setup-posthog-dashboards.sh
```

Creates six dashboards (skip if name exists). Set a billing alert near 700k events/month.

## Rules for new events

1. Name: `object_verb_past_tense` snake_case (e.g. `process_started`).
2. Props: counts, booleans, short enums — **never** paths, names, commands, messages, tokens.
3. Fallible ops: use `outcome(start, err, extra)` so `failure_reason` is always mapped.
4. Reach metrics: `captureOnce` / `TrackEventOnce` (e.g. `panel_opened`).
5. Register the event in `internal/analytics/category.go` (`eventCategory` map).
6. Add/adjust tests in `redact_test.go` / `category_test.go` / `enums_test.go` when introducing new string shapes.
7. Update `docs/privacy.md` if the public description of collection changes.

## Desktop vs landing

| Surface | Tools |
|---------|-------|
| Desktop app | PostHog via Go only |
| Landing site | GA4, Clarity, Vercel Analytics |

Datasets are **never joined**.

## Turning off

Settings → Privacy → master toggle off. Stops collection and deletes `analytics_queue.ndjson`. Category toggles only drop future events in that bucket.
