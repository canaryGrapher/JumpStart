# JumpStart Release Notes

Paste the relevant section into the GitHub release body for each tag. The
workflow has `generate_release_notes: true`, so GitHub appends an
auto-generated commit/PR changelog below whatever you write here.

---

## Installing on macOS (all releases)

The macOS build is ad-hoc signed but not notarized, so the first launch shows a
"could not verify" warning. To open it: click **Done**, then go to **System
Settings → Privacy & Security → Open Anyway** and confirm with Touch ID. This is
a one-time step per machine; JumpStart opens normally afterward.

---

## Unreleased

---

## v1.14.0

Collapsing the sidebar keeps a compact icon rail instead of hiding it
completely — projects, search, and destinations stay one click away.

### Features

- **Collapsed icon rail.** The sidebar toggle shrinks to a narrow rail with
  icons for Search, Dashboard, Ports, each project, Add Project, and Settings.
- **Project tooltips.** Hover a project icon to see its full name.
- **Search from the rail.** The search icon expands the sidebar and focuses
  the search field so you can filter without hunting for the toggle.

### Improvements

- Terminal dock, update banner, and layout offsets track the rail width so
  chrome stays aligned when collapsed.
- Favorites and the rest of the project list stay separated by a light divider
  in the rail.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.14.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.14.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.14.0_linux-amd64.tar.gz` |

### Upgrade notes

- Install over v1.13.x. No config migration. Existing `sidebarOpen` preference
  still applies; collapsed now means the icon rail rather than a fully hidden
  sidebar.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.13.0...v1.14.0

---

## v1.13.0

Browse a GitHub-compatible project wiki inside JumpStart — a local stand-in
for GitHub Wiki on free private repos.

### Features

- **In-app Wiki tab.** When a project has a local wiki, a new **Wiki** tab
  opens a GitHub-style page viewer with a **left** pages sidebar (GitHub puts
  that list on the right; JumpStart keeps navigation on the left).
- **GitHub-compatible layout.** Renders `_Sidebar.md` when present (otherwise
  an auto page list), the selected page as GFM markdown, and optional
  `_Footer.md`. Wiki-style links navigate in-app; `http(s)` links open in the
  browser.
- **Local wiki locations.** Detects the first match among `project/.wiki/`,
  `project/docs/wiki/`, or a sibling `ProjectName.wiki/` clone.

### Improvements

- Developer wiki docs updated (Features, Frontend, Backend-API, Code-Structure)
  for `WikiInfo` / `WikiPage` and the new UI.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.13.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.13.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.13.0_linux-amd64.tar.gz` |

### Upgrade notes

- Install over v1.12.x. No config migration. Add a `.wiki` (or `docs/wiki`)
  folder with `Home.md` to enable the tab on any project.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.12.0...v1.13.0

---

## v1.12.0

GitHub as the single source of truth for task sync, atomic conflict reasons, a
colorful Dashboard, and a richer Ports map.

### Features

- **GitHub SSOT sync.** Quiet remote changes always pull. Local edits wait as
  pending and flush in a **10s clubbed** batch that only pushes dirty cards.
- **Atomic conflict reasons.** Conflicts fire only when field values diverge
  (not on timestamp echoes). Each conflict lists title/status/labels/etc. with
  yours vs GitHub values. Resolve with **Accept GitHub** or **Overwrite GitHub**
  (per card or in bulk).
- **Infographic Dashboard.** Colorful KPI tiles, a Kanban flow pipeline, completion
  donut, activity bars, and live port stats at a glance.
- **Ports redesign.** Stats tiles, conflict highlighting, per-project grouping,
  and a guided empty state when nothing is listening yet.

### Improvements

- Pending badge reads as “Waiting for GitHub” during the batch window.
- Sync docs updated for SSOT rules, content-diff conflicts, and the 10s flush.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.12.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.12.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.12.0_linux-amd64.tar.gz` |

### Upgrade notes

- Install over v1.11.x. Linked boards keep working; existing conflict badges clear
  on the next sync under the new content-diff rules.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.11.2...v1.12.0

---

## v1.11.2

GA4 Realtime and active-user reporting for desktop Measurement Protocol events.

### Fixes

- **Analytics visible in GA4.** Measurement Protocol events now send a numeric
  `session_id` (Unix seconds at launch), `engagement_time_msec` on every event,
  and a `session_start` on launch — the shapes GA4 needs for Realtime and
  active-user tiles. Earlier releases could POST successfully while Home /
  Realtime stayed empty because `session_id` was a UUID and engagement time
  was missing.
- **Failed MP responses retry.** Non-success Measurement Protocol responses
  (including 4xx) are queued for the next launch instead of being dropped.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.11.2_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.11.2_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.11.2_linux-amd64.tar.gz` |

