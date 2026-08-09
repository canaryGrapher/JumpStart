# GA4 + Clarity analytics migration (desktop PostHog removal)

**Status:** Approved design (2026-08-09). Awaiting implementation plan after user review of this file.

**Supersedes (desktop ingest):** PostHog Cloud as the desktop product-analytics backend (`docs/superpowers/specs/2026-08-09-posthog-analytics-wiring-design.md` remains historical for the consent/category/UI work that still applies).

**Goal:** Remove PostHog from JumpStart’s analytics stack. Use Google Analytics 4 for structured product analytics on desktop (via Measurement Protocol from Go) and keep Microsoft Clarity (plus GA4 gtag) on the landing site for web behavioral/session analysis. Do not invent, reuse, or substitute analytics IDs — the operator supplies them separately.

---

## Decisions locked

| Topic | Choice |
|-------|--------|
| Desktop structured analytics | **GA4 Measurement Protocol from Go** (not gtag in the Wails webview) |
| Desktop Clarity | **None** — Clarity is landing-only |
| Desktop credentials | `DESKTOP_GA_MEASUREMENT_ID` + `DESKTOP_GA_API_SECRET` stamped via ldflags |
| Event taxonomy | **Hybrid** — keep JumpStart event names; add GA4 key-event / conversion mapping |
| Landing env names | Keep `VITE_GA_ID` / `VITE_CLARITY_ID` |
| Non-production traffic | Hard no-op unless IDs are stamped/set; do not send local/fork traffic to prod by default |
| Implementation shape | **Transport swap** inside `internal/analytics` — keep consent, categories, redaction, offline queue |

---

## Architecture

```
Desktop (Wails / Go)                    Landing (Vercel)
─────────────────────                   ─────────────────
App helpers + frontend bridge           landing/src/analytics.js
        ↓                               GA4 gtag.js + Clarity + Vercel Analytics
internal/analytics                      Env: VITE_GA_ID, VITE_CLARITY_ID
  Track / redact / consent
  category + detail gating
  offline NDJSON queue
        ↓
GA4 Measurement Protocol
  measurement_id + api_secret (ldflags)
        ↓
Google Analytics 4 (desktop property)

Clarity: landing only
Datasets: desktop GA4 ≠ landing GA4/Clarity (never join)
WordShield: N/A (not in this repository)
```

### Why Go MP for desktop

Wails ships a webview, not a full browser. Prior GA4/Clarity-in-webview work was replaced by Go-side ingest because:

- CSP / SDK friction in the webview
- Missed `Startup` / `Shutdown` and cold-start metrics
- No reliable offline delivery from JS alone
- Clarity session recording is a poor fit for embedded webviews

GA4 Measurement Protocol preserves the Go boundary while changing the vendor wire format.

### GA4 vs Clarity responsibilities

| Tool | Answers | Used where |
|------|---------|------------|
| **GA4** | What happened? Counts, funnels, retention-oriented metrics, feature usage | Desktop (MP) + Landing (gtag) |
| **Clarity** | Why did it happen? Recordings, heatmaps, rage/dead clicks | Landing only |

Do not treat them as interchangeable. Do not duplicate every GA4 event into Clarity beyond useful tags/context.

---

## Desktop: GA4 Measurement Protocol

### Credentials (release builds)

| GitHub Actions variable | ldflag | Notes |
|-------------------------|--------|-------|
| `DESKTOP_GA_MEASUREMENT_ID` | `-X main.GAMeasurementID=…` | GA4 Measurement ID (`G-…`) |
| `DESKTOP_GA_API_SECRET` | `-X main.GAAPISecret=…` | MP API secret from GA4 Admin → Data streams → Measurement Protocol |

Empty either value → analytics client is a full no-op (same pattern as empty PostHog key today).

Trust model: API secret ships in the release binary (write-oriented ingest credential), analogous to the previous write-only PostHog project key.

### Replace, keep, remove

**Replace**

- `internal/analytics/transport.go` PostHog `/batch/` payload → GA4 MP collect endpoint payload
- Wire `Event` / queue serialization to GA4 MP event shape (or map at send time from an internal queued form)
- `main.PostHogAPIKey` / `main.PostHogHost` → `main.GAMeasurementID` / `main.GAAPISecret`
- `.github/workflows/build.yml` env + ldflags
- Docs: wiki Analytics-and-Privacy, Build-and-Release, privacy.md, README, Landing-Site notes, analytics-plan status note

