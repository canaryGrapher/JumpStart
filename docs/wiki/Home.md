# JumpStart Developer Wiki

JumpStart is a cross-platform desktop control panel for local apps and development projects. Add a project once, auto-detect runnable processes, then start/stop, inspect logs and ports, manage git, Docker, scripts, tests, and a per-project Kanban — all from one native-feeling Wails app.

**Stack:** Go (Wails v2) · React 18 · Vite · SCSS  
**Module path:** `devdeck` (historical; product name is JumpStart)  
**Vendor:** Workvar · Repo: [canaryGrapher/JumpStart](https://github.com/canaryGrapher/JumpStart)

## Wiki map

| Page | What it covers |
|------|----------------|
| [Architecture](Architecture) | Layering, App bootstrap, events, data flow |
| [Code-Structure](Code-Structure) | Repository layout and package map |
| [Features](Features) | Product features mapped to code |
| [Data-Model](Data-Model) | `~/.jumpstart/` files and JSON shapes |
| [Backend-API](Backend-API) | All Wails-bound `App` methods |
| [Frontend](Frontend) | React shell, API layer, styles, hooks |
| [Process-Management](Process-Management) | procman, ports, logs, platform nuances |
| [AI-and-Chat](AI-and-Chat) | Ollama, codectx, chatstore |
| [Agents-MCP](Agents-MCP) | Local MCP server for external AI agents |
| [Analytics-and-Privacy](Analytics-and-Privacy) | GA4 MP, consent, redaction |
| [Build-and-Release](Build-and-Release) | Dev, CI, ldflags, signing |
| [Contributing](Contributing) | How to change things safely |
| [How-to-Change](How-to-Change) | Concrete recipes for common edits |
| [Remote-Banners](Remote-Banners) | Publish in-app announcements via `banner.json` |
| [Landing-Site](Landing-Site) | Marketing site (`landing/`) |

## Quick start (contributors)

```sh
# Prerequisites: Go 1.22+, Node 18+, Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0

wails dev          # hot reload; generates frontend/wailsjs/
wails build        # output in build/bin/
go test ./...      # Go unit tests
```

First `wails dev` installs frontend deps and generates the Go↔JS bindings under `frontend/wailsjs/`.

## Mental model

```
┌─────────────────────────────────────────────────────────┐
│  frontend/ (React)                                      │
│  App.jsx → panels → api.js → wailsjs/go/main/App        │
└───────────────────────────┬─────────────────────────────┘
                            │ Wails bindings + EventsEmit
┌───────────────────────────▼─────────────────────────────┐
│  main package: App (app.go + *_api.go)                  │
│  Thin orchestration + analytics instrumentation         │
└───────────────────────────┬─────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────┐
│  internal/*  domain packages (store, procman, detect,…) │
└─────────────────────────────────────────────────────────┘
```

- **Frontend never owns product analytics ingestion.** It only bridges UI-only signals (`panel_opened`, AI accept) through Go.
- **`App` is the only Wails-bound type.** Domain logic lives in `internal/`.
- **Projects live in `~/.jumpstart/config.json`.** Sidecars (chats, code index, analytics) are separate files so they do not rewrite the main config on every chat turn.

## Where to look first

| Goal | Start here |
|------|------------|
| Add a backend method | `app.go` or a focused `*_api.go`, export from `frontend/src/api.js` |
| Change process start/stop | `internal/procman/` |
| Add a language detector | `internal/detect/frameworks.go` |
| Change Kanban UI | `frontend/src/components/kanban/` |
| Touch privacy/events | `internal/analytics/` + `docs/privacy.md` |
| Ship a release | tag `vX.Y.Z` → `.github/workflows/release.yml` |