### Upgrade notes

- Install this build over v1.11.1. Keep Privacy → Share anonymous usage data on,
  open the app for ~15 seconds, then check **GA4 → Reports → Realtime** for
  `session_start` / `app_launched`. The Home “no data from your website” banner
  can still appear — that refers to web tagging, not the desktop app.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.11.1...v1.11.2

---

## v1.11.1

Custom Kanban columns, bulk GitHub conflict resolve, and a critical fix so
release builds finally send product analytics to GA4.

### Features

- **Custom board columns.** Add a column from the Tasks board and map it to an
  existing GitHub Status option, or create a new Status on the board from
  JumpStart. Column mapping under GitHub settings supports the same flow.
- **Create Status options from JumpStart.** Edit column mapping can add a
  Status option on the linked board (Create next to an unset column, or New
  status on board) and map it to a Kanban column such as Testing.
- **Bulk resolve conflicts.** When several cards conflict with GitHub, Resolve
  conflicts lists them so you can Keep mine or Dismiss for many at once.
- **Appearance picker.** Settings → General offers Light, Dark, or Auto
  (follow system).

### Improvements

- **Testing column auto-maps on existing boards.** Sync fills in a missing
  Testing→Status mapping when the board already has a Testing/QA option,
  without rewriting hand-tuned column maps.
- **Agents client configs.** Settings → Agents can copy Cursor, Claude Desktop
  (stdio + `mcp-remote`), Claude Code, or Codex snippets. Claude Desktop docs
  clarify that localhost `http://` connectors are rejected on purpose.
- **Landing polish.** Hero and marketing sections refreshed for the current
  product story.

### Fixes

- **Release analytics actually send.** Stable and beta release workflows now
  pass `secrets: inherit` into the reusable build. Without that,
  `DESKTOP_GA_API_SECRET` was empty in every official binary, so GA4 never
  received events even though the Measurement ID was stamped. Install this
  build (or newer) for usage data to appear in the Jumpstart (App) property.
- **Keep mine stays kept.** Choosing Keep mine, push no longer lets the
  follow-up sync re-raise the Conflict badge when GitHub’s timestamp is
  slightly ahead of local time.
- **Fewer false GitHub conflicts.** Local task saves merge watermarks by task
  ID so a stale full-board save cannot undo a reconcile and flag ordinary
  edits as conflicts.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.11.1_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.11.1_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.11.1_linux-amd64.tar.gz` |

### Upgrade notes

- No project data migration. Existing boards keep the default five columns until
  you add a custom one (defaults are materialized then).
- After upgrading, GA4 Realtime should show events within a minute if Privacy →
  Share anonymous usage data is on.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.11.0...v1.11.1

---

## v1.11.0

AI agents can drive JumpStart over a local MCP server, and the Tasks board
gains a Testing column that syncs with GitHub Projects.

### Features

- **Agents MCP.** Settings → Agents starts a localhost Model Context Protocol
  server so Cursor and other AI agents can list projects, start/edit
  processes, read/write files inside project roots, inspect git changes, and
  manage kanban tasks. Disabled by default; bearer-token auth on `127.0.0.1`.
- **Testing column.** The Tasks board adds a Testing stage between In Progress
  and Done. GitHub Projects sync maps common QA/Testing option names, and new
  board presets include a Testing column.

### Improvements

- **MCP connect docs.** Website `#/docs/mcp` and the GitHub Wiki Agents-MCP
  page cover Cursor, Claude Code/Desktop, Codex, ChatGPT desktop, Hermes,
  Paperclip, and other harnesses.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.11.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.11.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.11.0_linux-amd64.tar.gz` |

### Upgrade notes

- No project data migration. Existing tasks keep their status; drag into
  Testing when ready. Linked GitHub boards need a Testing/QA Status option
  (or a manual column map) before that column syncs both ways.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.10.0...v1.11.0

---

## v1.10.0

PDF export can render the board as real kanban columns, project chrome
collapses to give the Tasks view more room, and GitHub sync stops raising
false conflict badges after a push.

### Features

- **Kanban PDF board layout.** Import/Export → PDF offers Board or Table.
  Board draws Backlog / To Do / In Progress / Done columns with typed cards,
  priority chips, and multi-page overflow.
- **Sprint-scoped CSV transfer.** Import and export can target one sprint
  board (or Backlog) instead of always moving the whole project; sample
  CSV/guide downloads use a native Save dialog.
- **Collapsible project chrome.** The project header (Start/Stop, open-in
  tools, Edit/Delete) and Progress & sync strip collapse independently,
  with open/closed remembered per project.
- **Task context menu.** Right-click a card to open, edit, or delete.

### Fixes

- **False GitHub conflicts after push.** Remote watermarks include a short
  grace window so the next local edit is not treated as a diverge-from-echo
  conflict; resolving a conflict clears the badge in the open modal immediately.
- **Darwin UI readable in dark mode.** The document root toggles `.dark` for
  Tailwind variants, and the global CSS reset no longer zeroes padding on
  every element (which had flattened dialog/button spacing).

### Improvements

- **Clearer open-in icons.** Finder, Terminal, VS Code, and GitHub controls
  use larger plates with stronger contrast on dark chrome.
- **Lighter AI chat entry.** SprintBar exposes AI; the chat dock no longer
  reserves scroll space under the board.
- **pnpm install guards.** Workspace allow-builds / overrides keep `wails`
  frontend installs working across pnpm 10 and 12.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.10.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.10.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.10.0_linux-amd64.tar.gz` |

