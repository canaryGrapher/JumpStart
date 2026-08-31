# PostHog analytics wiring and completion

> **Superseded (desktop ingest, 2026-08-09):** PostHog Cloud replaced by GA4 Measurement Protocol. See `docs/superpowers/specs/2026-08-09-ga4-clarity-migration-design.md`. This document remains historical reference for consent, categories, and frontend bridge design.

Status: approved design (revised)  
Date: 2026-08-09  
Depends on: existing `internal/analytics` (shipped in v1.3.0), `docs/analytics-plan.md`

## Goal

Finish and extend product analytics for the JumpStart Wails desktop app:

1. Wire the PostHog Cloud (US) project API key into release CI so product builds ingest.
2. Fill remaining event gaps from the analytics plan.
3. Create the six plan §7 dashboards via a one-shot script (personal API key).
4. Add Settings controls for analytics detail level and per-category toggles (all on by default).
5. Expose a PostHog-shaped frontend API that still sends only through Go (no webview network).

Local and fork builds stay no-op when `PostHogAPIKey` is empty.

## Non-goals

- Landing-site GA4 / Clarity / Vercel Analytics (already separate; never join datasets).
- Installing `posthog-js` or any SDK that talks to PostHog directly from the webview.
- Random event sampling (funnels break); volume control is category/detail-level gating only.
- Recreating dashboards on every CI release.
- Putting a personal API key in the repo or in GitHub Actions.
- Changing the master default off — usage sharing stays **on by default** when the build is configured.

## Context (what already exists)

- Go package `internal/analytics`: consent, identity, redaction, offline NDJSON queue, HTTP transport.
- App helpers: `analytics.go`, `analytics_lifecycle.go`, `analytics_ops.go`, `analytics_kanban.go`.
- Frontend bridge: `frontend/src/analytics.js` → `TrackEvent` / `TrackEventOnce`.
- Privacy UI: `frontend/src/components/PrivacySettings.jsx` (single master switch).
- Consent file: `~/.jumpstart/settings.json` → `{ "analytics": { "enabled", "decidedAt" } }`.
- CI: `.github/workflows/build.yml` injects `-X main.PostHogAPIKey=$POSTHOG_API_KEY` and `-X main.PostHogHost=$POSTHOG_HOST`.
- Most of the plan taxonomy is already instrumented.

## Approach

CI key + event gaps + one-shot dashboard script + category/detail Settings + Go-backed PostHog-shaped frontend API.

---

## 1. CI wiring — what to add in GitHub

### You must add (operator steps)

In the JumpStart GitHub repo: **Settings → Secrets and variables → Actions → Variables**:

| Variable | Value | Required |
|---|---|---|
| `POSTHOG_API_KEY` | PostHog **project** API key (`phc_…`) | **Yes** |
| `POSTHOG_HOST` | Leave unset for US, or `https://us.i.posthog.com` | No |

CLI equivalent:

```bash
gh variable set POSTHOG_API_KEY --body 'phc_…'
# do not set POSTHOG_HOST for US default
```

Use a **variable**, not a secret: the project key is write-only and is intentionally stamped into release binaries via ldflags (same as README / wiki).

Without `POSTHOG_API_KEY`, release builds compile but analytics is an unconditional no-op (`Configured() == false`).

### Workflow / docs changes

- Keep existing `build.yml` env + ldflags (already correct).
- Add a short comment block (or wiki table) listing required Actions variables so the next releaser does not miss them.
- No personal API key in Actions.

### Smoke test

1. Local build with `-X main.PostHogAPIKey=<project key>`.
2. Launch with analytics enabled.
3. Confirm `app_launched` in PostHog Live events (allow for batching; shutdown flush also delivers).

---

## 2. Event gaps

Follow existing patterns: `a.track` / `a.trackOnce` on Go; frontend `capture` / `captureOnce`; never paths, names, env keys/values, or free text. All props still pass `internal/analytics/redact.go`. Category gating applies after consent, before queue.

