# GitHub Projects Sync

A project's Kanban board can sync bidirectionally with a **GitHub Projects v2**
board. Cards move in both directions, every field on the GitHub board is
readable in the task modal, and the writable ones can be edited there.

## Packages

| Piece | Location |
|-------|----------|
| GraphQL client, OAuth, queries | `internal/github` |
| Reconcile engine, conflict rules, polling | `internal/ghsync` |
| Bindings | `github_api.go`, `github_sync_api.go` |
| Link config and per-task linkage | `internal/model/github.go` |
| UI | `frontend/src/components/github/`, `frontend/src/hooks/useGitHubSync.js` |

## Why Projects v2 and GraphQL

Projects v2 has no REST API worth using. Board columns, iterations, and custom
fields exist only in GraphQL, and those are exactly what the sync is for. The
REST issues API would give us titles and labels and nothing else.

## Setting up the OAuth app

The device flow needs an OAuth app client id. **It is not a secret.** The device
flow does not use a client secret at all: the id only says *which app is
asking*, and the user's own browser session is what grants access. GitHub's own
CLI ships its client id in public source.

So there is nothing to inject in CI and nothing to leak from the binary.

1. Go to **Settings → Developer settings → OAuth Apps → New OAuth App**
   (or the org's equivalent for an org-owned app).
2. Fill in the name, homepage, and any callback URL. The device flow never
   redirects, so the callback is only there to satisfy the form.
3. Open the app and tick **Enable Device Flow**. This is required, and it is off
   by default. Without it, `POST /login/device/code` returns an error.
4. Copy the **Client ID** (`Iv1_…` or `Ov23…`) and paste it into
   `GitHubClientID` in `version.go`.

That is the whole setup. Commit the id like any other constant.

A fork or an internal build can point at its own OAuth app without editing the
file:

```sh
wails build -ldflags "-X main.GitHubClientID=Iv23_yourapp"
```

A build that leaves it empty still works. Settings hides the Connect button and
offers a personal access token field instead, which follows exactly the same
code path from `secrets.SaveToken` onward.

### Scopes

`repo project read:org`, requested in `internal/github/auth.go`:

- `project` reads and writes Projects v2 boards.
- `repo` reads issue state and opens new issues.
- `read:org` lists org-owned boards.

## Connecting and linking

Connecting is per install; linking is per project.

- **Settings → GitHub** runs the device flow once and stores the token in the OS
  keychain under `secrets.KeyGitHubToken`, the same place the git tokens live.
- **Tasks → Link a board** binds one JumpStart project to one board. The repo
  field is prefilled from the project's git remote via `github.ParseRemote`.

On link, `EnsureStatusMapping` finds the board's Status field and guesses which
option each Kanban column maps to, using the aliases in `ghsync/mapping.go`
(so "Shipped" or "Icebox" is understood, not just "Done" and "Backlog"). Any
column it cannot place is left unset and editable under **Settings → Edit column
mapping**.

## How "real-time" works

GitHub has no push channel a desktop app can subscribe to, and a webhook would
need a public endpoint. So sync is polled, but only where polling is actually
needed:

| Trigger | Latency |
|---------|---------|
| A local edit (`UpdateTasks`) | Immediate, pushes without waiting for a tick |
| Board focused | ~10s (`ghsync.FocusedInterval`) |
| Window hidden | ~2min (`ghsync.IdleInterval`) |
| After an error | 30s doubling to 15min (`ghsync.MaxBackoff`) |

The direction the user can see is instant; the tick only exists to catch changes
made on github.com. `useGitHubSync` calls `GitHubSetFocused` on
`visibilitychange`, and only one project polls at a time.

Each pass emits `github:sync:start`, then `github:sync:done` (carrying the full
reconciled task list) or `github:sync:error`.

## Reconcile rules

All sync state lives on the task, in `model.GitHubLink`. The engine is stateless
between passes.

- `SyncedAt` — when this task last reconciled cleanly.
- `RemoteUpdatedAt` — the item's `updatedAt` as GitHub last reported it.

`ghsync/conflict.go` compares those two watermarks against `Task.UpdatedAt`:

| Local changed | Remote changed | Outcome |
|---------------|----------------|---------|
| no | no | nothing |
| yes | no | push |
| no | yes | pull |
| yes | yes | conflict, resolved last-write-wins |

A conflict sets `Conflict` on the link, which surfaces as a badge on the card
and a panel in the task modal offering **Keep mine, push it** or **Dismiss**
(`GitHubResolveConflict`). The losing copy is never silently discarded without
that badge appearing.

A task that has never synced counts as a local change, so linking a board
mid-project pushes existing work up rather than dropping it.

### What each side owns

GitHub does not know what a sprint is, so `sprintId`, `parentId`, `subtasks`,
and `acceptance` are never touched by a pull. Everything on the board lands in
`Task.Fields` keyed by field id, so a custom column survives a round trip even
where JumpStart has no native editor for it.

Kanban columns drive the Status field, so Status is deliberately excluded from
the modal's field editors: two controls for one value would fight each other.

### Concurrent edits

A pass can take seconds while the user keeps typing. `runSync` reloads the store
before writing and `mergeConcurrent` keeps any task whose `UpdatedAt` is newer
than the pass start, marking it `Pending` so the next pass pushes it. This is
why a sync never eats a keystroke.

## Field support

Writable, with an editor in the task modal:

`TEXT`, `NUMBER`, `DATE`, `SINGLE_SELECT`, `ITERATION`

Read-only rollups GitHub computes, shown but not editable:

`ASSIGNEES`, `LABELS`, `MILESTONE`, `REPOSITORY`, `REVIEWERS`,
`LINKED_PULL_REQUESTS`, `PARENT_ISSUE`, `SUB_ISSUES_PROGRESS`, `ISSUE_TYPE`,
`TRACKED_BY`

`github.Writable` is the single source of truth for that split; the API rejects
a write to a rollup, so the UI must not offer one.

## Drafts vs issues

New tasks become **draft issues** by default: they need no repository and cost
no issue number. Ticking **Create real issues instead of drafts** at link time
opens a real issue in the configured repo instead. Unlinking a project clears
every task's link but deletes nothing, locally or on GitHub.

## Rate limits

GraphQL is scored per query, not per request. A pass costs one board query plus
one items query per 50 rows. `github.RateLimitError` carries `Retry-After` or
`X-RateLimit-Reset`, and the scheduler backs off on it like any other error.

## Testing

```sh
go test ./internal/ghsync/... ./internal/github/... .
```

The tests cover the parts worth pinning down: conflict direction, column-name
guessing, remote-to-local mapping, and the concurrent-edit merge. Nothing there
touches the network.