### Upgrade notes

- No project data migration.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.9.1...v1.10.0

---

## v1.9.1

AI task fill is more reliable, Import/Export can save filtered boards as
Excel/PDF/PNG, and task fields stay aligned across prompts, sample CSVs,
and the editor — including editable acceptance criteria on every type.

### Fixes

- **AI task generation survives messy model JSON.** Populate-with-AI and chat
  story drafts tolerate string/bool acceptance lists, task `description`
  instead of `title`, and code fences. Embedding-only Ollama models are hidden
  from the picker.
- **Task field catalog stays aligned.** Import-config Copy prompt, AI enrich/
  chat prompts, and sample CSVs share the same 22 columns as CSV import
  (including `parentId`, sprint, acceptance on any type, reviewers, etc.).

### Features

- **Task sheet download.** Import/Export can save the board as Excel, PDF,
  PNG, or CSV, with filters for columns (boards/views), sprints, types,
  labels, and priority.
- **Editable acceptance criteria.** Acceptance criteria show for every task
  type and can be edited inline; subtask titles are editable too.

### Improvements

- **Equal-height process cards.** Cards in a row stretch to match the tallest
  sibling, with footers pinned to the bottom.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.9.1_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.9.1_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.9.1_linux-amd64.tar.gz` |

### Upgrade notes

- No project data migration.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.9.0...v1.9.1

---

## v1.9.0

GitHub sync stops burning the rate limit, labels stick around after a
reconcile, and the Tasks import flow ships a sample guide. The import modal
and sync bar move onto Darwin UI.

### Fixes

- **GitHub sync no longer hammers the API.** Focused polling is ~90s (was
  10s), idle is ~10min, local edits are debounced (~3s), and rate-limit
  responses honour GitHub's `Retry-After`.
- **Labels survive sync.** Draft pulls no longer wipe local labels/assignees;
  unresolved label ids no longer clear the issue; post-push echo pulls no
  longer treat our own write as a remote change.

### Features

- **Bulk import sample guide.** The Tasks Import tab offers Sample guide /
  Sample CSV downloads so you can fill a spreadsheet offline and import it.
- **Darwin UI.** Import/export modal and GitHub sync bar use
  [@pikoloo/darwin-ui](https://darwin-ui.mandalsuraj.com) (React 19).

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.9.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.9.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.9.0_linux-amd64.tar.gz` |

### Upgrade notes

- Frontend now requires React 19 (bundled). No project data migration.
- Linked boards sync less often in the background; use **Sync now** for an
  immediate pass.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.8.0...v1.9.0

---

## v1.8.0

Process, script, and test output moves out of the card and into a
Windows-style terminal dock at the bottom of the window.

### Features

- **Terminal dock.** Clicking "Logs" on a process, running a script, or
  running a test no longer prints output inline on the card — it pops up as
  a floating window above a taskbar fixed to the bottom of the screen.
  Click a tab to minimize or restore its window; click the tab's close
  button to dismiss it. Multiple runs can be open (or minimized) at once,
  each with its own tab.

### Improvements

- **Script run history.** The run-history list on a process card is now a
  compact chip list (name, time, status dot) that reopens a past run's
  terminal window in the dock, instead of expanding its log inline.
