# Build and Release

## Local development

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0
wails dev     # Vite + Go, generates frontend/wailsjs
wails build   # → build/bin/JumpStart(.app|.exe)
go test ./...
```

Requirements: Go (see `go.mod`), Node 18+, platform webview deps (macOS CLT; Linux GTK/WebKit; Windows WebView2 + gcc).

## Version stamping

`version.go`:

- `Version` default `"dev"`
- `BuildDate` default `""`
- Release CI sets both via `-ldflags -X main.Version=… -X main.BuildDate=…`

Also stamped: `GAMeasurementID`, `GAAPISecret` (see `analytics.go`).

About pane uses `GetAboutInfo` / `GetAppVersion` (leading `v` trimmed for display).

## CI workflows

| Workflow | Trigger | Role |
|----------|---------|------|
| `build.yml` | `workflow_call` | Matrix: Linux amd64, Windows amd64, macOS universal |
| `release.yml` | tag `v*` excluding `v*-*` | Stable release |
| `release-beta.yml` | pre-release tags (`v*-*`) | Beta channel |

Build job notes:

- Node 20, Wails `v2.13.0`
- Linux: `webkit2_41` build tag
- macOS: stamp `Info.plist` versions **before** codesign; **ad-hoc** sign (`codesign --sign -`) so Gatekeeper offers “Open Anyway” instead of “damaged”
- Artifacts named `jumpstart_<tag>_<platform>.(zip|tar.gz)`
- Desktop GA4 credentials from repo **variable** `DESKTOP_GA_MEASUREMENT_ID` and **secret** `DESKTOP_GA_API_SECRET` — write-only MP credentials. **Required** for release builds to emit analytics; empty → no-op.
- Mark key events manually in GA4 Admin; delete obsolete `POSTHOG_*` repo vars.

## Channels

- Stable updater reads non-prerelease GitHub releases
- Beta (Preferences) considers prerelease tags
- Channel preference is frontend localStorage, reported to Go for analytics

## macOS Gatekeeper

Builds are ad-hoc signed, not notarized. Documented user steps live in README / release notes (Open Anyway once).

## Landing site

Separate Vite app under `landing/`. Env: `VITE_GA_ID`, `VITE_CLARITY_ID` (see `landing/.env.example`). Not bundled into the desktop binary.

## Release notes

Maintain human notes in `RELEASE_NOTES.md`; workflow also sets `generate_release_notes: true` on the GitHub Release.
