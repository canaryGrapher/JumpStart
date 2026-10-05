# Data Model

## Directory: `~/.jumpstart/`

| Path | Purpose |
|------|---------|
| `config.json` | Array of projects (authoritative app state) |
| `config.json.tmp` | Atomic save scratch (renamed into place) |
| `import.json` | Programmatic / pasted import payload |
| `settings.json` | App settings (analytics consent) |
| `mcp.json` | Agents MCP server (enabled, port, bearer token) |
| `install_id` | Random UUID for analytics `distinct_id` |
| `analytics_queue.ndjson` | Offline analytics buffer (≤5000 events) |
| `chats/<projectID>.json` | Story-assistant sessions |
| Code index files | Managed by `internal/codectx` (per project; deleted with project) |

Legacy: if `config.json` is missing, `~/.devdeck/config.json` is copied once.

## `config.json` — Project

See `internal/model/model.go`.

```json
{
  "id": "…",
  "name": "My App",
  "root": "/absolute/path",
  "description": "optional",
  "processes": [ /* Process */ ],
  "tasks": [ /* Task */ ],
  "sprints": [ /* Sprint */ ],
  "tasksEnabled": true,
  "favorite": false,
  "lastUsedAt": 0,
  "useCount": 0,
  "testCommand": ""
}
```

### Process

```json
{
  "id": "…",
  "name": "frontend",
  "dir": "/abs/path",
  "command": "npm run dev",
  "env": { "FOO": "bar" },
  "testCommand": "",
  "scripts": [
    { "id": "…", "name": "Migrate", "command": "go run . --migrate", "source": "package.json" }
  ]
}
```

### Task

- `status`: `backlog` | `todo` | `inprogress` | `testing` | `done`
- `type`: `story` | `task` | `bug` (default task)
- `parentId`: child task → story
- `sprintId`: empty = backlog
- `priority`: `low` | `medium` | `high`
- `subtasks` / `acceptance`: checklist items `{ id, title, done }`
- `createdAt` / `updatedAt`: unix ms

### Sprint

- `status`: `planned` | `active` | `completed`
- `order`: rewritten from slice index on `UpdateSprints`

### GitHub linkage

Set once a task has synced to a GitHub Projects v2 board. See
[GitHub Projects Sync](GitHub-Projects-Sync).

- `github`: `{ itemId, contentId, contentType, number, url, repo, state }` plus
  the sync watermarks `remoteUpdatedAt`, `syncedAt`, and the `conflict` /
  `pending` flags
- `fields`: every Projects v2 field value on the card, keyed by field id
- `milestone`, `reviewers`, `linkedPrs`, `issueType`, `parentKey`: mirrored
  read-only rollups

Per project, `github` holds the link config: board id and title, owner, repo for
new issues, the Status field id and the column → option map, plus `direction`
(`both` | `pull` | `push`) and the last sync stamp.

### Live Status (not persisted)

Returned by `GetStatus` — running flag, PID, ports, startedAt, exitCode (`-1` while running).

## `settings.json`

```json
{
  "analytics": {
    "enabled": true,
    "decidedAt": 1767225600000
  }
}
```

Missing file ⇒ analytics treated as enabled. `decidedAt` set only when the user toggles.

## Frontend-only preferences (localStorage)

| Key area | Examples |
|----------|----------|
| Theme | `theme` = `system` \| `light` \| `dark` |
| Accent | `accent` (e.g. `forest`) |
| Sidebar | open flag, width |
| AI | Ollama host + model (`ai.js`) |
| Updates | channel `stable`/`beta`, snoozed version |
| Banners | dismissed banner IDs |

These never appear in `config.json`. The update channel is pushed to Go via `SetUpdateChannel` so analytics can segment beta users.

## Secrets

GitHub/GitLab tokens: OS keychain, service name `jumpstart`, keys `github_token` / `gitlab_token`. See `internal/secrets`. The Projects sync token is stored under the same `github_token` key, whether it came from the OAuth device flow or a pasted PAT.

## Import format

Either:

```json
[ { "name": "…", "root": "…", "processes": [] } ]
```

or:

```json
{ "projects": [ /* same */ ] }
```

Missing IDs are generated. Merge logic in `internal/config` updates by ID when present.