- **Test runs.** A "View log" button reopens a finished test run's output
  in the dock without re-running it.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.8.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.8.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.8.0_linux-amd64.tar.gz` |

### Upgrade notes

- No config or data changes. Existing projects, processes, and scripts work
  as before — only where their output is shown has changed.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.7.2...v1.8.0

---

## v1.7.2

GitHub Projects sync now respects deletes in both directions, and task
assignees are a team multi-select that syncs with the backing issue.

### Fixes

- **Deleted tasks stay deleted.** Removing a card in JumpStart queues the
  Projects row for `deleteProjectV2Item` before the next reconcile, so sync
  cannot pull it back as a new card. Removing a row on github.com drops the
  linked local task (on pull/both projects). Deletes made while a sync pass is
  in flight are kept and flushed afterward via `PendingDeletes`.

### Features

- **Assignees multi-select.** The task modal lists assignable users from the
  linked repository (searchable chips + checklist, GitHub-style). Assignees
  work on every card type, not only stories, and show as `@login` pills on the
  board.

### Improvements

- **Assignee sync both ways.** Pull copies issue assignees onto `task.assignee`
  (comma-separated logins). Push writes them back with `updateIssue(assigneeIds)`
  when the card is a real issue. Drafts still cannot take assignees on GitHub
  until promoted.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.7.2_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.7.2_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.7.2_linux-amd64.tar.gz` |

### Upgrade notes

- After deleting synced cards, leave the board focused briefly (or hit Sync)
  so pending remote deletes can flush.
- Custom typed assignee names that are not GitHub users stay local-only and
  are skipped on push.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.7.1...v1.7.2

---

## v1.7.1

CSV import/export polish: richer columns, Add vs Replace import, sprint names
that create missing sprints (including on GitHub), and a restored dock icon.

### Features

- **CSV Add / Replace import.** Choose **Add** to merge an incremental CSV
  (matching ids update; new rows create; cards missing from the file stay) or
  **Replace** to treat the spreadsheet as the full board and remove anything
  not listed.
- **Human-readable `sprint` column.** Exports include the sprint name next to
  `sprintId`. On import, unknown names create a local planned sprint. When the
  board is linked to GitHub, missing Iteration cycles are created on the
  Projects v2 board (or a new Sprint iteration field if the board has none).

### Improvements

- **Richer CSV columns.** Export/import now round-trips checklists
  (`subtasks`, `acceptance`), timestamps, `milestone`, `issueType`,
  `parentKey`, `reviewers`, and `linkedPrs`, alongside the original fields.
- **Dock icon restored.** `build/appicon.png` was accidentally overwritten with
  the default Wails “W” in v1.7.0; the teal rocket icon is back for macOS Dock
  and Finder.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.7.1_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.7.1_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.7.1_linux-amd64.tar.gz` |

### Upgrade notes

- Prefer the `sprint` column for board membership in spreadsheets; leave it
  blank or set `Backlog` to clear sprint assignment.
- **Replace** import deletes local cards that are not in the file — export a
  backup first if you are unsure.
- Reinstall or update so the corrected app icon replaces any cached Dock
  artwork from v1.7.0.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.7.0...v1.7.1

---

## v1.7.0

Bulk task CSV import/export, live progress on long GitHub and install
operations, and clearer reconnect flows when tokens expire.

### Features

- **Import / Export tasks as CSV.** From the Tasks tab, open **Import / Export**
  to download the board as a spreadsheet-friendly CSV (optionally filtered by
  status, sprint, type, label, or priority) or upload edits back. Matching IDs
  update existing cards; new rows create tasks. Import is upsert-only — cards
  missing from the file are left alone.
- **Live progress on GitHub sync and board import.** Syncing local cards to a
  GitHub Project and importing repo issues/PRs into a new board show
  `Syncing 3/40…` / `Importing 3/13…` instead of a stuck Working… state.
  Initial link passes and large imports also get a longer timeout so they can
  finish while progress keeps updating.
- **Dependency install progress.** Installing packages on a process shows a
  live meter (and a short success strip) instead of a raw log terminal.

### Improvements

- **GitHub and GitLab reconnect.** When a token is rejected, Settings → Accounts
  keeps a **Reconnect** banner until you finish signing in again (device flow
  or fresh PAT), instead of hiding the CTA after a failed attempt.
- **Update notice.** The in-app update banner is a compact bottom-left notice
  with clearer progress, release notes, and dismiss/restart actions. Release
  note links open in the system browser.
- **Kanban layout.** Tracker / board sticky height math is simpler so the sprint
  bar and search stay pinned without clipping column headers.
- **Frontend tooling.** Local and CI builds use pnpm (`wails.json` + lockfile);
  tracked `landing/node_modules` is removed from the repo.
- App icon artwork refreshed.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.7.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.7.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.7.0_linux-amd64.tar.gz` |

### Upgrade notes

- CSV import merges by task `id`; export a board first if you want stable IDs
  for round-trip edits in a spreadsheet.
- Existing GitHub / GitLab connections carry over. If sync starts failing with
  a rejected token, use **Reconnect** under Settings → Accounts.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.6.0...v1.7.0