| Event | Call site | Properties | Dedup / volume |
|---|---|---|---|
| `logs_opened` | `LogPanel` mount | `line_count`, `source` ∈ {`process`, `script`, `test`} | `trackOnce` per `(source, procId)` per session |
| `ports_viewed` | First entry into Ports view | `port_count`, `conflict_count` | Once per session; never on the 3s poll |
| `env_file_edited` | Go `trackProjectSaved` when process env maps change | `var_count`, `added`, `removed` (key-set diff) | Skip when unchanged |
| `roadmap_opened` | `TaskTracker` when roadmap modal opens | `item_count` = sprint count | Low volume |

### Detection acceptance

Keep frontend `processes_accepted` (`detected_count`, `accepted_count`, `env_prompt_count`) as the join to Go `processes_detected`. Do not retrofit `accepted_count` onto `processes_detected`.

---

## 3. Category gating and detail level (“sampling”)

No random sampling. Volume and sensitivity are controlled by a master switch, a detail-level preset, and per-category toggles. **All categories default on** (Full).

### Storage (`~/.jumpstart/settings.json`)

```json
{
  "analytics": {
    "enabled": true,
    "decidedAt": 0,
    "detailLevel": "full",
    "categories": {
      "lifecycle": true,
      "onboarding": true,
      "processes": true,
      "git_docker": true,
      "kanban": true,
      "ai": true,
      "updates": true,
      "ui_panels": true
    }
  }
}
```

Missing file → same defaults as above (`enabled: true`, `detailLevel: "full"`, all categories `true`).

### Presets

| Preset | Categories on | Categories off |
|---|---|---|
| `full` | all eight | — |
| `balanced` | lifecycle, onboarding, processes, git_docker, kanban, ai, updates | ui_panels |
| `minimal` | lifecycle, updates | onboarding, processes, git_docker, kanban, ai, ui_panels |
| `custom` | user-defined | user-defined |

Selecting a preset overwrites the category map. Flipping any category after a preset sets `detailLevel` to `custom`.

### Event → category map (representative)

| Category | Example events |
|---|---|
| `lifecycle` | `app_launched`, `app_closed`, `app_crashed`, `consent_decided` |
| `onboarding` | `project_created`, `process_added`, `processes_detected`, `processes_accepted`, `config_imported` |
| `processes` | `process_started`, `process_stopped`, `process_crashed`, `deps_installed`, `logs_opened`, `ports_viewed`, `env_file_edited`, `script_run`, `tests_run`, `external_open_performed` |
| `git_docker` | `git_action_performed`, `git_token_saved`, `docker_action_performed`, `release_created` |
| `kanban` | `task_created`, `task_moved`, `sprint_created`, `roadmap_opened` |
| `ai` | `ollama_detected`, `ai_*`, `code_context_*` |
| `updates` | `update_*`, `banner_*` |
| `ui_panels` | `panel_opened`, `theme_changed`, `accent_changed` |

Unmapped event names default to `lifecycle` only if they are crash/consent-adjacent; otherwise treat as `ui_panels` or reject in review — implementation must maintain an explicit map in `internal/analytics` (e.g. `category.go`) so gating cannot silently miss new events.

### Gate order in `Track`

1. Nil client / empty API key → no-op.
2. Master `enabled == false` → no-op (and purge queue on disable, existing behaviour).
3. Resolve event category; if that category is false → no-op.
4. Redact → queue/send as today.

`consent_decided` is always allowed when turning **off** (existing ordering); when turning categories off, no special event required beyond optional future `analytics_prefs_changed` (out of scope unless cheap).

---

## 4. Settings UI

Expand `PrivacySettings.jsx` (Preferences → Privacy), not a new top-level app tab:

1. Master: “Share anonymous usage data” (existing).
2. Detail level: segmented control Full | Balanced | Minimal (show Custom when state does not match a preset).
3. Category switches for the eight categories; disabled when master is off.
4. Existing unconfigured-build hint when `Configured() == false`.
5. Link to privacy policy (existing).

