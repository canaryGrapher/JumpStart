# Architecture

## Runtime

JumpStart is a **Wails v2** desktop app:

1. `main.go` embeds `frontend/dist`, constructs `App`, and calls `wails.Run`.
2. Only `app` is in `Bind` — every frontend call hits exported `(*App)` methods.
3. macOS uses a hidden-inset title bar, transparent webview, and translucent window so AppKit vibrancy shows through the sidebar. Windows uses Mica similarly.
4. A top-level `recover` in `main` reports panics to analytics (`trackPanic`) then re-panics.

Cold-start timing starts at package init (`processStart`) so `app_launched.cold_start_ms` measures the whole launch, not only post-Wails work.

## App struct

Defined in `app.go`:

| Field | Role |
|-------|------|
| `ctx` | Wails context for dialogs and `EventsEmit` |
| `store` | `*store.Store` → `~/.jumpstart/config.json` |
| `manager` | `*procman.Manager` — all long-lived subprocesses |
| `mcp` | `*mcpserver.Server` — optional localhost Agents MCP |
| `scriptRuns` | In-memory script/test run bookkeeping (`scripts_api.go`) |
| `analytics` | `*analytics.Client` (nil-safe no-op when unconfigured) |
| `procStarts` / `stopping` | Maps for uptime and intentional-stop vs crash |
| `shuttingDown` | Suppresses crash events during quit |
| `lastDetect` | Attributes a subsequent save to auto-detect |

### Lifecycle

**Startup**

1. Create `procman.Manager` with an emitter that (a) tracks process exit/crash analytics and (b) forwards to `runtime.EventsEmit`.
2. Open `store.New()` (migrates `~/.devdeck/config.json` → `~/.jumpstart/config.json` once).
3. `initAnalytics()` then `trackLaunch()`.
4. `initMCP()` — starts the Agents MCP listener if enabled in `~/.jumpstart/mcp.json`.

**Shutdown**

1. Set `shuttingDown`.
2. `stopMCP()` then `manager.StopAll()`.
3. `trackClose()` then `analytics.Close(2s)` — undelivered events stay on the offline queue.

## Layering rules

1. **Bindings are thin.** Prefer calling `internal/*` from `App` methods; keep HTTP/CLI/OS details out of `app.go` when a package already owns them.
2. **Instrument at the binding.** Analytics wrappers (`gitOp`, `dockerOp`, `trackProcessStarted`, …) sit next to the call so every path is covered and raw errors never leak.
3. **Events for push, methods for pull.** Live logs/ports/exits use `EventsEmit`; status snapshots use `GetStatus` / `GetUsage`.
4. **Platform files use build tags.** Examples: `theme_darwin.go` / `theme_other.go`, `procman/proc_unix.go` / `proc_windows.go`, `analytics/osinfo_*.go`, `opener/platform_*.go`.

## Event bus (Wails runtime events)

| Event | Producer | Consumer |
|-------|----------|----------|
| `log:<id>` | procman | `LogPanel`, script/test logs |
| `ports:<id>` | procman | `ProcessCard` |
| `exit:<id>` | procman | cards, deps install, tests, scripts |
| `update:progress` / `update:ready` | `InstallUpdate` | `UpdateBanner` |
| `codectx:progress` / `codectx:done` | `BuildCodeContext` | `codeContext.js` |

IDs are process IDs, or synthetic IDs like `<procID>:deps` and script/test run IDs.

## Request flow examples

### Start a process

```
ProcessCard → StartProcess(projectID, procID)
  → findProcess → manager.Start
  → trackProcessStarted
  → touchUsage (LastUsedAt / UseCount)
  → frontend listens log:/ports:/exit:
```

### Fill task with AI

```
TaskDetailModal → OllamaEnrichTask(host, model, …, projectID)
  → optional codectx retrieval
  → internal/ai Ollama /api/chat
  → EnrichResult JSON for the modal
  → acceptance tracked from UI via analytics bridge
```

### Theme change

```
App.jsx useTheme → localStorage + dataset.theme
  → SetNativeTheme(mode)
  → setNativeAppearance (ObjC on Darwin) + WindowSet*Theme
```

Wails’ `WindowSetLightTheme` / `DarkTheme` are Windows-oriented; on macOS, `theme_darwin.go` pins `NSApp.appearance` so sidebar vibrancy matches the in-app theme (not only the OS appearance).

## Module name nuance

`go.mod` says `module devdeck`. Imports are `devdeck/internal/...`. The binary/product is **JumpStart**; legacy `~/.devdeck` is only used for one-time config migration.

## Frontend ↔ Go contract

- All bound methods are re-exported from `frontend/src/api.js` (single import surface).
- Bindings are generated under `frontend/wailsjs/` by `wails dev` / `wails build` — **do not hand-edit**.
- Prefer `api.js` over importing `wailsjs` directly in components.