---

## v1.6.0

Account settings for GitHub and GitLab, richer profile cards, and a privacy-first
analytics option.

### Features

- **GitHub and GitLab in Settings → Accounts.** Connect either provider via browser
  sign-in or a personal access token. Each account shows a profile card (avatar,
  name, username) and a disconnect action.
- **Analytics privacy: None detail level.** Settings → Privacy now offers a
  *None* detail level that turns off product analytics entirely while keeping
  the rest of the app unchanged.
- **About page branding.** The About screen uses JumpStart branding and moves
  the privacy policy link to a clearer spot.

### Improvements

- Preferences modal is larger with titled tabs for easier navigation.
- Contribute issue list simplified; settings actions aligned across screens.
- Card grids use a fixed two-column layout with mobile stacking.

### Fixes

- GitLab client ID and frontend API exports wired correctly for browser sign-in.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.6.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.6.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.6.0_linux-amd64.tar.gz` |

### Upgrade notes

- Existing GitHub connections carry over; no reconnect needed.
- To add GitLab, open **Settings → Accounts** and sign in or paste a token.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.5.1...v1.6.0

---

## v1.5.1

Automatic GitHub token refresh, so board sync and repo linking stop asking you
to reconnect every few hours.

### Fixes

- **GitHub connection no longer expires every 8 hours.** The GitHub App used
  for Settings → GitHub ships with token expiry on: the device flow returns an
  access token good for 8 hours and a refresh token good for 6 months, but
  JumpStart was only keeping the access token, so sync and the repo wizard
  started failing with "GitHub rejected the token, reconnect in Settings"
  every few hours. The access token is now refreshed automatically shortly
  before it expires, and any request that still comes back 401 (say, after the
  laptop was asleep) triggers one forced refresh and retry before it's shown
  to you as an error.

### Internals

- New `internal/github` token/source layer: `TokenSet` (access + refresh +
  absolute expiry instants, so a stored set is still known-stale after the app
  has been closed for a day), `RefreshAccessToken`, and a `TokenSource` the
  client asks for a token per request instead of holding one.
- `App.ghAccessToken` (`github_token.go`) is now the single place a GitHub
  token comes from; refreshes are serialized so a burst of concurrent sync
  requests only refreshes once, and both keychain entries
  (`KeyGitHubToken`, `KeyGitHubTokenSet`) are kept in step so git push and
  release publishing always read a current token.
- Pasted personal access tokens and connections made by older builds still
  work as-is; a refresh failure GitHub can't recover from (`bad_refresh_token`,
  `invalid_grant`) is the only case that surfaces "reconnect in Settings".
- Documented in `docs/wiki/GitHub-Projects-Sync.md` under *Expiring tokens and
  refresh*; covered by `internal/github/token_test.go`,
  `internal/github/source_test.go`, and `github_token_test.go`.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.5.1_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.5.1_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.5.1_linux-amd64.tar.gz` |

### Upgrade notes

- Existing users will see the in-app update banner; no manual reconnect to
  GitHub is needed, the next sync after updating just starts working again.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.5.0...v1.5.1

---

## v1.4.0

Granular analytics privacy controls, remaining product events, auto-download
updates, and release builds that actually ingest to PostHog.

### Highlights

- **Settings → Privacy detail levels.** Full, Balanced, and Minimal presets,
  plus per-category toggles (lifecycle, onboarding, processes, Git & Docker,
  kanban, AI, updates, UI panels). Everything stays on by default; turning
  the master switch off still clears the offline queue.
- **PostHog-shaped frontend bridge.** `capture` / `optIn` / prefs helpers in
  the webview still talk only to Go — no `posthog-js`, no second identity.
- **Release builds emit analytics.** CI stamps `POSTHOG_API_KEY` from the
  GitHub Actions variable into the binary (US cloud by default).
- **Auto-download updates.** When a newer release is found, the banner
  downloads in the background; you only confirm Restart.

### Added

- Category map and gating in `internal/analytics` (`category.go`); prefs in
  `settings.json` (`detailLevel` + `categories`).
- Events: `logs_opened`, `ports_viewed`, `env_file_edited`, `roadmap_opened`.
- `scripts/setup-posthog-dashboards.sh` to create the six product dashboards
  (run locally with a personal PostHog API key).
- Install mutex so banner and Settings cannot race `InstallUpdate`.

### Changed

- Privacy UI expanded beyond a single switch; privacy policy and wiki updated.
- Update banner starts download automatically; dismiss still snoozes Restart.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.4.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.4.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.4.0_linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.3.x users will see the in-app update banner; download starts
  automatically, then restart when ready.
