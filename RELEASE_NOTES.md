# JumpStart Release Notes

Paste the relevant section into the GitHub release body for each tag. The
workflow has `generate_release_notes: true`, so GitHub appends an
auto-generated commit/PR changelog below whatever you write here.

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
