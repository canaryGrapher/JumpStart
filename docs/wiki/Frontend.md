# Frontend

## Stack

- React 18 + Vite 7 + Sass
- Scripts: `npm run dev` | `build` | `preview` (usually via Wails)
- No Redux — React `useState` / `useEffect` in `App.jsx` and feature components

## App shell (`App.jsx`)

Owns:

- `projects` list and reload
- View mode: `dashboard` | `ports` | `project`
- Selected project, modal, toast, usage poll (3s)
- Theme + accent hooks
- Sidebar open/width
- Preferences modal, update banner, remote ad/banner overlays

Children:

- `Sidebar`, `Dashboard`, `PortsView`, `ProjectView`, `ProjectModal`, `Preferences`, `UpdateBanner`, `BuildBadge`, `AdOverlay`

## API layer (`api.js`)

- Single re-export of generated `App` bindings
- Re-exports `EventsOn` / `EventsOff` / `BrowserOpenURL` / `ClipboardSetText` from runtime
- Wraps update checks so callers do not pass the beta flag (`updateChannel.js`)
- Comment block documents the analytics rule: no product SDK in the bundle

## Feature → files

| Area | Components / modules |
|------|----------------------|
| Dashboard / sidebar | `Dashboard`, `Sidebar`, `sidebar/ProjectRow` |
| Project | `ProjectView`, `ProcessCard`, `ProcForm`, `LogPanel`, `EnvEditor` |
| Ports | `PortsView` |
| Kanban | `TaskTracker`, `kanban/KanbanBoard`, `TaskCard`, `TaskDetailModal`, `SprintBar`, `ChatDock`, `chat/*` |
| Git | `GitPanel`, `git/CommitBox`, `DiffModal`, `BranchManager`, `BranchTimeline` |
| Scripts | `scripts/ScriptBar`, `ScriptsEditor`, `ScriptRunsPanel`, `ScriptRunLog` |
| Deps | `DepsPanel` |
| Containers | `containers/*` |
| Wiki | `wiki/WikiPanel`, `wiki/WikiMarkdown` |
| Tests | `TestPanel` |
| Import | `ImportConfigModal`, `import/*` |
| Prefs | `Preferences`, `PrivacySettings`, `UpdateSettings`, `contribute/*`, `about/*` |
| Misc | `OpenActions`, `ConfirmDialog`, `ErrorBoundary`, `SearchableSelect`, `Switch`, `Icon` |

## Styles

`styles.scss` `@use`s partials under `styles/`. Conventions:

- One concern per partial (`_kanban.scss`, `_git.scss`, …)
- Theme tokens in `_theme.scss`; glass/vibrancy helpers in `_glass.scss`
- Accent variants in `_accent.scss` via `data-accent`
- `_fields.scss` loaded last for spacious form overrides

Match existing patterns; do not introduce a parallel design system.

## Hooks

| Hook | Role |
|------|------|
| `useSidebarWidth` | Persist width |
| `useUpdateCheck` | Poll GitHub releases; snooze per version |
| `useRemoteBanner` | Fetch + dismiss remote banners |
| `useScriptRuns` | Script run list state |
| Kanban chat hooks | Sessions + code context wiring |

## Analytics bridge

`analytics.js` → `TrackEvent` / `TrackEventOnce` / `TrackModelSelected` / `SetUpdateChannel`.

Use `trackPanel(panel)` for tab reach (once per session per panel). Never send paths, names, or free text.

## Platform helpers

`platform.js` only adjusts labels (`Finder` vs `Explorer` vs `Files`) from the user agent. Behavior lives in Go `opener`.

## How to add a project tab

1. Add tab button in `ProjectView.jsx` and call `trackPanel('yourpanel')`.
2. Create component under `components/`.
3. Add SCSS partial and `@use` it from `styles.scss`.
4. Wire Go APIs through `api.js`.
5. Prefer local state; lift to `App` only if multiple routes need it.