- Analytics remains on by default. Use Settings → Privacy to lower detail
  level or disable categories without turning collection off entirely.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.3.0...v1.4.0

---

## v1.3.0

Privacy-first product analytics, About branding, open-in Finder/Terminal,
AI commit messages, and a full developer wiki.

### Highlights

- **PostHog analytics from the Go process.** GA4 and Microsoft Clarity are
  gone from the desktop app (they remain on the landing site only). Every
  user action already crosses the Wails binding boundary, so instrumentation
  lives there — events carry real outcomes and durations, work offline via a
  disk queue, and never leave the machine when the PostHog key is absent
  (local/fork builds).
- **Settings → Privacy.** One switch controls anonymous usage reporting.
  Off stops collection immediately and deletes anything still buffered on
  disk. Policy: [`docs/privacy.md`](docs/privacy.md).
- **Settings → About.** Version, build date, Workvar vendor links, platform,
  and Go version in one pane.
- **Open in Finder / Terminal.** Project header actions open the project
  root in the host file manager or a terminal (labels adapt per OS).
- **AI commit messages.** Git commit box can draft a message from the local
  Ollama model using a redacted diff context — nothing leaves your machine.
- **Developer wiki.** Architecture, API, data model, contribution guides,
  and how to publish remote banners:
  https://github.com/canaryGrapher/JumpStart/wiki

### Added

- `internal/analytics` — consent, redaction, offline queue, PostHog transport;
  CI injects `PostHogAPIKey` / `PostHogHost` via ldflags (empty = no-op).
- `internal/opener` + Open Actions UI for file manager / terminal.
- About API and UI; Workvar product metadata in `version.go` / `wails.json`.
- AI commit helper (`ai_commit.go`, `internal/gitops/commitctx.go`) and
  `CommitBox` UI.
- `docs/wiki/` source + `scripts/publish-wiki.sh` to sync the GitHub Wiki.
- Landing-site engagement: page views, scroll depth, section visibility,
  exit summary, and named CTA events (still separate from desktop PostHog).

### Changed

- Frontend analytics is a thin bridge to Go (`TrackEvent*`); no GA/Clarity
  in the desktop bundle.
- Release build stamps `BuildDate` alongside `Version` for the About pane.
- Landing Privacy page and FAQ copy aligned with the desktop privacy model.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.3.0_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.3.0_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.3.0_linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.2.x users will see the in-app update banner and can update without
  a manual download.
- Analytics is on by default in release builds; turn it off anytime under
  Settings → Privacy. Builds without a PostHog key send nothing.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.5...v1.3.0

---

## v1.2.5

Fixes the in-app version number so it matches the release you downloaded.

### Fixes

- **Correct in-app version.** The app's version was hardcoded in the binary and
  hadn't been bumped for 1.2.4, so downloads reported the wrong number (1.2.4
  showed as 1.2.3) and the updater could offer a build an "update" to itself.
  The release build now injects the git tag into the binary at build time
  (`-ldflags -X main.Version`), so the About window and updater always match the
  tag automatically. No more manual version bumps to keep in sync.

### Internals

- `main.Version` is now a build-time variable (defaults to `dev` for local
  builds); the release workflow stamps it from the tag, and `GetAppVersion`
  trims the leading `v` for display.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.2.5_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.2.5_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.2.5_linux-amd64.tar.gz` |

### Upgrade notes

- If you're on the mislabeled 1.2.4 build, update to 1.2.5 to see the correct
  version reported in-app.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.4...v1.2.5

---

## v1.2.4

Colored process logs, plus new legal pages on the site.

### Highlights

- **Colored terminal logs.** Process log panels now parse ANSI color and bold
  escape sequences, so output from dev servers and CLIs (Vite, npm, test
  runners, etc.) renders in color instead of showing raw escape codes as
  garbage. Supports the standard 8 colors, bright variants, bold, and reset.
- **Privacy Policy and Terms of Use.** The landing site now has proper legal
  pages, linked from the footer. They reflect JumpStart's local-first design,
  explain the website analytics (GA4, Microsoft Clarity, Vercel), and state
  plainly that data is never sold, only studied to improve the product.

### What's new and internals

- New ANSI SGR parser (`frontend/src/ansi.js`) wired into the log panel; each
  line is split into styled segments instead of a raw string.
- Landing site: hash-routed `#/privacy` and `#/terms` pages with a shared
  layout and themed styling; footer Legal links now point to them, and a
  FoundrList verification badge was added to the footer.
- Expanded landing-page documentation and added marketing assets and social
  media guides to the repo.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.2.4_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.2.4_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.2.4_linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.2.x users will see the in-app update banner and can update without
  a manual download.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.3...v1.2.4

