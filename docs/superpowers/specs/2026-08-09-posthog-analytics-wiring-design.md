# PostHog analytics wiring and completion

Status: approved design  
Date: 2026-08-09  
Depends on: existing `internal/analytics` (shipped in v1.3.0), `docs/analytics-plan.md`

## Goal

Finish product analytics for the JumpStart Wails desktop app:

1. Wire the PostHog Cloud (US) project API key into release CI so product builds actually ingest.
2. Fill the remaining event gaps from the analytics plan.
3. Create the six plan §7 dashboards via a one-shot, reproducible script (personal API key).

Local and fork builds stay no-op when `PostHogAPIKey` is empty.

## Non-goals

- Landing-site GA4 / Clarity / Vercel Analytics (already separate; never join datasets).
- Sampling, frontend PostHog JS SDK, or changing consent defaults.
- Recreating dashboards on every CI release.
- Putting a personal API key in the repo or in GitHub Actions.

## Context (what already exists)

- Go package `internal/analytics`: consent, identity, redaction, offline NDJSON queue, HTTP transport to PostHog.
- App helpers: `analytics.go`, `analytics_lifecycle.go`, `analytics_ops.go`, `analytics_kanban.go`.
- Frontend bridge: `frontend/src/analytics.js` → `TrackEvent` / `TrackEventOnce`.
- CI: `.github/workflows/build.yml` already injects `-X main.PostHogAPIKey=$POSTHOG_API_KEY` and `-X main.PostHogHost=$POSTHOG_HOST`.
- Most of the plan taxonomy is already instrumented (lifecycle, onboarding, process loop, git/docker, AI, updates, banners, kanban).

## Approach

**CI key + event gaps in code + one-shot dashboard script** (not hand-only dashboards, not per-release dashboard automation).

---

## 1. CI wiring

### Actions

- Set repository variable `POSTHOG_API_KEY` to the PostHog **project** API key (`phc_…`).
- Leave `POSTHOG_HOST` unset so ingestion uses PostHog Cloud US (`https://us.i.posthog.com` / empty host → `DefaultHost` in `internal/analytics`).
- No change required to `build.yml` unless verification finds a quoting/env bug.

### Notes

- A PostHog project key is write-only and intentionally shippable in the binary (same pattern as documented in README / wiki).
- Store it as a GitHub **variable**, not a secret, matching existing workflow comments.
- Forks and local `wails build` without ldflags remain analytics no-ops.

### Smoke test

1. Local: `wails build` (or `wails dev` equivalent ldflags) with `-X main.PostHogAPIKey=<project key>`.
2. Launch the app with analytics enabled (default on in product builds; Settings → Privacy if toggled off).
3. Confirm `app_launched` appears in PostHog Live events within ~1 minute (batching/flush may delay slightly; shutdown flush should also deliver).

---

## 2. Event gaps

Follow existing patterns: `a.track` / `a.trackOnce` on Go; `track` / `trackOnce` / `trackPanel` on the frontend; never paths, names, env keys/values, or free text. All props still pass `internal/analytics/redact.go`.

| Event | Call site | Properties | Dedup / volume |
|---|---|---|---|
| `logs_opened` | When `LogPanel` mounts (process logs, script run log, or test log) | `line_count` (from current buffer / `GetLogs`), `source` ∈ {`process`, `script`, `test`} | `trackOnce` per `(source, procId)` per session |
| `ports_viewed` | First entry into Ports view (`PortsView` / App navigate to `ports`) | `port_count`, `conflict_count` (ports appearing on ≥2 processes in the current snapshot) | Once per session; **never** on the 3s `GetPortMap` poll |
| `env_file_edited` | Go `trackProjectSaved` when a process env map changes vs previous project | `var_count` (after), `added`, `removed` (key-set diff only) | Skip when env maps are unchanged |
| `roadmap_opened` | `TaskTracker` when the roadmap modal opens | `item_count` = number of sprints | Once per open is fine; low volume |

### Detection acceptance (no new funnel event)

Keep the existing frontend event `processes_accepted` with `detected_count`, `accepted_count`, and `env_prompt_count`. Document it as the join to Go’s `processes_detected` (same session / install). Do **not** add a second acceptance event or retrofit `accepted_count` onto `processes_detected` (those are separate round trips).

