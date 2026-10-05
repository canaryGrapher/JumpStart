# Backend API

All methods are on `*App` and exposed through Wails. Frontend imports them via `frontend/src/api.js`.

Return values and errors cross the binding boundary as promises. Prefer returning structured types over free-form strings when the UI must branch.

## Projects & processes

| Method | Notes |
|--------|-------|
| `GetProjects` | Full project list |
| `SaveProject` | Insert or replace by ID; tracks create vs update |
| `DeleteProject` | Also deletes codectx + chats |
| `SetProjectFavorite` | Targeted write |
| `StartProcess` / `StopProcess` | Single process |
| `StartAll` / `StopAll` | Per project; StartAll returns error strings |
| `GetStatus` / `GetLogs` | From procman |
| `GetUsage` | System + per-PID via sysinfo |
| `GetPortMap` | Aggregate listening ports for running procs |

## Tasks

| Method | Notes |
|--------|-------|
| `UpdateTasks` | Replaces task slice; diffs for analytics |
| `UpdateSprints` | Reindexes `Order` from slice position |

## Detect / import / deps

| Method | Notes |
|--------|-------|
| `DetectProcesses` | `detect.Scan` |
| `ReadEnvFile` | dotenv → map |
| `PickDirectory` / `PickConfigFile` | Native dialogs |
| `GetImportPath`, `ImportConfig`, `ImportConfigText`, `ImportConfigFile`, `ReadConfigFile` | Import flows |
| `GetDependencies`, `InstallDeps` | InstallDeps returns synthetic proc ID |

## AI / chat / code context

| Method | Notes |
|--------|-------|
| `OllamaListModels` | |
| `OllamaEnrichTask` | Task fill |
| `OllamaChat` | Stateless chat helper |
| `OllamaGenerateCommitMessage` | Needs project root |
| `GenerateProjectDescription` | |
| `BuildCodeContext` / `GetCodeContextStatus` / `ClearCodeContext` / `SearchCodeContext` | |
| `ListChats` / `GetChat` / `NewChat` / `RenameChat` / `DeleteChat` | |
| `SendChatMessage` | Persists turns; may attach retrieved sources |

## Git / release / secrets

| Method | Notes |
|--------|-------|
| `GitStatus`, `GitInit`, `GitFetch`, `GitPull`, `GitPush`, `GitCommit` | Wrapped with `gitOp` analytics |
| `GitAddRemote`, branch CRUD, `GitGraphLog`, `GitDiff`, `GitListStashes` | |
| `SaveGitToken` / `HasGitToken` / `DeleteGitToken` | Keychain |
| `CreateRelease` | `internal/release` |

## Docker

`DockerInfo`, `ComposeUp`, `ComposeDown`, `ListContainers`, `ListImages`, `ListVolumes`, `StartContainer`, `StopContainer`, `RemoveContainer` — docker actions use `dockerOp` for analytics.

## Scripts & tests

| Method | Notes |
|--------|-------|
| `DetectScripts` | |
| `RunScript` / `StopScriptRun` / `ListScriptRuns` / `ListProcessScriptRuns` / `GetScriptRunStatus` | Run IDs share log/exit events with procman |
| `DetectTestConfig` / `RunTests` | |

## Updates / about / opener / contribute / analytics / agents

| Method | Notes |
|--------|-------|
| `GetAppVersion` | Trims leading `v` for display |
| `CheckForUpdate` / `InstallUpdate` / `RestartApp` | Channel from frontend wrapper |
| `GetRemoteBanner` | |
| `GetAboutInfo` | Version, build date, vendor |
| `OpenInFileManager` / `OpenInTerminal` | |
| `GetContributeInfo` / `SubmitIssue` / `ListRepoIssues` / `CollectDiagnostics` | |
| `GetAnalyticsSettings` / `SetAnalyticsEnabled` | |
| `TrackEvent` / `TrackEventOnce` / `TrackModelSelected` / `SetUpdateChannel` | Used by `frontend/src/analytics.js` |
| `SetNativeTheme` | light/dark/system + macOS appearance |
| `GetMCPSettings` / `SetMCPSettings` / `RotateMCPToken` | Local Agents MCP server (see [Agents-MCP](Agents-MCP)) |

## Adding a new binding

1. Add `func (a *App) Foo(...)` in `app.go` or a dedicated `*_api.go`.
2. Put domain logic in `internal/<pkg>` when non-trivial.
3. Add analytics if it is a user-facing action (bounded props only).
4. Run `wails dev` (or build) to regenerate `wailsjs`.
5. Export from `frontend/src/api.js`.
6. Call from UI; never import `wailsjs/go/main/App` in feature files if `api.js` already re-exports it.
