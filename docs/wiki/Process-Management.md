# Process Management

Package: `internal/procman`.

## Responsibilities

- Start/stop commands in an isolated process group
- Ring-buffer logs
- Discover listening ports (log scrape + OS poll)
- Emit `log:`, `ports:`, `exit:` events through the injected `Emitter`

## Start semantics

1. If the same ID exists and is still running → error.
2. If it exited but a child may hold the port → `killTree` before restart.
3. Command runs via platform shell helper (`shellCommand`).
4. Env = resolved process environment + per-process `Env` map.
5. Goroutines: stdout pipe, stderr pipe, port poll (≈15 × 2s), wait for exit.

## Stop semantics

- Signal the process group (Unix: SIGTERM then SIGKILL after grace; Windows: job/tree kill).
- App marks intentional stops in `stopping` so non-zero exits are not reported as `process_crashed`.
- App shutdown sets `shuttingDown` and stops everything.

## Ports

- From log lines (`portsFromLine`) and `portsForGroup` / `lsof`-style helpers (`ports_unix.go` / `ports_windows.go`).
- Aggregated in UI via `GetPortMap` and per-card badges (open `http://localhost:<port>`).

## Logs

- In-memory `logBuf` per process; `GetLogs` returns a snapshot; live lines via events.
- Scanner allows large lines (up to 1MB buffer).

## Platform files

| File | Role |
|------|------|
| `proc_unix.go` / `proc_windows.go` | Process group + kill |
| `env_unix.go` / `env_windows.go` | Base environment |
| `ports_unix.go` / `ports_windows.go` | Port enumeration |
| `ports.go` | Shared parsing / merge |
| `logbuf.go` | Log ring buffer |
| `manager.go` | Orchestration |

## Related runners

Scripts, tests, and deps-install reuse the same manager + event names with synthetic IDs so `LogPanel` / `EventsOn('exit:…')` work unchanged.

## Gotchas for contributors

- Never start long-lived work outside procman if the UI expects logs/ports.
- Always use stable IDs; regenerating IDs breaks event subscriptions.
- Compose processes may be hidden from the Processes tab — check `frontend/src/procUtils.js`.
- Closing the app **always** stops managed processes (by design).
