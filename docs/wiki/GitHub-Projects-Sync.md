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

### Expiring tokens and refresh

A **GitHub App** client id (`Ov23…`) ships with **Expire user authorization
tokens** switched on by default. Under that setting the device flow returns an
access token good for **8 hours** plus a **refresh token** good for 6 months,
and the access token simply stops working after those 8 hours. An **OAuth App**
client id (`Iv1_…`) with expiry left off returns a token that never expires and
no refresh token. JumpStart handles both.

What is stored, in the OS keychain:

| Key | Contents |
| --- | --- |
| `secrets.KeyGitHubToken` | the bare access token, kept in step on every refresh so git push and release publishing read a current one |
| `secrets.KeyGitHubTokenSet` | the JSON `github.TokenSet`: refresh token plus both expiry instants |

How a token stays fresh:

- `App.ghAccessToken` (`github_token.go`) is the only place a token comes from.
  It refreshes when the access token is within `github.RefreshSkew` (5 minutes)
  of expiring, writes the rotated set back to the keychain, and serializes the
  whole thing behind `ghState.tokenMu` so a burst of concurrent sync requests
  performs one refresh, not one each.
- `github.Client` holds a `TokenSource`, not a token string, so it asks for one
  per request. A 401 triggers one forced refresh and a single retry
  (`internal/github/source.go`); a token GitHub keeps rejecting fails fast
  rather than looping.
- Expiries are stored as absolute instants, so a set read back after the app has
  been closed for a day still knows it is stale.
- A transient network failure during an early top-up is not fatal: the current
  token has not actually expired yet (`TokenSet.Usable`), so it is used and the
  refresh retried on the next call.
- A refusal (`bad_refresh_token`, `invalid_grant`) surfaces as
  `github.ErrReauthRequired`. Only then does the user see "reconnect in
  Settings", and the dead set is deliberately left in the keychain so Settings
  can tell "expired" apart from "never connected".

Pasted personal access tokens and connections made by builds from before this
existed are stored as a set with no refresh token and no expiry, and are used
as-is.

Tests: `internal/github/token_test.go` (lifetimes, refresh, refusals),
`internal/github/source_test.go` (401 retry), `github_token_test.go` (storage,
proactive refresh, PAT and legacy fallback).

### Scopes

`repo project read:org`, requested in `internal/github/auth.go`:

- `project` reads and writes Projects v2 boards.
- `repo` reads issue state and opens new issues.
- `read:org` lists org-owned boards.

## Connecting and linking

Connecting is per install; linking is per project.

- **Settings → GitHub** runs the device flow once and stores the token set in
  the OS keychain under `secrets.KeyGitHubToken` and
  `secrets.KeyGitHubTokenSet`, the same place the git tokens live. From then on
  it is refreshed automatically; see *Expiring tokens and refresh* above.
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
| A local edit (`UpdateTasks`) | Debounced ~3s, then one reconcile |
| Board focused | ~90s (`ghsync.FocusedInterval`) |
| Window hidden | ~10min (`ghsync.IdleInterval`) |
| After an error / rate limit | 30s doubling to 15min (`ghsync.MaxBackoff`), or GitHub's `Retry-After` |

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

`UpdateTasks` also runs `PreserveGitHubLinks`: if the task modal opened before
the first sync finished and Save would otherwise wipe the new link, the prior
`ItemID` is restored (marked pending) so the next pass updates that row
instead of creating a duplicate.

### What each side owns

GitHub does not know what a sprint is, so `sprintId` and `parentId` are never
touched by a pull. Acceptance criteria and subtasks sync through the
issue/draft **body**: JumpStart writes them as GitHub task-list sections wrapped
in `<!-- jumpstart:… -->` markers, and a pull parses those markers back into
the local checklists. Unmarked body prose stays in `Description`.

`Task.Assignee` is a comma-separated list of GitHub logins. Sync **pulls**
assignees from the issue onto the card, and **pushes** them with
`updateIssue(assigneeIds: …)` when the backing content is a real issue.
`Task.Labels` push the same way (`labelIds`), creating missing repo labels as
needed. Drafts cannot take assignees or labels on GitHub: if the card has
either and a repository is linked, sync promotes the draft to an issue first.

Story points push and pull through a numeric Projects field whose name looks
like points / estimate / size.

Kanban columns drive the Status field, so Status is deliberately excluded from
the modal's field editors: two controls for one value would fight each other.

### Concurrent edits

A pass can take seconds while the user keeps typing. `runSync` reloads the store
before writing and `mergeConcurrent` keeps any task whose `UpdatedAt` is newer
than the pass start, marking it `Pending` so the next pass pushes it. This is
why a sync never eats a keystroke.

A task the user **deletes** while a pass is in flight is not resurrected: if it
was in the baseline when the pass started and is gone from the live store,
`mergeConcurrent` drops it and queues its board row on
`GitHubSync.PendingDeletes`.

### Deletes

Deletes sync in both directions when the project direction allows it:

| Action | Effect |
|--------|--------|
| Delete a card in JumpStart | Its Projects item id is queued on `PendingDeletes`. The next pass calls `deleteProjectV2Item` before reading the board, so the row cannot be pulled back as a new card. |
| Delete (or archive-remove) a row on github.com | On a pull/both project, the linked local task is removed. Push-only projects only clear the stale link and keep the local card. |

Failed remote deletes stay on `PendingDeletes` and are retried; those item ids
are also excluded from the "new from board" import path so a stuck delete
cannot bounce the card back into JumpStart.

Unlinking a project clears every task's link but deletes nothing, locally or on
GitHub.

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
opens a real issue in the configured repo instead.

## Rate limits

GraphQL is scored per query, not per request. A pass costs one board query plus
one items query per 50 rows. `github.RateLimitError` carries `Retry-After` or
`X-RateLimit-Reset`; the scheduler sleeps for that duration (capped at
`MaxBackoff`) instead of the usual exponential ladder. Local edits are also
debounced (`pushSyncDebounce`, 3s) so rapid board changes coalesce into one
reconcile rather than one pass per keystroke.

## Testing

```sh
go test ./internal/ghsync/... ./internal/github/... .
```

The tests cover the parts worth pinning down: conflict direction, column-name
guessing, remote-to-local mapping, concurrent-edit merge, delete
propagation, and assignee parse/push helpers. Nothing there touches the
network.
