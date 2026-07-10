# DevDeck

Desktop control panel for your local dev projects. Built with Wails (Go) + React, styled after the Cloudflare dashboard.

Add a project, define its subprocesses (e.g. Next.js frontend, Go Fiber backend), and start or stop them with one click. No more juggling terminal tabs.

## Features

Start/stop any subprocess with a click (the whole card is clickable), start all / stop all per project, live log streaming per process, automatic port detection (from logs and lsof), clickable port badges that open localhost in your browser, per-process env vars, and JSON persistence at `~/.devdeck/config.json`.

## Requirements

- Go 1.22+
- Node.js 18+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- macOS: Xcode command line tools (`xcode-select --install`)

## Run

```sh
wails dev
```

First run generates `frontend/wailsjs/` bindings and installs npm deps automatically.

## Build a distributable app

```sh
wails build
```

Output lands in `build/bin/` (a .app on macOS).

## Usage

1. Click "+ Add project", set a name and root folder.
2. Add subprocesses: name, working directory, start command (e.g. `npm run dev`, `go run .`, `rails s`), optional env vars.
3. Click a card (or its Start button) to launch. The card shows status, PID, and detected ports. Click a port badge to open it in the browser.
4. "Logs" toggles a live output panel per process.

## Structure

```
main.go                     Wails bootstrap
app.go                      API exposed to the frontend
internal/model/             data types
internal/store/             JSON config persistence
internal/procman/           process lifecycle, logs, port detection
frontend/src/               React UI (cf-ui inspired styling)
```

## Notes

- Processes run via `/bin/sh -c` in their own process group; Stop sends SIGTERM to the group, then SIGKILL after 5s. Unix only (macOS/Linux).
- Closing the app stops all running processes.