---

## v1.2.3

Bug fixes for launched-from-Finder process spawning and dark-mode UI.

### Fixes

- **Processes now find your tools when launched from Finder.** Apps started
  outside a terminal inherited a minimal `PATH`, so commands like `npm` failed
  with "command not found" even when installed. JumpStart now resolves your real
  login-shell `PATH` (homebrew, nvm, asdf, etc.) and injects it into every
  process it spawns, matching how commands run in your terminal.
- **Readable announcement and update banners in dark mode.** The bottom-right
  announcement card and the update banner referenced undefined color tokens and
  fell back to a white background, rendering near-white text unreadable in dark
  mode. They now use the themed surface and text colors in both light and dark.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.2.3_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.2.3_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.2.3_linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.2.x users will see the in-app update banner and can update without
  a manual download.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs
  a one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.2...v1.2.3

---

## v1.2.2

Release packaging and versioning polish.

### Highlights

- **Versioned download filenames.** Release archives are now named
  `jumpstart_v<version>_<platform>.<ext>` (e.g.
  `jumpstart_v1.2.2_macos-universal.zip`), so downloads from the landing page
  and GitHub clearly show the version and extension.
- **Correct in-app version on macOS.** The build now stamps the release version
  into the app bundle's `Info.plist`, so Finder "Get Info" and the About window
  show the real version instead of `1.0.0`.

### What's new

- Homebrew tap and cask (`brew install --cask canaryGrapher/jumpstart/jumpstart`)
  that installs to `/Applications`, clears quarantine, and keeps the self-updater
  working.

### Fixes and internals

- Renamed CI packaging outputs, the landing-page fallback URL, and the cask to
  the `jumpstart_v<version>_<platform>` convention. The updater and landing
  buttons match assets by platform keyword, so self-update is unaffected.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart_v1.2.2_macos-universal.zip` |
| Windows (x64) | `jumpstart_v1.2.2_windows-amd64.zip` |
| Linux (x64) | `jumpstart_v1.2.2_linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.2.x users will see the in-app update banner and can update without
  a manual download.
- macOS builds remain ad-hoc signed but not notarized; first launch still needs a
  one-time **System Settings → Privacy & Security → Open Anyway** approval.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.1...v1.2.2

---

## v1.2.1

Maintenance release focused on a clean, predictable macOS install experience.
Downloaded builds now land in the recoverable "Open Anyway" state instead of the
"app is damaged" dead-end, and the first-launch steps are documented.

### Highlights

- **Ad-hoc signed macOS builds from CI.** The release workflow now signs
  `JumpStart.app` (with a hardened runtime) and verifies the signature before
  packaging. This keeps every download openable via Gatekeeper's "Open Anyway"
  flow rather than the "app is damaged / Move to Bin" hard block, with no Apple
  Developer account required.
- **Documented one-time launch step.** The README and release notes now walk
  users through the first launch: **Done → System Settings → Privacy & Security
  → Open Anyway**, a single approval per machine.

### What's new

- New `Ad-hoc code sign (macOS)` step in the release workflow runs
  `codesign --deep --force --options runtime --sign -` and then
  `codesign --verify --deep --strict` so a broken signature fails the build
  instead of shipping.
- New **Installing the macOS release** section in the README with the exact
  Gatekeeper steps.
- Reusable **Installing on macOS (all releases)** block at the top of
  `RELEASE_NOTES.md` so the guidance applies to every future tag.

### Fixes and internals

- Prevents the intermittent "JumpStart is damaged and can't be opened" dialog
  caused by an unsigned or inconsistent bundle; macOS now shows the recoverable
  "could not verify" warning instead.
- macOS packaging keeps using `ditto -c -k --keepParent` so the bundle's
  signature and symlinks survive zipping.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart-v1.2.1-macos-universal.zip` |
| Windows (x64) | `jumpstart-v1.2.1-windows-amd64.zip` |
| Linux (x64) | `jumpstart-v1.2.1-linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.2.x users will see the in-app update banner and can update without
  a manual download.
- macOS builds are ad-hoc signed but still not notarized. First launch requires
  a one-time approval via **System Settings → Privacy & Security → Open Anyway**;
  the self-updater preserves the ad-hoc signature when swapping the bundle.

**Requirements:** macOS 11+, Windows 10/11 (WebView2), or a modern Linux
desktop with WebKit2GTK. AI features use a local Ollama model if present.

**Full Changelog**: https://github.com/canaryGrapher/JumpStart/compare/v1.2.0...v1.2.1

---

## v1.2.0

Cross-platform, self-updating, and instrumented. This release brings JumpStart
to Windows and Linux, adds a real in-app updater, and wires up product
analytics for both the app and the website.