### Tests / docs

- Extend redact/enum tests if `source` needs an explicit allowlist helper.
- Update `docs/wiki/Analytics-and-Privacy.md` (and privacy.md only if the public description of collection changes — these events are already covered by the plan’s shape).

---

## 3. Dashboard script

### Location

`scripts/setup-posthog-dashboards.sh` (optional small JSON sibling under `scripts/posthog/` for insight query bodies).

### Auth (operator machine only)

| Env var | Purpose |
|---|---|
| `POSTHOG_PERSONAL_API_KEY` | Bearer token with `dashboard:write` and `insight:write` |
| `POSTHOG_PROJECT_ID` | Numeric project id |
| `POSTHOG_HOST` | Optional; default `https://us.posthog.com` for API (management host, not the ingestion `i.posthog.com` host) |

Never commit these. Never put the personal key in GitHub Actions.

### Behaviour

- Create (or skip if a dashboard with the same name already exists) six dashboards matching plan §7:

  1. **Health** — DAU/WAU/MAU from `app_launched`; session duration from `app_closed`; `app_crashed` rate by `app_version`.
  2. **Activation** — funnel: `app_launched` → `project_created` → `process_added` → `process_started`.
  3. **Feature adoption** — reach of `panel_opened` (breakdown by `panel`) and key feature events among users with `app_launched`.
  4. **Reliability** — `process_started` success rate by `runtime`; top `failure_reason` values.
  5. **AI value** — `ai_suggestion_accepted` / enrich acceptance rates; `ai_chat_message_sent` volume; correlation with `used_code_context` where available.
  6. **Version fragmentation** — `app_version` distribution over time on `app_launched`.

- Prefer PostHog Insights API (`InsightVizNode` trends/funnels) over fragile HogQL where the stock insight types suffice.
- Idempotent by dashboard **name**: if present, print skip; do not delete or overwrite hand-edited tiles on re-run (document that re-creation requires deleting the dashboard first).
- Print resulting dashboard URLs on success.

### Billing alert

Document setting a PostHog billing/usage alert near **700k events/month** (plan guardrail). Automate only if a stable API exists and is low-cost to maintain; otherwise a checklist step in the wiki is enough.

---

## 4. Documentation updates

- `docs/wiki/Analytics-and-Privacy.md` — CI variable, smoke test, dashboard script usage, `processes_accepted` as the acceptance join, new events.
- `docs/wiki/Build-and-Release.md` — confirm `POSTHOG_API_KEY` must be set for release builds to emit.
- `README.md` — only if the existing analytics paragraph needs a one-line pointer to the script (keep short).

---

## 5. Error handling and privacy

- Analytics must never break UI or bindings: frontend `safe()` continue; Go `Track` non-blocking.
- Turning analytics off still purges `analytics_queue.ndjson`.
- New properties: counts, booleans, short enums only.
- `conflict_count` and `line_count` are integers derived from in-memory snapshots — no hostnames, paths, or process names.

---

## 6. Verification

1. Unit tests for any new redact/enum helpers and env key-set diff logic.
2. Local smoke build with project key → Live event `app_launched`.
3. Exercise Ports, Logs, env edit, roadmap → confirm new event names in Live view.
4. Run dashboard script once → six dashboards visible in PostHog project.
5. Confirm GitHub variable is set: `gh variable list` shows `POSTHOG_API_KEY`.

---

## 7. Rollout order

1. Set CI variable (enables next tagged release).
2. Land event-gap PR; smoke-test locally with ldflags.
3. Run dashboard script after first live events exist (insights need events to look meaningful, but empty dashboards are fine to create first).
4. Optionally cut a patch release so distributed binaries include the new events **and** the key.

---

## Open decisions (resolved)

| Decision | Choice |
|---|---|
| Region | US (empty `POSTHOG_HOST`) |
| Scope | CI + all four event gaps + six dashboards |
| Dashboard creation | One-shot script with personal API key |
| Acceptance signal | Keep `processes_accepted`; do not mutate `processes_detected` |
