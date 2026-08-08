# Code Structure

## Top level

```
jumpstart/
├── main.go              # wails.Run, embed frontend/dist, panic recover
├── app.go               # App struct, projects/processes/tasks/import/docker/git/…
├── *_api.go             # Focused binding groups (ai, chat, codectx, scripts, …)
├── analytics*.go        # Analytics wiring on App (not the package itself)
├── version.go           # Version, BuildDate, UpdateOwner/Repo, Vendor*
├── theme_darwin.go      # NSAppearance bridge (darwin)
├── theme_other.go       # no-op stub
├── wails.json           # product name JumpStart, frontend npm scripts
├── go.mod               # module devdeck
├── frontend/            # React desktop UI
├── landing/             # Solid.js marketing site
├── internal/            # Domain packages (see below)
├── docs/                # privacy, analytics plan, this wiki source
├── build/               # Wails build assets / output
├── .github/workflows/   # build.yml (reusable), release.yml, release-beta.yml
├── social/              # Marketing assets
└── tools/               # Misc tooling
```

## Root `*_api.go` files

Keep `app.go` from growing forever by splitting binding groups:

| File | Responsibility |
|------|----------------|
| `ai_api.go` | Ollama list/enrich/chat |
| `ai_commit.go` | AI commit message from git diff context |
| `chat_api.go` | Chat CRUD + `SendChatMessage` |
| `codectx_api.go` | Build/search/clear code index |
| `scripts_api.go` | Detect/run/stop scripts + run history |
| `contribute_api.go` | GitHub issues + diagnostics |
| `opener_api.go` | Finder/Explorer/terminal open |
| `about_api.go` | About pane metadata |
| `analytics.go` / `_ops` / `_kanban` / `_lifecycle` | Track helpers + settings bindings |

## `internal/` packages

| Package | Purpose |
|---------|---------|
| `model` | Shared JSON types: Project, Process, Task, Sprint, Status, Script |
| `store` | Load/save `config.json` (atomic write via `.tmp` + rename) |
| `procman` | Subprocess lifecycle, log buffer, port discovery |
| `detect` | Recursive project/process auto-detection + dotenv parse |
| `deps` | Inspect package managers + install command |
| `scripts` | Discover one-off scripts from manifests (npm, make, go, …) |
| `sysinfo` | System + per-PID CPU/memory snapshots |
| `docker` | Compose/containers/images/volumes via CLI |
| `gitops` | Status, branches, graph log, diff, stash, remote ops |
| `secrets` | OS keychain tokens (`go-keyring`, service `jumpstart`) |
| `release` | Create GitHub/GitLab releases |
| `update` | Check GitHub Releases, download, apply, restart |
| `banner` | Remote announcement banners |
| `ai` | Local Ollama HTTP client |
| `codectx` | Local BM25 code index for chat grounding |
| `chatstore` | Per-project chat sessions under `chats/` |
| `config` | `import.json` parse/merge |
| `testrunner` | Detect and run project tests |
| `analytics` | PostHog client, consent, redact, queue |
| `opener` | Open directory in file manager / terminal |
| `contribute` | Issue draft body + GitHub API helpers |

## Frontend layout

```
frontend/src/
├── main.jsx, App.jsx
├── api.js                 # Re-exports Wails App + runtime helpers
├── analytics.js           # UI → Go TrackEvent bridge
├── ai.js, updateChannel.js, platform.js, …
├── components/            # Feature UI (see Frontend wiki page)
├── hooks/                 # Sidebar width, updates, banners, script runs
├── styles.scss + styles/  # One partial per concern
└── wailsjs/               # Generated — gitignored / produced by Wails
```

## Landing layout

```
landing/src/
├── App.jsx, Home.jsx, index.jsx
├── pages/ (Downloads, Privacy, …)
├── components/ (Hero, Nav, Faq, Contribute, …)
├── analytics.js, engagement.js
└── styles.css
```

Solid.js + Vite; deployed separately (Vercel). Not part of the Wails binary.

## Tests

Go tests sit next to packages:

- `internal/store/store_test.go`
- `internal/detect/detect_test.go`
- `internal/deps/deps_test.go`
- `internal/config/importer_test.go`
- `internal/scripts/scripts_test.go`
- `internal/analytics/*_test.go`
- `internal/update/update_test.go`
- `internal/opener/opener_test.go`
- `ai_commit_test.go`, `analytics_test.go` (main package)

Frontend has no Jest suite; `npm run build` is the compile check.
