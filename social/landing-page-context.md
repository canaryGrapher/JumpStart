# JumpStart — Landing Page Context

A single source of truth for building the JumpStart landing page. Everything below is drawn from the actual product. Use it to write copy, structure sections, and pick visuals.

---

## 1. One-liner

**One window to start, stop, and monitor every app and project on your Mac.**

Alt versions:
- The native macOS control panel for all your local projects.
- Stop juggling terminal tabs. Run everything from one place.

## 2. What it is

JumpStart is a native macOS desktop app that acts as a control panel for the applications and development projects on your machine. You add a project once, JumpStart detects its runnable parts, and from then on you start, stop, inspect, and organize everything from a single native-feeling window.

Built with Wails, Go, React, and Vite. Ships as a real macOS app bundle.

## 3. The problem (use for hero + pain section)

Anyone running more than one local project lives the same daily ritual:
- `cd` into five different folders every morning
- Re-typing the same start commands over and over
- "Wait, which app grabbed port 3000?"
- Hunting down a zombie process that won't die
- A wall of terminal tabs with no shared view of what's actually running

JumpStart replaces all of that with one window.

## 4. Target audience

Solo founders, freelancers, indie hackers, and hobbyists who run several projects locally and want them in one calm, native place. Not aimed at large teams or CI pipelines; aimed at the individual builder juggling their own machine.

## 5. Positioning & voice

- Voice: builder/authentic, first-person. "I built this to fix my own problem."
- Tone: calm, native, uncluttered. The opposite of terminal chaos.
- Privacy-forward: AI features run on a **local** Ollama model. Code and notes never leave the machine.
- CTA: try it now, free.

## 6. Core features (landing page feature blocks)

**Project library**
Add a project once. Recent and most-used shortcuts keep what you touch daily within reach.

**One-click start / stop**
Start or stop a single subprocess, or every process in a project at once with "Start all."

**Live process visibility**
Real-time status for each process: PID, exit code, streaming logs, CPU usage, and memory usage.

**Automatic port detection**
Ports are detected from logs and `lsof`, then shown as clickable localhost badges. No more `lsof` archaeology.

**Smart project auto-detection**
Points at a folder and finds the runnable parts, including nested monorepos. Supports Node, Go, Python, Ruby, PHP, Java/Maven, Gradle, Rust, and Docker Compose.

**Environment variables & .env import**
Per-process env vars with dotenv import prompts.

**Dependency inspection & install**
Inspects dependencies and offers install actions for common package managers.

**Per-project Kanban board**
Backlog, To Do, In Progress, Done. User stories with child tasks, acceptance criteria, story points, assignees, labels, priorities, subtasks, and progress tracking. Click any card to edit in a detail modal.

**AI task assistant (local)**
"Fill with AI" drafts descriptions, acceptance criteria, subtasks, priority, and labels. A pinned story-assistant chat (expandable to full screen) generates single stories or whole batches. Powered by a local Ollama model; nothing leaves your machine.

**Live port usage table**
See every port in use across all managed processes at a glance.

**Native macOS feel**
Native titlebar behavior, translucent sidebar, appearance preferences, and accent colors.

## 7. How it works (3-step "how" section)

1. **Add a project** — pick a folder. JumpStart auto-detects runnable subprocesses (or add them manually).
2. **Start it** — launch one process or hit "Start all."
3. **Watch everything** — live status, ports, CPU, memory, and logs from one window. Track work on the built-in task board.

## 8. Differentiators / why it matters

- One window instead of many terminal tabs.
- Native macOS app, not a web dashboard or Electron shell.
- Broad language/runtime auto-detection, including monorepos.
- Local-first AI: privacy by default.
- Free to try.

## 9. Suggested page structure

1. Hero: one-liner + subhead (the problem) + primary CTA (download / try free) + hero image.
2. Problem section: the terminal-juggling pain points.
3. How it works: 3 steps.
4. Feature grid: the blocks in section 6 (lead with start/stop, live visibility, ports, auto-detection).
5. AI + Kanban highlight: privacy angle (local Ollama).
6. Native macOS section: screenshots showing the interface.
7. Requirements / who it's for.
8. Final CTA: try it free.

## 10. Requirements (for a footer or FAQ)

- macOS
- For running from source: Go 1.22+, Node.js 18+, Wails CLI, Xcode command line tools
- AI features: a local Ollama install (defaults to `http://localhost:11434`)

## 11. Trust & privacy notes

- Processes run through `/bin/sh -c` in their own process group.
- Stop sends `SIGTERM`, then `SIGKILL` after five seconds.
- Closing JumpStart stops all running managed processes.
- Config stored locally at `~/.jumpstart/config.json`.
- AI runs locally; no code or notes are sent off-device.

## 12. Ready-made copy snippets

Headline options:
- "One window. Every project."
- "Stop juggling terminals. Start building."
- "The control panel your Mac was missing."

Subhead:
- "Add a project once. JumpStart finds everything runnable inside it, then lets you start, stop, and monitor it all from one native window."

CTA buttons:
- "Try JumpStart free"
- "Download for macOS"

Social proof / closing line:
- "Built by a solo dev tired of terminal chaos. Free to try."

## 13. Existing assets

Located in `social/`:
- `jumpstart_hero_landscape.png` (1600×900) — hero / social
- `jumpstart_square.png` (1080×1080) — square / social
- `JumpStart-social-posts.md` — launch copy in the same voice

Recommended additional asset: a 6–10s screen recording of adding a project, hitting "Start all," ports lighting up, and opening the AI task board. Nothing sells a dev tool like the real thing moving.

## 14. Placeholders to fill before launch

- Real download / landing URL (replaces `[LINK]` in social posts)
- Pricing (currently positioned as "free to try")
- App screenshots or GIF of the live UI
