# Features

Product capabilities mapped to implementation. Use this when deciding *where* a change belongs.

## Project library

- **UI:** `Sidebar`, `Dashboard`, `ProjectModal`, `App.jsx`
- **API:** `GetProjects`, `SaveProject`, `DeleteProject`, `SetProjectFavorite`
- **Nuances:** Favorites group at top of sidebar; `LastUsedAt` / `UseCount` updated on process start (`touchUsage`). Delete also evicts `codectx` + `chatstore` sidecars.

## Process control

- **UI:** `ProcessCard`, `LogPanel`, `EnvEditor`, `EnvImportPrompt`, `PortsView`
- **API:** `StartProcess`, `StopProcess`, `StartAll`, `StopAll`, `GetStatus`, `GetLogs`, `GetUsage`, `GetPortMap`
- **Core:** `internal/procman`
- **Nuances:** Compose-shaped processes are filtered from the Processes tab (`procUtils.isComposeProc`) and managed under Containers when Docker is present.

## Auto-detection

- **UI:** `ProjectModal` (scan on folder pick)
- **API:** `DetectProcesses`, `ReadEnvFile`, `PickDirectory`
- **Core:** `internal/detect` — markers for Node, Go, Rust, Python, Ruby, PHP, Maven, Gradle, Docker Compose; max depth 6; skips `node_modules`, `.git`, build dirs, etc.
- **Analytics:** Detection remembered via `lastDetect` so the next save can be tagged `source: detect`.

## Dependencies & scripts

- **UI:** `DepsPanel`, `components/scripts/*`
- **API:** `GetDependencies`, `InstallDeps`, `DetectScripts`, `RunScript`, `StopScriptRun`, …
- **Core:** `internal/deps`, `internal/scripts`
- **Nuance:** Deps install is a managed process with ID `<procID>:deps`; UI waits on `exit:<id>`.

## Kanban / tasks / sprints / roadmap

- **UI:** `TaskTracker`, `kanban/*`, `roadmap/*`
- **API:** `UpdateTasks`, `UpdateSprints`
- **Model:** Tasks in project JSON; columns `backlog|todo|inprogress|testing|done`; types `story|task|bug`; stories use `ParentID` / acceptance criteria.
- **Toggle:** `project.tasksEnabled` gates the Tasks tab.

## AI (local Ollama)

- **UI:** Preferences AI section, `TaskDetailModal` fill, `ChatDock`, git `CommitBox`
- **API:** `OllamaListModels`, `OllamaEnrichTask`, `OllamaChat`, `OllamaGenerateCommitMessage`, `GenerateProjectDescription`
- **Settings:** Host/model in **frontend localStorage** (`ai.js`), not Go settings.
- **Default host:** `http://localhost:11434` (or `OLLAMA_HOST`).

## Agents MCP

- **UI:** Preferences → Agents (`AgentsSettings.jsx`)
- **API:** `GetMCPSettings`, `SetMCPSettings`, `RotateMCPToken`
- **Core:** `internal/mcpserver` — localhost streamable HTTP MCP for Cursor and other agents
- **Docs:** [Agents-MCP](Agents-MCP)

## Code context & chat

- **UI:** Chat dock hooks (`useCodeContext`, `useChatSessions`)
- **API:** `BuildCodeContext`, `SearchCodeContext`, `SendChatMessage`, chat CRUD
- **Core:** `internal/codectx` (BM25, local), `internal/chatstore` (`~/.jumpstart/chats/<id>.json`, max 50 sessions/project)

## Git

- **UI:** `GitPanel`, `git/*` (diff, branches, timeline, commit)
- **API:** `GitStatus`, fetch/pull/push/commit, branches, diff, stashes, tokens
- **Tokens:** OS keychain via `internal/secrets` — never written to `config.json`.
- **Commit AI:** `internal/gitops/commitctx.go` builds diff context for Ollama.

## Docker

- **UI:** `containers/*` tab (shown if compose/Dockerfile present)
- **API:** `DockerInfo`, Compose up/down, container/image/volume lists, start/stop/remove

## Wiki

- **UI:** `wiki/WikiPanel` tab (shown when a local wiki is detected)
- **API:** `WikiInfo`, `WikiPage`
- **Core:** `internal/wiki`
- **Layout:** GitHub-wiki-compatible markdown with a **left** pages sidebar (`_Sidebar.md` when present, otherwise an auto page list). Renders `_Footer.md` under the page body.
- **Locations (first match wins):** `project/.wiki/`, `project/docs/wiki/`, or a sibling `ProjectName.wiki/` clone — for private repos that cannot enable GitHub Wiki on the free plan.

## Tests

- **UI:** `TestPanel`
- **API:** `DetectTestConfig`, `RunTests`
- **Overrides:** `project.testCommand` or per-process `testCommand`

## Import / export config

- **UI:** `ImportConfigModal`, `import/BlockBuilder`
- **API:** `ImportConfig`, `ImportConfigText`, `ImportConfigFile`, `GetImportPath`, …
- **File:** `~/.jumpstart/import.json` — bare array or `{ "projects": [...] }`

## Open in Finder / Terminal

- **UI:** `OpenActions` (labels via `platform.js`)
- **API:** `OpenInFileManager`, `OpenInTerminal` → `internal/opener`

## Updates & banners

- **UI:** `UpdateBanner`, `UpdateSettings`, `useUpdateCheck`, `useRemoteBanner`
- **API:** `CheckForUpdate`, `InstallUpdate`, `RestartApp`, `GetRemoteBanner`
- **Flow:** when a newer release is found, the banner appears and `InstallUpdate` starts automatically (progress in-banner); user only confirms Restart. Dismiss snoozes that version (in-flight download may still finish; no Restart prompt if snoozed).
- **Channels:** stable vs beta via localStorage; beta prefers pre-release tags.
- **Repo:** `version.go` → `UpdateOwner` / `UpdateRepo`
- **Remote announcements:** maintainers publish JSON at `social/banner.json` on `main` — see [Remote Banners](Remote-Banners).

## Preferences / About / Privacy / Contribute

- **UI:** `Preferences` (tabs), `PrivacySettings`, `about/*`, `contribute/*`
- **API:** analytics settings, `GetAboutInfo`, contribute issue APIs, `CollectDiagnostics`

## Theme & chrome

- Theme / accent / sidebar width / open state in localStorage.
- Accent via `document.documentElement.dataset.accent` + `styles/_accent.scss`.
- Native theme sync is mandatory for correct vibrancy (see Architecture).