New Wails bindings as needed: `GetAnalyticsSettings` returns enabled/configured/detailLevel/categories; `SetAnalyticsEnabled`; `SetAnalyticsDetailLevel`; `SetAnalyticsCategories` (or one `SetAnalyticsPrefs` struct). Persist via extended `ConsentState` / `SaveConsent` (rename conceptually to analytics settings save; keep file shape backward compatible).

---

## 5. Frontend PostHog-shaped API (option C)

Extend `frontend/src/analytics.js` (or split `posthogBridge.js`) to expose a small PostHog-like surface:

- `capture(event, props)` → `TrackEvent`
- `captureOnce(key, event, props)` → `TrackEventOnce`
- `optIn()` / `optOut()` → `SetAnalyticsEnabled`
- `setDetailLevel(level)` / `setCategories(map)` → new bindings
- `getSettings()` → `GetAnalyticsSettings`

Rules:

- Do **not** add `posthog-js` or open network from the webview.
- Do **not** invent a second `distinct_id`; identity stays in Go.
- Existing helpers (`track`, `trackPanel`, …) become thin wrappers around `capture` / `captureOnce` for compatibility.

---

## 6. Dashboard script

`scripts/setup-posthog-dashboards.sh` (+ optional JSON under `scripts/posthog/`).

Auth on the operator machine only:

| Env var | Purpose |
|---|---|
| `POSTHOG_PERSONAL_API_KEY` | Bearer with `dashboard:write`, `insight:write` |
| `POSTHOG_PROJECT_ID` | Numeric project id |
| `POSTHOG_HOST` | Optional management host; default `https://us.posthog.com` |

Creates or skips by name the six dashboards:

1. Health  
2. Activation funnel  
3. Feature adoption  
4. Reliability  
5. AI value  
6. Version fragmentation  

Idempotent by dashboard name (skip if exists; document delete-to-recreate). Print URLs on success. Billing alert ~700k events/month: wiki checklist unless a stable API is trivial.

---

## 7. Documentation updates

- `docs/wiki/Analytics-and-Privacy.md` — CI variables, category map, Settings UI, frontend bridge, dashboard script, `processes_accepted`.
- `docs/wiki/Build-and-Release.md` — `POSTHOG_API_KEY` required for emitting releases.
- `docs/privacy.md` — mention category toggles / detail level; still on by default; still no sensitive payloads.
- This spec remains the implementation source of truth until the wiki is updated.

---

## 8. Error handling and privacy

- Analytics never breaks UI or bindings.
- Disable master → purge queue (existing).
- Category off → drop only; do not purge historical queue entries already written (optional: purge is master-only to avoid surprise data loss when trimming categories).
- Properties: counts, booleans, short enums only.

---

## 9. Verification

1. Unit tests: category map coverage for known events; preset expand/collapse; env key-set diff; redact unchanged.
2. Local smoke with project key → `app_launched`.
3. Toggle Minimal → confirm AI/process events stop; lifecycle still fires.
4. Ports / logs / env / roadmap → new events in Live view (Full).
5. `gh variable list` shows `POSTHOG_API_KEY`.
6. Dashboard script creates six dashboards.

---

## 10. Rollout order

1. Set GitHub Actions variable `POSTHOG_API_KEY`.
2. Land prefs + category gate + frontend API + event gaps.
3. Smoke-test locally with ldflags.
4. Run dashboard script.
5. Cut a release so distributed binaries include key + new controls/events.

---

## Resolved decisions

| Decision | Choice |
|---|---|
| Region | US (empty `POSTHOG_HOST`) |
| Event gaps + six dashboards | Yes |
| Frontend “SDK” | C — PostHog-shaped API via Go only |
| Sampling | D — detail level + category toggles, no random drops |
| Category UI | D — presets and per-category overrides |
| Master / categories default | All enabled (`full`) |
| Acceptance signal | Keep `processes_accepted` |
| GitHub storage for project key | Actions **variable** `POSTHOG_API_KEY` |
