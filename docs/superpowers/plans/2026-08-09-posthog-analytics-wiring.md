# PostHog Analytics Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire PostHog CI key usage, category/detail analytics prefs, Go-backed frontend API, remaining events, and dashboard script per `docs/superpowers/specs/2026-08-09-posthog-analytics-wiring-design.md`.

**Architecture:** Extend `internal/analytics` with category map + prefs persistence; gate `Track` before queue; expand Privacy Settings UI and Wails bindings; thin PostHog-shaped JS bridge; fill event gaps; one-shot dashboard script.

**Tech Stack:** Go (Wails), React frontend, PostHog HTTP capture + management API, GitHub Actions variables.

## Global Constraints

- No `posthog-js`; no webview network to PostHog.
- No random sampling; category/detail gating only.
- Master + all categories default **on** (`detailLevel: full`).
- Never send paths, names, env keys/values, free text.
- Project key via Actions **variable** `POSTHOG_API_KEY`; personal key never in CI.

## File map

| File | Role |
|---|---|
| `internal/analytics/category.go` | Event→category map, presets, `CategoryAllowed` |
| `internal/analytics/consent.go` | Persist detailLevel + categories |
| `internal/analytics/client.go` | Gate Track on category |
| `internal/analytics/category_test.go` | Map + preset tests |
| `analytics.go` | Extended settings bindings |
| `analytics_ops.go` | `env_file_edited` in trackProjectSaved |
| `frontend/src/analytics.js` | PostHog-shaped API |
| `frontend/src/components/PrivacySettings.jsx` | UI |
| `frontend/src/components/LogPanel.jsx` | `logs_opened` |
| `frontend/src/components/PortsView.jsx` / `App.jsx` | `ports_viewed` |
| `frontend/src/components/TaskTracker.jsx` | `roadmap_opened` |
| `scripts/setup-posthog-dashboards.sh` | Dashboard creation |
| `.github/workflows/build.yml` | Comment required vars |
| `docs/wiki/Analytics-and-Privacy.md`, `Build-and-Release.md`, `docs/privacy.md` | Docs |

---

### Task 1: Category map + prefs persistence + Track gate

**Files:** Create `internal/analytics/category.go`, `category_test.go`; Modify `consent.go`, `client.go`

- [ ] Add `Category` constants and `EventCategory(name) Category` explicit map
- [ ] Add `DefaultCategories()`, `CategoriesForLevel(level)`, `NormalizeDetailLevel`
- [ ] Extend `ConsentState` with `DetailLevel` + `Categories`; load/save with defaults
- [ ] Client holds category flags; `Track` checks category after enabled
- [ ] Methods: `Prefs()`, `SetDetailLevel`, `SetCategories`, `CategoryEnabled`
- [ ] Tests for map coverage of known events and preset expansion
- [ ] `go test ./internal/analytics/ -count=1`

### Task 2: Wails bindings

**Files:** Modify `analytics.go`, `analytics_test.go`, `frontend/src/api.js`

- [ ] Extend `AnalyticsSettings` with `detailLevel`, `categories`
- [ ] `SetAnalyticsDetailLevel(level string) error`
- [ ] `SetAnalyticsCategories(cats map[string]bool) error`
- [ ] Export bindings in `api.js`

### Task 3: Privacy Settings UI + frontend API

**Files:** Modify `PrivacySettings.jsx`, `analytics.js`; CSS if needed

- [ ] Detail level control + category toggles
- [ ] `capture` / `captureOnce` / `optIn` / `optOut` / `getSettings` / `setDetailLevel` / `setCategories`
- [ ] Wrap existing `track*` helpers

### Task 4: Event gaps

**Files:** `LogPanel.jsx`, `PortsView.jsx`/`App.jsx`, `TaskTracker.jsx`, `analytics_ops.go`

- [ ] `logs_opened`, `ports_viewed`, `env_file_edited`, `roadmap_opened`
- [ ] Unit test for env key-set diff helper

### Task 5: CI comment + GH variable + dashboard script + docs

**Files:** `build.yml`, `scripts/setup-posthog-dashboards.sh`, wiki + privacy docs

- [ ] Ensure `POSTHOG_API_KEY` repo variable is set
- [ ] Document required vars in workflow comment
- [ ] Dashboard script (idempotent by name)
- [ ] Update Analytics-and-Privacy, Build-and-Release, privacy.md

### Task 6: Verify

- [ ] `go test ./internal/analytics/ ./...` (focused packages)
- [ ] `gh variable list` shows `POSTHOG_API_KEY`
