# JumpStart

JumpStart is a macOS desktop control panel for the applications and development projects on your machine. Add a project once, let JumpStart detect its runnable parts, then start, stop, inspect, and organize everything from one native-feeling app.

Built with Wails, Go, React, and Vite.

## Developer documentation

Full architecture, API, data model, and contribution guides live in the
[GitHub Wiki](https://github.com/canaryGrapher/JumpStart/wiki).
Source markdown is mirrored under [`docs/wiki/`](docs/wiki/) and can be
re-published with `./scripts/publish-wiki.sh`.

## Features

- Project library with recent and most-used project shortcuts.
- One-click start/stop for individual subprocesses or every process in a project.
- Live process status, PID, exit code, logs, CPU usage, and memory usage.
- Automatic port detection from logs and `lsof`, with clickable localhost port badges.
- Project auto-detection for nested monorepos, including Node, Go, Python, Ruby, PHP, Java/Maven, Gradle, Rust, and Docker Compose projects.
- Per-process environment variables and dotenv import prompts.
- Dependency inspection and install actions for common package managers.
- Per-project Kanban board (Backlog, To Do, In Progress, Testing, Done) with user stories, tasks, and bugs, labels, priorities, subtasks, and progress tracking.
- User stories that contain child tasks, with acceptance criteria, story points, and assignees. Click any card to edit it in a detail modal.
- One-click "Fill with AI" in the task modal, using a local Ollama model to draft descriptions, acceptance criteria, subtasks, priority, and labels.
- A story-assistant chat pinned to the board that expands to full screen, where you can generate single stories or whole batches and add them to the board.
- AI settings in Preferences to auto-detect and select an installed Ollama model (defaults to `http://localhost:11434`).
- Agents MCP server (Settings → Agents) so external AI agents can control projects, processes, files, git, and tasks over localhost.
- Two-way sync between a project's board and a GitHub Projects v2 board, with every board field readable on the card and the writable ones editable. Connect once in Settings, link a board per project.
- Live port usage table across all managed processes.
- JSON import flow for adding projects programmatically.
- macOS-style interface with native titlebar behavior, translucent sidebar, and Light / Dark / Auto appearance in Settings.

## Data Locations

JumpStart stores its main project config at:

```sh
~/.jumpstart/config.json
```

Programmatic imports are read from:

```sh
~/.jumpstart/import.json
```

On first launch after upgrading from older builds, JumpStart copies an existing `~/.devdeck/config.json` into the new JumpStart config location if the new file does not already exist.

## Requirements

- Go 1.22+
- Node.js 18+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- macOS: Xcode command line tools (`xcode-select --install`)
- Windows: WebView2 runtime (bundled on Windows 10/11) and a C toolchain (`gcc` via MSYS2/TDM-GCC)

Prebuilt macOS, Windows, and Linux packages are published on the [Releases page](https://github.com/canaryGrapher/JumpStart/releases).

## Installing the macOS release

JumpStart is ad-hoc signed but not notarized by Apple, so the first launch shows a Gatekeeper warning. This is expected. To open it:

1. Unzip and move **JumpStart.app** to your Applications folder.
2. Double-click it. macOS says it "could not verify" the app; click **Done** (not Move to Bin).
3. Open **System Settings → Privacy & Security**, scroll to the **Security** section, and click **Open Anyway** next to the JumpStart message.
4. Confirm with Touch ID or your password.

This is a one-time step per machine. After that, JumpStart opens normally.

## Run

```sh
wails dev
```

First run generates `frontend/wailsjs/` bindings and installs npm dependencies automatically.

## Build

```sh
wails build
```

The app bundle/executable is written to `build/bin/`.

## Analytics

The app and the website use different tools, and the two data sets are never joined.

**The desktop app** reports anonymous product usage to Google Analytics 4 from the Go process (`internal/analytics`) via the Measurement Protocol, not from the frontend bundle. Wails ships a webview rather than a browser, so a JS SDK would need CSP exemptions, would deliver nothing offline, and would be blind to `Startup`/`Shutdown`. Every user action already crosses the Wails binding boundary, so that is where it is instrumented.

It is on by default and can be turned off in Settings → Privacy. No filesystem paths, project or process names, commands, repo URLs, commit messages, environment variables, credentials, or AI prompts are ever sent — see [`docs/privacy.md`](docs/privacy.md) for the full list and `internal/analytics/redact.go` for the enforcement.

GA4 credentials are injected at build time, so builds made without them send nothing:

```sh
wails build -ldflags "-X main.Version=v1.4.0 -X main.GAMeasurementID=G-XXX -X main.GAAPISecret=..."
```

In CI they come from the `DESKTOP_GA_MEASUREMENT_ID` repository variable and `DESKTOP_GA_API_SECRET` repository secret (write-only MP credentials, safe to ship in a binary). Delete obsolete `POSTHOG_*` repo vars.

**The landing site** (`landing/`) uses GA4, Microsoft Clarity, and Vercel Analytics, wired through Vite env vars and disabled unless IDs are set. Copy `landing/.env.example` to `landing/.env` and fill in:

```sh
VITE_GA_ID=G-XXXXXXXXXX
VITE_CLARITY_ID=xxxxxxxxxx
```

## Usage

1. Click **Add Project** and choose a project folder.
2. Let JumpStart auto-detect runnable subprocesses, or add processes manually.
3. Start a process from its card, or use **Start all** on the project.
4. Open logs, dependencies, detected ports, and resource usage from each process card.
5. Use the **Tasks** tab to track built and pending project work.

## Config Import

The Dashboard can copy a prompt that asks an AI agent to inspect your repositories and write a valid `~/.jumpstart/import.json`. After the file is written, click **Import config** in JumpStart.

The import format accepts either a bare project array or:

```json
{
  "projects": [
    {
      "name": "Project Alpha",
      "root": "/absolute/path/to/project",
      "tasksEnabled": true,
      "processes": [
        {
          "name": "frontend",
          "dir": "/absolute/path/to/project/frontend",
          "command": "npm run dev",
          "env": { "PORT": "3000" }
        }
      ],
      "tasks": [
        { "title": "Auth flow", "done": true },
        { "title": "Billing page", "done": false }
      ]
    }
  ]
}
```

## Testing

```sh
go test ./...
cd frontend && npm run build
```

The Go tests cover config import, config storage, recursive project detection, and dependency inspection. The frontend build verifies the React/Wails UI compiles.

## Project Structure

```text
main.go                     Wails bootstrap and macOS window options
app.go                      API exposed to the frontend
internal/config/            JSON import path, loading, and merge logic
internal/detect/            Recursive project and process detection
internal/deps/              Dependency inspection and install command detection
internal/model/             Shared data types
internal/procman/           Process lifecycle, logs, port detection
internal/store/             JSON config persistence
internal/sysinfo/           System and process resource snapshots
frontend/src/               React UI
```

## Notes

- Processes run through `/bin/sh -c` in their own process group.
- Stop sends `SIGTERM` to the group, then `SIGKILL` after five seconds.
- Closing JumpStart stops all running managed processes.