### Highlights

- **Windows and Linux builds.** JumpStart now ships signed-free universal
  macOS, Windows (amd64), and Linux (amd64) binaries from CI. The landing page
  has a download button per OS.
- **In-app self-update.** When a newer release exists, a bar slides up from the
  bottom of the window. "Update now" downloads the correct build for your
  platform, shows live download progress, and swaps it in place (whole `.app`
  bundle on macOS; executable replace on Windows/Linux). "Restart now"
  relaunches into the new version. No more manual reinstall.
- **Product analytics (GA4 + Microsoft Clarity).** The desktop app and the
  landing site each report to their own GA4 property and Clarity project.
  Events fire to both platforms; event parameters become Clarity tags for
  filtering.

### What's new

- App update banner with `Update now` / progress / `Restart now` states, plus a
  manual-download fallback on error.
- Backend self-update engine (`internal/update`): fetches the platform release
  asset, downloads with progress events, and atomically replaces the running
  app.
- Analytics event taxonomy in the app: `app_installed`, `app_open`,
  `process_start`/`process_stop` (+ start_all/stop_all), `script_run`,
  `tests_run`, `deps_install`, `project_saved`/`project_deleted`,
  `config_imported`, `git_connected`, `git_commit`, `git_push`,
  `release_created`, `compose_up`/`compose_down`. User properties
  `project_count` and `process_count` for segmentation.
- Landing analytics: per-OS `download` events, `releases_redirect`, and
  `outbound_github`.
- Redesigned download UX on the landing page: macOS/Windows/Linux buttons with
  OS logos, and a themed "all releases" link.

### Fixes and internals

- Release workflow now packages the actual Wails output names and publishes
  assets as `jumpstart-<tag>-<platform>` for all three platforms (Windows and
  macOS packaging previously failed).
- Removed the redundant `build-windows.yml` workflow.
- Analytics IDs are injected at build time from repository variables
  (`VITE_GA_ID`, `VITE_CLARITY_ID`); absent values simply disable analytics.

### Downloads

| Platform | Asset |
| --- | --- |
| macOS (universal) | `jumpstart-v1.2.0-macos-universal.zip` |
| Windows (x64) | `jumpstart-v1.2.0-windows-amd64.zip` |
| Linux (x64) | `jumpstart-v1.2.0-linux-amd64.tar.gz` |

### Upgrade notes

- Existing 1.1.x users will see the in-app update banner and can update without
  a manual download.
- macOS builds are not code-signed or notarized. The self-updater swaps the
  whole bundle to keep its ad-hoc signature valid; on locked-down Macs you may
  still need to allow the app on first launch.
- Self-update requires write access to the app's install location. If it can't
  write there, the banner falls back to a manual download link.

**Requirements:** macOS 11+, Windows 10/11 (WebView2), or a modern Linux
desktop with WebKit2GTK. AI features use a local Ollama model if present.

---

## v1.1.0

First public release of JumpStart: a native macOS control panel for the
applications and development projects on your machine. Add a project once, let
JumpStart detect its runnable parts, then start, stop, inspect, and organize
everything from one window.

### Highlights

- **Process control.** One-click start/stop for individual subprocesses or a
  whole project, with live status, PID, exit code, logs, CPU, and memory.
- **Automatic port detection.** Ports are pulled from logs and `lsof` in real
  time and shown as clickable localhost badges, with a live port-usage table
  across all managed processes.
- **Project auto-detection.** Detects runnable parts in nested monorepos across
  Node, Go, Python, Ruby, PHP, Java/Maven, Gradle, Rust, and Docker Compose.
- **Ship without leaving the app.** Built-in Git panel (status, branch,
  ahead/behind, stage, commit, fetch, pull, push) with tokens stored in the OS
  keychain, plus one-click tagged releases to GitHub or GitLab, a test runner,
  and Docker/Compose management.
- **Per-project Kanban with local AI.** Stories, tasks, and bugs with labels,
  priorities, story points, and subtasks; "Fill with AI" and a story-assistant
  chat powered by a local Ollama model, so nothing leaves your machine.

### Also included

- Per-process environment variables and dotenv import prompts.
- Dependency inspection and install actions for common package managers.
- JSON/config import via an interactive block builder, pasted JSON, or a file.
- Background update checks with an in-app banner.
- Native macOS interface: native titlebar, translucent sidebar, system
  appearance support, and accent colors.

### Data locations

- Config: `~/.jumpstart/config.json`
- Programmatic imports: `~/.jumpstart/import.json`
- Git tokens: OS keychain (never written to disk in config)

**Requirements:** macOS. AI features require a local Ollama install.