**Keep**

- Consent master toggle + detail levels + category toggles (`consent.go`, `category.go`, Privacy Settings UI)
- Redaction (`redact.go`), enums (`enums.go`), identity (`identity.go`), props (`props.go`)
- Offline NDJSON queue + flush-on-shutdown
- Frontend bridge `frontend/src/analytics.js` (`capture` / `captureOnce` / prefs) calling Go only — rename comments from “PostHog-shaped” to neutral “analytics bridge”
- Call-site helpers in `analytics.go`, `analytics_ops.go`, `analytics_kanban.go`, `analytics_lifecycle.go`

**Remove**

- PostHog host/key concepts, PostHog wire comments that imply the vendor
- `scripts/setup-posthog-dashboards.sh`
- `POSTHOG_API_KEY` / `POSTHOG_HOST` CI variables (document removal; operator deletes from GitHub)
- PostHog-centric dashboard instructions
- Any remaining PostHog package references (there is no `posthog-go` / `posthog-js` today — keep it that way)

### Identity and privacy

- `distinct_id` / install UUID continues as the anonymous client identity; map to GA4 `client_id` (stable UUID string without PII).
- Do **not** send passwords, tokens, API keys, emails, paths, project names, prompts, chat text, stack traces, or raw error strings.
- Continue sanitizing at the boundary via `redact.go`.
- Optional: set GA4 `user_id` only if a future authenticated product identity exists; JumpStart has none today — **do not invent one**.

### Session and globals

Keep attaching global properties where they fit GA4 event params (or user properties if clearly justified):

- `app_version`, `os`, `os_version`, `arch`, `locale`, `update_channel`, `install_age_days`, `session_id`, `is_first_session`, `app=desktop`, `platform` / `environment` as appropriate

Prefer event **parameters** over embedding meaning in event names.

---

## Event taxonomy (hybrid)

### Source of truth

Keep JumpStart product event names (`object_verb_past_tense`, snake_case) and existing parameters. Canonical lists live in:

- `internal/analytics/category.go` (`eventCategory` map)
- `docs/analytics-plan.md` (historical plan; update vendor sections during implementation)

### Categories (unchanged)

`lifecycle`, `onboarding`, `processes`, `git_docker`, `kanban`, `ai`, `updates`, `ui_panels`

### Intentionally not added

Events that do not apply to JumpStart today:

- `user_signed_up`, `user_logged_in`, `user_logged_out`
- `subscription_started`, `subscription_cancelled`, `subscription_upgraded`
- Generic UI noise (`button_clicked`, `div_clicked`, `component_rendered`, `modal_opened`)

### GA4 key events / conversions

Do **not** mark every event as a conversion. Document and configure in GA4 Admin (manual step) these key events:

| Event | Condition / notes | Product question |
|-------|-------------------|------------------|
| `consent_decided` | Prefer when `granted=true` | Opt-in rate |
| `project_created` | Activation | First project |
| `process_started` | Core loop | First successful run |
| `processes_accepted` | Detection quality → acceptance | Onboarding friction |
| `update_installed` | Upgrade success | Version adoption |

Implementation support: a small `key_events.go` (or equivalent) that can attach a stable param such as `is_key_event=true` for the subset above so DebugView and explorations stay consistent. Actual “Key event” toggles in GA4 Admin remain an operator step.

### Funnels worth answering after migration

- How many installs grant consent?
- How many create a project / start a process?
- Which features (`panel_opened`, AI, git/docker) are used?
- Return after 1/7/30 days (GA4 retention using `client_id`)
- Which `app_version` is active?
- Where onboarding drops (`processes_detected` with `accepted_count: 0`, failed `process_started`)

### Errors / failures

Keep bounded `failure_reason` / `succeeded` style params. Do not send stack traces or bodies to GA4. Crashes: keep existing panic recover → `app_crashed` with non-sensitive classification only; do not turn GA4 into a log sink.

### Events intentionally removed from vendor surface

None of the product taxonomy is deleted solely for this migration. Removed **vendor** surface:

- PostHog Live / dashboards / management API scripts
- Any docs instructing engineers to verify in PostHog

If audit finds duplicate or meaningless UI events, trim in a follow-up — not required for transport migration.

---

## Landing site (web)

Already implemented: `landing/src/analytics.js` with GA4 + Clarity + Vercel Analytics.

