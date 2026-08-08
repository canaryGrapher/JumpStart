# How to Change Things

Practical recipes. For background see [Architecture](Architecture) and [Contributing](Contributing).

## Change the default start command for a detected stack

**File:** `internal/detect/frameworks.go`  
Adjust `detectNode` / `detectPython` / … return `Command`. Add a test under `detect_test.go` with a temp directory containing the marker files.

## Change kanban columns

**Files:** `frontend/src/components/kanban/columns.js`, styles in `_kanban.scss`, and any analytics that assume column names (`analytics_kanban.go`).  
Task `status` strings are part of the on-disk model — migrating existing `config.json` values if you rename columns.

## Change what “Fill with AI” returns

**File:** `ai_api.go` (`OllamaEnrichTask`) and the modal that applies `EnrichResult` (`TaskDetailModal.jsx`).  
Keep the JSON shape stable or version it; the modal maps fields into `model.Task`.

## Change port detection

**Files:** `internal/procman/ports.go` (+ `_unix` / `_windows`).  
Log-based detection is best-effort; OS polling covers processes that never print a URL.

## Change stop / kill behaviour

**Files:** `proc_unix.go` / `proc_windows.go`.  
Preserve group kill so `npm run dev` children die with the shell. App-level `stopping` / `shuttingDown` flags must stay consistent or crash analytics will lie.

## Add a Preferences section

1. Edit `Preferences.jsx` (section list + panel).
2. Persist either:
   - **Go** `settings.json` (extend `analytics.Settings` pattern in `consent.go` / a new settings helper), or
   - **localStorage** (theme/AI/channel pattern).
3. If Go needs the value at runtime (analytics channel), add a binding to push it across.

## Send an in-app announcement (remote banner)

Edit `social/banner.json`, give the entry a **new unique `id`**, put it first in the array, commit, and **push to `main`**. Installed apps fetch that file on launch — no new binary required.

Full field reference, scheduling, dismissal rules, and failure checklist: [Remote Banners](Remote-Banners).

## Change update check repository

**File:** `version.go` constants `UpdateOwner`, `UpdateRepo`.  
Updater reads GitHub Releases for that repo.

## Change product / window chrome

**Files:** `main.go` (Wails options), `wails.json` (name, company), `theme_darwin.go` (vibrancy/appearance).  
Transparent backgrounds are intentional for sidebar glass — do not paint an opaque full-window background in CSS without checking `_glass.scss` / sidebar styles.

## Add offline-safe work

Anything that must work without network already should: process control, local Ollama, local config.  
If you add network calls, fail soft in the UI and avoid blocking `Shutdown` (analytics already uses a 2s bound).

## Rename module `devdeck` → `jumpstart`

Large intentional migration: every import path, plus communication to forks. Not required for features; leave alone unless you are doing a dedicated rename PR.

## Ship a release

1. Update `RELEASE_NOTES.md` “Unreleased” → version section.
2. Tag `vX.Y.Z` (stable) or `vX.Y.Z-beta.N` (beta).
3. CI builds all platforms, signs macOS ad-hoc, uploads assets.
4. Verify About version matches the tag and updater sees the release on the right channel.
