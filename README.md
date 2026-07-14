# JumpStart

JumpStart is a macOS desktop control panel for the applications and development projects on your machine. Add a project once, let JumpStart detect its runnable parts, then start, stop, inspect, and organize everything from one native-feeling app.

Built with Wails, Go, React, and Vite.

## Features

- Project library with recent and most-used project shortcuts.
- One-click start/stop for individual subprocesses or every process in a project.
- Live process status, PID, exit code, logs, CPU usage, and memory usage.
- Automatic port detection from logs and `lsof`, with clickable localhost port badges.
- Project auto-detection for nested monorepos, including Node, Go, Python, Ruby, PHP, Java/Maven, Gradle, Rust, and Docker Compose projects.
- Per-process environment variables and dotenv import prompts.
- Dependency inspection and install actions for common package managers.
- Per-project Kanban board (Backlog, To Do, In Progress, Done) with user stories, tasks, and bugs, labels, priorities, subtasks, and progress tracking.
- User stories that contain child tasks, with acceptance criteria, story points, and assignees. Click any card to edit it in a detail modal.
- One-click "Fill with AI" in the task modal, using a local Ollama model to draft descriptions, acceptance criteria, subtasks, priority, and labels.
- A story-assistant chat pinned to the board that expands to full screen, where you can generate single stories or whole batches and add them to the board.
- AI settings in Preferences to auto-detect and select an installed Ollama model (defaults to `http://localhost:11434`).
- Live port usage table across all managed processes.
- JSON import flow for adding projects programmatically.
- macOS-style interface with native titlebar behavior, translucent sidebar, appearance preferences, and accent colors.

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

## Run

```sh
wails dev
```

First run generates `frontend/wailsjs/` bindings and installs npm dependencies automatically.

## Build

```sh
wails build
```

The macOS app bundle is written to `build/bin/`.

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