| Concern | Action |
|---------|--------|
| Env names | **Keep** `VITE_GA_ID`, `VITE_CLARITY_ID` |
| Separation | Must never use desktop GA credentials |
| Clarity tags | Keep useful `clarity("set", …)` from track params; Clarity answers UX “why” |
| Taxonomy | Landing keeps marketing/download-oriented events; do not merge with desktop event names into one property |
| Vercel | Inspect Production / Preview / Development; ensure `VITE_*` set only where intentional; Preview should not silently share prod IDs unless desired (default: only set where operator intends) |
| WordShield | **N/A** — not part of this repo |

No requirement to introduce `WEB_GA_MEASUREMENT_ID` / `WEB_CLARITY_ID` symbol names in code.

---

## Abstraction

Do **not** over-engineer a new `analytics/{ga,clarity,events}` tree.

Practical abstraction:

- Go: `Client.Track(name, props)` remains the app-facing API
- JS desktop: `capture` / `captureOnce` bridge
- JS landing: existing `track` / `trackPageView`

Provider swap happens behind `transport.go` (+ client init). Changing vendors later should not require rewriting every call site.

---

## CI / deployment

### GitHub Actions (`build.yml`)

**Add**

- `DESKTOP_GA_MEASUREMENT_ID` from `vars` (or secrets — prefer vars for Measurement ID; API secret may be `secrets` if desired; either is acceptable if documented consistently)
- `DESKTOP_GA_API_SECRET`
- ldflags: `main.GAMeasurementID`, `main.GAAPISecret`

**Remove**

- `POSTHOG_API_KEY`, `POSTHOG_HOST`
- `main.PostHogAPIKey`, `main.PostHogHost`

Desktop workflows must not receive or stamp landing Vite IDs. Landing builds must not receive desktop GA secrets.

### Vercel (landing)

- Find existing `VITE_GA_ID`, `VITE_CLARITY_ID` (and any PostHog leftovers — remove if present)
- Production / Preview / Development configured intentionally
- Do not commit real credentials to the repo

### Manual operator steps (not automated in code)

1. Create desktop GA4 property + data stream; create MP API secret
2. Set GitHub Actions variables/secrets for desktop IDs
3. Confirm Vercel `VITE_*` for landing
4. Delete obsolete `POSTHOG_*` GitHub variables
5. In GA4 Admin, mark the key events listed above
6. Smoke-test: release or local ldflag build → GA4 DebugView / realtime

---

## Validation

### Desktop

- `go test ./…` (especially `internal/analytics`)
- Build with empty credentials → no network / no-op
- Build with test Measurement ID + API secret → `app_launched` (and flush) visible in GA4
- Consent off → no events
- Category Minimal → only lifecycle/updates
- Correct desktop credentials only (never landing IDs)

### Landing

- Existing build; GA4 page views + Clarity init when env set
- No desktop IDs in landing bundle

### Docs / repo cleanliness

Repo search after migration should find no operational PostHog dependency (packages, init, CI, required env). Historical RELEASE_NOTES / superseded specs may still mention PostHog as history — prefer updating active wiki/README/privacy; optionally leave dated release notes intact.

---

## Final report (implementation deliverable)

After implementation, report:

1. Files changed  
2. PostHog components removed  
3. GA4 MP implementation (desktop)  
4. Clarity scope (landing only; any tag changes)  
5. Event taxonomy (pointer to category map)  
6. GA4 event → parameters mapping (globals + key events)  
7. Clarity custom tags (landing)  
8. Events migrated (transport) vs renamed (expect: none renamed)  
9. Events intentionally not added  
10. GitHub Actions vars added/removed  
11. Vercel vars found/changed (or “manual verify”)  
12. WordShield: N/A  
13. Web vs desktop separation  
14. Build/test results  
15. Remaining manual configuration  
16. Events requiring manual verification in GA4/Clarity Admin  

---

## Non-goals

- Installing `posthog-js`, `gtag` in the desktop webview, or Clarity in the desktop app  
- Inventing analytics IDs  
- Joining landing and desktop datasets  
- Rebuilding the consent/category system from scratch  
- WordShield deployment changes  
- Marking every event as a GA4 conversion  

---

## Open inputs (from operator)

Provide later (do not invent):

- Desktop: `DESKTOP_GA_MEASUREMENT_ID`, `DESKTOP_GA_API_SECRET`
- Landing (if not already set in Vercel): `VITE_GA_ID`, `VITE_CLARITY_ID`
