# macOS-Native Theme Conversion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle DevDeck (a Wails v2 + React desktop app) to follow Apple's Human Interface Guidelines for Mac apps — native window chrome with vibrancy, a unified macOS sidebar, system colors and typography, and a Preferences pane with a user-selectable accent color.

**Architecture:** The Go layer (`main.go`) opts into a hidden-inset titlebar and a translucent window so macOS paints real vibrancy behind the web content. CSS then keeps the sidebar transparent (showing that vibrancy) while painting the main content pane opaque. All visual styling flows from a single CSS custom-property token block in `styles.css`; every component reads tokens rather than hardcoded colors, so the accent picker works by flipping one `data-accent` attribute on `<html>`.

**Tech Stack:** Go 1.x + Wails v2.13.0, React 18, Vite 5, plain CSS (no framework, no preprocessor).

## Global Constraints

- **No test runner exists.** `frontend/package.json` defines only `dev`, `build`, `preview`; there are no test dependencies and no test files in the repo. This plan is a visual conversion with no logic to unit-test. Verification is therefore: (a) `npm run build` must succeed (catches JSX/syntax errors), and (b) `wails dev` must launch and the described visual result must be observed. Do not add a test framework — that is out of scope for this plan.
- **Wails version is v2.13.0.** The v2 `mac.Options` struct has NO `Backdrop` field (that is Wails v3). Translucency is achieved with `WebviewIsTransparent: true` + `WindowIsTranslucent: true` + `BackgroundColour: &options.RGBA{R:0,G:0,B:0,A:0}`. Verified against `~/go/pkg/mod/github.com/wailsapp/wails/v2@v2.13.0/pkg/options/mac/mac.go`.
- **The `wails` binary lives at `~/go/bin/wails`** and `~/go/bin` is already on PATH via `~/.zshrc`.
- **Wails drag regions** use the CSS property `--wails-draggable: drag` (and `no-drag` to opt back out). These are the framework defaults for `CSSDragProperty`/`CSSDragValue`; do not override them in Go.
- **`color-mix(in srgb, ...)` is permitted.** It ships in Safari 16.2+ / macOS 13+, which is the WebKit engine Wails uses on macOS. It is used to derive hover and tint variants from `--accent` so each of the 8 accents needs only one hex per theme.
- **Class names are preserved.** Except where a task explicitly says otherwise, keep existing CSS class names so component JSX does not churn.
- **Radius scale (use these tokens, never raw px):** `--r-control: 6px` (buttons, inputs, segmented controls), `--r-card: 10px` (cards, panels, tiles), `--r-modal: 12px` (modals), `--r-chip: 5px` (small tag chips). Full pills (`999px`) survive ONLY on status badges.
- **Accent hexes (light / dark):** blue `#007AFF` / `#0A84FF`, purple `#AF52DE` / `#BF5AF2`, pink `#FF2D55` / `#FF375F`, red `#FF3B30` / `#FF453A`, orange `#FF9500` / `#FF9F0A`, yellow `#FFCC00` / `#FFD60A`, green `#28CD41` / `#32D74B`, graphite `#8E8E93` / `#98989D`.
- **Yellow accent uses black foreground text** (`--accent-fg: #000`); all other accents use white. Never hardcode `#fff` on an accent-filled surface — always use `var(--accent-fg)`.

---

### Task 1: Design token rewrite

Replace the blue-lavender "Mind Bridge" palette with macOS system colors and introduce the accent/radius token system. The app keeps its current layout and shapes after this task — only colors change. This lands first so every later task has tokens to consume.

**Files:**
- Modify: `frontend/src/styles.css:1-65` (the two `:root`/`[data-theme]` blocks and `body`)
- Modify: `frontend/src/styles.css` (global find/replace of `--navy` usages)

**Interfaces:**
- Consumes: nothing (first task).
- Produces: CSS custom properties consumed by every later task —
  `--bg`, `--card`, `--card-solid`, `--card-soft`, `--sidebar-hover`, `--border`, `--line`, `--text`, `--dim`, `--shadow`, `--shadow-sm`, `--overlay`, `--green`, `--green-soft`, `--red`, `--red-soft`, `--yellow`, `--yellow-soft`, `--log-bg`, `--log-text`, `--accent`, `--accent-hover`, `--accent-soft`, `--accent-fg`, `--r-control`, `--r-card`, `--r-modal`, `--r-chip`.
- Produces: the `[data-accent="<name>"]` attribute contract on `<html>` — Task 5's `useAccent` hook sets it; the 8 valid values are `blue purple pink red orange yellow green graphite`.
- Removes: `--bg-grad`, `--navy`, `--navy-hover`, `--accent-grad`. No later task may reference these.

- [ ] **Step 1: Replace the token blocks and `body` rule**

Replace `frontend/src/styles.css` lines 1 through 65 (everything from the opening `/* Mind Bridge design language: ... */` comment through the closing `}` of the `body` rule) with exactly this:

```css
/* macOS-native design language:
   system background + label colors, hairline separators, native vibrancy
   in the sidebar (painted by AppKit, see main.go), user-selectable accent. */

:root, [data-theme="light"] {
  --bg: #f6f6f7;
  --card: #ffffff;
  --card-solid: #ffffff;
  --card-soft: #f0f0f2;
  --sidebar-hover: rgba(0, 0, 0, 0.06);
  --border: #d1d1d6;
  --line: #d1d1d6;
  --text: #1d1d1f;
  --dim: #6e6e73;
  --shadow: 0 4px 12px rgba(0, 0, 0, 0.10);
  --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.12);
  --overlay: rgba(0, 0, 0, 0.28);
  --green: #28cd41;
  --green-soft: rgba(40, 205, 65, 0.14);
  --red: #ff3b30;
  --red-soft: rgba(255, 59, 48, 0.12);
  --yellow: #b8860b;
  --yellow-soft: rgba(255, 204, 0, 0.18);
  --log-bg: #1e1e1e;
  --log-text: #d4d4d4;
}

[data-theme="dark"] {
  --bg: #1e1e1e;
  --card: #2c2c2e;
  --card-solid: #2c2c2e;
  --card-soft: #3a3a3c;
  --sidebar-hover: rgba(255, 255, 255, 0.08);
  --border: #38383a;
  --line: #38383a;
  --text: #f5f5f5;
  --dim: #98989d;
  --shadow: 0 4px 12px rgba(0, 0, 0, 0.45);
  --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.40);
  --overlay: rgba(0, 0, 0, 0.45);
  --green: #32d74b;
  --green-soft: rgba(50, 215, 75, 0.18);
  --red: #ff453a;
  --red-soft: rgba(255, 69, 58, 0.18);
  --yellow: #ffd60a;
  --yellow-soft: rgba(255, 214, 10, 0.18);
  --log-bg: #141414;
  --log-text: #d4d4d4;
}

/* Radius scale — shared across themes */
:root {
  --r-control: 6px;
  --r-card: 10px;
  --r-modal: 12px;
  --r-chip: 5px;
}

/* Accent colors. `data-accent` is set on <html> by the useAccent hook in App.jsx.
   Defaults to blue when the attribute is absent. */
:root,
[data-accent="blue"] { --accent: #007aff; }
[data-accent="purple"] { --accent: #af52de; }
[data-accent="pink"] { --accent: #ff2d55; }
[data-accent="red"] { --accent: #ff3b30; }
[data-accent="orange"] { --accent: #ff9500; }
[data-accent="yellow"] { --accent: #ffcc00; }
[data-accent="green"] { --accent: #28cd41; }
[data-accent="graphite"] { --accent: #8e8e93; }

[data-theme="dark"],
[data-theme="dark"][data-accent="blue"] { --accent: #0a84ff; }
[data-theme="dark"][data-accent="purple"] { --accent: #bf5af2; }
[data-theme="dark"][data-accent="pink"] { --accent: #ff375f; }
[data-theme="dark"][data-accent="red"] { --accent: #ff453a; }
[data-theme="dark"][data-accent="orange"] { --accent: #ff9f0a; }
[data-theme="dark"][data-accent="yellow"] { --accent: #ffd60a; }
[data-theme="dark"][data-accent="green"] { --accent: #32d74b; }
[data-theme="dark"][data-accent="graphite"] { --accent: #98989d; }

/* Derived accent variants — one source of truth per accent. */
:root {
  --accent-fg: #ffffff;
  --accent-soft: color-mix(in srgb, var(--accent) 14%, transparent);
  --accent-hover: color-mix(in srgb, var(--accent) 86%, black);
}
[data-theme="dark"] {
  --accent-hover: color-mix(in srgb, var(--accent) 86%, white);
}
[data-accent="yellow"] { --accent-fg: #000000; }

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", sans-serif;
  background: var(--bg);
  color: var(--text);
  font-size: 13px;
  -webkit-font-smoothing: antialiased;
}
```

Note: `--yellow` in light mode is deliberately `#b8860b` (a dark gold), not `#ffcc00`. Pure macOS yellow on a white chip fails contrast for the `.kb-pill.prio-medium` and `.dep-status` text that use `--yellow` as a *foreground* color. `--yellow-soft` keeps the bright hue for backgrounds.

- [ ] **Step 2: Repoint every `--navy` and `--accent-grad` usage**

`--navy`, `--navy-hover`, `--accent-grad`, and `--bg-grad` no longer exist. Find every remaining usage and repoint it. Run this to list them:

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -n -- "--navy\|--accent-grad\|--bg-grad" frontend/src/styles.css
```

Expected: 13 matches across `.rail-btn.active`, `.side-item.active`, `.sidebar .add-btn`, `.sidebar .add-btn:hover`, `.theme-toggle button.active`, `.btn.primary`, `.btn.primary:hover`, `.tabs button.active`, `.avatar`, `.full-btn`, `.full-btn:hover`, `.meter > div`.

Apply these replacements:

- `background: var(--navy);` → `background: var(--accent);`
- `background: var(--navy-hover);` → `background: var(--accent-hover);`
- `border-color: var(--navy);` → `border-color: var(--accent);`
- Any `color: #fff;` on the *same rule* as an accent background → `color: var(--accent-fg);`
- `.avatar`'s `background: var(--accent-grad);` → `background: var(--accent);` and its `color: #fff;` → `color: var(--accent-fg);`
- `.meter > div`'s `background: linear-gradient(90deg, #8ec2f5, var(--accent));` → `background: var(--accent);`

Also fix the two hardcoded selection colors so they follow the accent:
- `.side-item.active { background: var(--navy); color: #fff; }` → `background: var(--accent); color: var(--accent-fg);`
- `.rail-btn.active { background: var(--navy); border-color: var(--navy); color: #fff; }` → `background: var(--accent); border-color: var(--accent); color: var(--accent-fg);`

And the modal overlay, which hardcodes a blue-tinted scrim:
- `.modal-overlay { background: rgba(18, 26, 54, 0.4); ... }` → `background: var(--overlay);`

- [ ] **Step 3: Verify no dead tokens remain**

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -n -- "--navy\|--accent-grad\|--bg-grad" frontend/src/styles.css
```

Expected: no output (exit code 1).

- [ ] **Step 4: Verify the build succeeds**

```bash
cd /Users/yasharyan/Projects/jumpstart/frontend && npm run build
```

Expected: `✓ built in ...` with no errors.

- [ ] **Step 5: Verify visually**

```bash
cd /Users/yasharyan/Projects/jumpstart && wails dev
```

Expected: the app launches. The gradient background is gone — flat light gray `#f6f6f7`. Selected sidebar rows and primary buttons are macOS blue. Toggle to Dark in the theme control: background becomes `#1e1e1e`, accent becomes the brighter `#0A84FF`. Shapes are still the old pills/big radii — that is correct at this stage. Quit with `Ctrl+C`.

- [ ] **Step 6: Commit**

```bash
cd /Users/yasharyan/Projects/jumpstart
git add frontend/src/styles.css
git commit -m "style: replace Mind Bridge palette with macOS system color tokens"
```

---

### Task 2: Native window chrome and app shell

Turn on the hidden-inset titlebar and window translucency in Go, then restructure the CSS shell so the sidebar is transparent (showing AppKit's vibrancy) while the main pane is opaque. Adds the drag regions that replace the now-hidden titlebar.

**Files:**
- Modify: `/Users/yasharyan/Projects/jumpstart/main.go` (whole file)
- Modify: `frontend/src/styles.css` — the `body` rule (from Task 1) and the `.layout` / `.sidebar` / `.main` rules

**Interfaces:**
- Consumes: `--bg` from Task 1.
- Produces: `.sidebar` is transparent and 240px wide; `.main` is the opaque content pane. Task 3 rebuilds the sidebar's *contents* against this shell.
- Produces: `.titlebar-drag` (a 38px draggable strip at the top of the sidebar) and a draggable `.topbar`. Any interactive element placed inside either MUST set `--wails-draggable: no-drag` or it will not receive clicks.

- [ ] **Step 1: Rewrite `main.go`**

Replace the entire contents of `/Users/yasharyan/Projects/jumpstart/main.go` with:

```go
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "DevDeck",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		// Fully transparent so AppKit's vibrancy shows through wherever the
		// web content does not paint an opaque background (i.e. the sidebar).
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Mac: &mac.Options{
			// Hides the title bar and title text, keeps the traffic lights,
			// inset from the window edge (the Mail / Notes / Xcode look).
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "DevDeck",
				Message: "Local dev process manager",
			},
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
```

- [ ] **Step 2: Verify Go compiles**

```bash
cd /Users/yasharyan/Projects/jumpstart && go build -o /dev/null .
```

Expected: no output, exit 0. If it fails with `undefined: mac.TitleBarHiddenInset`, the import path is wrong — it is `github.com/wailsapp/wails/v2/pkg/options/mac`.

- [ ] **Step 3: Make `body` transparent and restructure the shell**

In `frontend/src/styles.css`, change the `body` rule's background from `var(--bg)` to `transparent`, and add the html/root height rules. The `body` rule becomes:

```css
html, body, #root { height: 100%; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", sans-serif;
  /* Transparent so AppKit vibrancy shows through the sidebar. The main pane
     paints itself opaque; see `.main`. */
  background: transparent;
  color: var(--text);
  font-size: 13px;
  -webkit-font-smoothing: antialiased;
}
```

- [ ] **Step 4: Replace the `.layout` rule and the shell parts of `.sidebar` / `.main`**

Replace the existing `.layout` rule (currently `.layout { display: flex; height: 100vh; gap: 20px; padding: 20px; }`) with:

```css
.layout { display: flex; height: 100vh; overflow: hidden; }
```

Replace the existing `.sidebar` rule's *shell* properties. The full `.sidebar` rule becomes:

```css
/* Transparent: AppKit paints native vibrancy behind it (see main.go). */
.sidebar {
  width: 240px;
  flex-shrink: 0;
  background: transparent;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 0 10px 10px;
  overflow: hidden;
  transition: width 0.2s ease, padding 0.2s ease, opacity 0.2s ease;
}

/* Hidden via the toolbar toggle (Task 3). */
.layout.sidebar-hidden .sidebar {
  width: 0;
  padding-left: 0;
  padding-right: 0;
  border-right: none;
  opacity: 0;
}

/* Replaces the titlebar we hid in Go: gives the traffic lights room and
   keeps the window draggable by the empty space beside them. */
.titlebar-drag {
  height: 38px;
  flex-shrink: 0;
  --wails-draggable: drag;
}
```

Delete the now-dead `.sidebar.collapsed` rules — the icon-only collapsed mode is gone. Run this to find them:

```bash
grep -n "sidebar.collapsed" frontend/src/styles.css
```

Expected: 6 matches (`.sidebar.collapsed`, `.sidebar.collapsed .sidebar-head`, `.sidebar.collapsed .side-item`, `.sidebar.collapsed .side-text`, `.sidebar.collapsed .collapse-btn svg`, `.sidebar.collapsed .add-btn`). Delete all 6 rules.

Replace the `.main` rule with:

```css
/* Opaque: this is the one surface that must NOT show vibrancy. */
.main {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  background: var(--bg);
  padding: 0 22px 24px;
}
```

Replace the `.topbar` rule with:

```css
.topbar {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  margin-bottom: 16px;
  position: sticky;
  top: 0;
  background: var(--bg);
  z-index: 10;
  --wails-draggable: drag;
}
/* Buttons inside a drag region need to opt back out to stay clickable. */
.topbar button,
.topbar input,
.topbar select { --wails-draggable: no-drag; }
.topbar h1 { font-size: 22px; font-weight: 700; letter-spacing: -0.4px; }
.topbar .right { display: flex; align-items: center; gap: 8px; margin-left: auto; }

/* When the sidebar is hidden the traffic lights sit over the main pane,
   so the toolbar content has to clear them. */
.layout.sidebar-hidden .topbar { padding-left: 72px; }
```

- [ ] **Step 5: Add the `sidebar-hidden` class hook to `App.jsx`**

This is a placeholder wiring step so the CSS above has something to react to; Task 3 adds the actual toggle button. In `frontend/src/App.jsx`, change the root element from `<div className="layout">` to:

```jsx
<div className={`layout ${sidebarOpen ? "" : "sidebar-hidden"}`}>
```

and add this state alongside the other `useState` calls in `App()`:

```jsx
  const [sidebarOpen, setSidebarOpen] = useState(
    () => localStorage.getItem("sidebarOpen") !== "0"
  );

  useEffect(() => {
    localStorage.setItem("sidebarOpen", sidebarOpen ? "1" : "0");
  }, [sidebarOpen]);
```

`setSidebarOpen` is intentionally unused until Task 3 adds the toggle button.

- [ ] **Step 6: Add the drag strip to the sidebar**

In `frontend/src/components/Sidebar.jsx`, add the drag strip as the first child of `<aside>`, immediately before `<div className="sidebar-head">`:

```jsx
      <div className="titlebar-drag" />
```

- [ ] **Step 7: Verify the build succeeds**

```bash
cd /Users/yasharyan/Projects/jumpstart/frontend && npm run build
```

Expected: `✓ built in ...` with no errors.

- [ ] **Step 8: Verify visually**

```bash
cd /Users/yasharyan/Projects/jumpstart && wails dev
```

Expected, all of which must hold before moving on:
1. No title bar or title text. Traffic lights float, inset, over the top-left of the sidebar.
2. The sidebar region is translucent — moving a colorful window or the desktop wallpaper behind DevDeck visibly tints/blurs through it.
3. The main content pane on the right is fully opaque (`#f6f6f7` light / `#1e1e1e` dark), NOT see-through.
4. Dragging the empty space above the sidebar's "Projects" heading moves the window.
5. Dragging the toolbar area beside the page title moves the window, but clicking the theme toggle in it still works.

If (3) fails and the whole window is see-through, `.main`'s `background: var(--bg)` did not apply — check that the `.main` rule was replaced, not appended after a later conflicting rule.

Quit with `Ctrl+C`.

- [ ] **Step 9: Commit**

```bash
cd /Users/yasharyan/Projects/jumpstart
git add main.go frontend/src/styles.css frontend/src/App.jsx frontend/src/components/Sidebar.jsx
git commit -m "feat: native macOS titlebar and vibrancy sidebar"
```

---

### Task 3: Unified sidebar

Merge the circular icon rail into the projects sidebar to form one macOS-style sidebar (Mail/Notes pattern): nav rows on top, a `Projects` section header, project rows, then a footer with "Add Project" and a Preferences gear. Delete `IconRail.jsx`. Move the collapse control into the main toolbar.

**Files:**
- Create: `frontend/src/components/Icon.jsx`
- Rewrite: `frontend/src/components/Sidebar.jsx`
- Delete: `frontend/src/components/IconRail.jsx`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/styles.css` (replace the `.rail*`, `.sidebar-head`, `.collapse-btn`, `.side-item`, `.sidebar .add-btn` rules)

**Interfaces:**
- Consumes: `.sidebar` / `.titlebar-drag` / `.layout.sidebar-hidden` shell and the `sidebarOpen` state from Task 2; all tokens from Task 1.
- Produces: `frontend/src/components/Icon.jsx` exporting a default `Icon` component with signature `({ d }: { d: string }) => JSX.Element` (where `d` is one or more SVG path `d` strings joined by `|`), and a named export `ICONS` — an object with the string keys `dashboard`, `ports`, `gear`, `plus`, `sidebar`. Tasks 4 and 5 import from here.
- Produces: `Sidebar` props change from `{ projects, selectedId, onSelect, onAdd }` to `{ projects, view, selectedId, onNavigate, onSelect, onAdd, onOpenPrefs }`. `onNavigate(view: "dashboard" | "ports")`. `onOpenPrefs()` takes no arguments and is consumed by Task 5.

- [ ] **Step 1: Create the shared icon component**

Create `frontend/src/components/Icon.jsx`:

```jsx
export const ICONS = {
  dashboard: "M3 3h8v8H3z|M13 3h8v5h-8z|M13 10h8v11h-8z|M3 13h8v8H3z",
  ports: "M4 17l6-6-6-6|M12 19h8",
  plus: "M12 5v14|M5 12h14",
  sidebar: "M3 4h18v16H3z|M9 4v16",
  gear: "M12 15a3 3 0 100-6 3 3 0 000 6z|M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 11-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 11-4 0v-.09A1.65 1.65 0 008 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 11-2.83-2.83l.06-.06A1.65 1.65 0 004.6 15a1.65 1.65 0 00-1.51-1H3a2 2 0 110-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 112.83-2.83l.06.06A1.65 1.65 0 009 4.6a1.65 1.65 0 001-1.51V3a2 2 0 114 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 112.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 110 4h-.09a1.65 1.65 0 00-1.51 1z",
};

export default function Icon({ d }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {d.split("|").map((p, i) => (
        <path key={i} d={p} />
      ))}
    </svg>
  );
}
```

- [ ] **Step 2: Rewrite `Sidebar.jsx`**

Replace the entire contents of `frontend/src/components/Sidebar.jsx` with:

```jsx
import Icon, { ICONS } from "./Icon";

const initial = (name) => (name || "?").trim().charAt(0).toUpperCase();

const plural = (n) => `${n} subprocess${n === 1 ? "" : "es"}`;

export default function Sidebar({
  projects,
  view,
  selectedId,
  onNavigate,
  onSelect,
  onAdd,
  onOpenPrefs,
}) {
  return (
    <aside className="sidebar">
      <div className="titlebar-drag" />

      <nav className="side-group">
        <button
          className={`side-row ${view === "dashboard" ? "active" : ""}`}
          onClick={() => onNavigate("dashboard")}
        >
          <Icon d={ICONS.dashboard} />
          <span>Dashboard</span>
        </button>
        <button
          className={`side-row ${view === "ports" ? "active" : ""}`}
          onClick={() => onNavigate("ports")}
        >
          <Icon d={ICONS.ports} />
          <span>Ports</span>
        </button>
      </nav>

      <div className="side-section">Projects</div>

      <nav className="side-group side-projects">
        {projects.map((p) => (
          <button
            key={p.id}
            className={`side-row project ${p.id === selectedId ? "active" : ""}`}
            onClick={() => onSelect(p.id)}
          >
            <span className="avatar">{initial(p.name)}</span>
            <span className="side-text">
              <span className="side-name">{p.name}</span>
              <span className="side-sub">{plural((p.processes || []).length)}</span>
            </span>
          </button>
        ))}
        {projects.length === 0 && <div className="side-empty">No projects yet</div>}
      </nav>

      <div className="sidebar-foot">
        <button className="side-row" onClick={onAdd}>
          <Icon d={ICONS.plus} />
          <span>Add Project</span>
        </button>
        <button className="icon-btn" title="Preferences" onClick={onOpenPrefs}>
          <Icon d={ICONS.gear} />
        </button>
      </div>
    </aside>
  );
}
```

- [ ] **Step 3: Delete `IconRail.jsx`**

```bash
cd /Users/yasharyan/Projects/jumpstart
git rm frontend/src/components/IconRail.jsx
```

- [ ] **Step 4: Rewrite the sidebar CSS**

In `frontend/src/styles.css`, delete every rule in the `/* Icon rail ... */` block (`.rail`, `.rail .logo`, `.rail-btn`, `.rail-btn svg`, `.rail-btn:hover`, `.rail-btn.active`, `.rail .spacer`) and every remaining `.sidebar-head`, `.collapse-btn`, `.sidebar .sub`, `.sidebar nav`, `.side-item*`, and `.sidebar .add-btn` rule. Replace them all with:

```css
/* Sidebar contents — Finder / Mail style rows */
.side-group { display: flex; flex-direction: column; gap: 1px; }
.side-projects { flex: 1; overflow-y: auto; overflow-x: hidden; min-height: 0; }

.side-section {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--dim);
  padding: 14px 8px 5px;
}

.side-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  text-align: left;
  padding: 5px 8px;
  background: none;
  border: none;
  border-radius: var(--r-control);
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
}
.side-row svg { width: 17px; height: 17px; flex-shrink: 0; color: var(--dim); }
.side-row:hover { background: var(--sidebar-hover); }
.side-row.active { background: var(--accent); color: var(--accent-fg); }
.side-row.active svg { color: var(--accent-fg); }

.side-row.project { padding: 5px 8px; }
.side-text { display: flex; flex-direction: column; min-width: 0; }
.side-name { overflow: hidden; text-overflow: ellipsis; }
.side-sub { font-size: 11px; color: var(--dim); font-weight: 400; }
.side-row.active .side-sub { color: var(--accent-fg); opacity: 0.75; }
.side-row.active .avatar { background: rgba(255, 255, 255, 0.22); color: var(--accent-fg); }

.side-empty { padding: 6px 8px; font-size: 12px; color: var(--dim); }

.sidebar-foot {
  display: flex;
  align-items: center;
  gap: 4px;
  padding-top: 6px;
  border-top: 1px solid var(--border);
  margin-top: 6px;
}
.sidebar-foot .side-row { flex: 1; }

/* Square icon button — toolbar toggle, preferences gear */
.icon-btn {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border: none;
  background: none;
  border-radius: var(--r-control);
  color: var(--dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.icon-btn svg { width: 17px; height: 17px; }
.icon-btn:hover { background: var(--sidebar-hover); color: var(--text); }
```

- [ ] **Step 5: Shrink `.avatar` to sidebar scale**

The existing `.avatar` rule is sized 34px for the old design. Replace it with:

```css
.avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--accent);
  color: var(--accent-fg);
  font-weight: 600;
  font-size: 11px;
  display: inline-grid;
  place-items: center;
  flex-shrink: 0;
}
```

Note: `Dashboard.jsx`'s `.quick-row` also uses `.avatar`. 22px reads correctly there too — do not add a size override.

- [ ] **Step 6: Wire `App.jsx`**

In `frontend/src/App.jsx`:

Replace the `IconRail` import with the shared icon:

```jsx
import Icon, { ICONS } from "./components/Icon";
```

Delete the `<IconRail ... />` element entirely and replace the `<Sidebar ... />` element with:

```jsx
      <Sidebar
        projects={projects}
        view={view}
        selectedId={view === "project" ? selectedId : null}
        onNavigate={(v) => {
          setView(v);
          setSelectedId(null);
        }}
        onSelect={openProject}
        onAdd={() => setModal("new")}
        onOpenPrefs={() => setPrefsOpen(true)}
      />
```

Add the `prefsOpen` state alongside the other `useState` calls (the Preferences modal itself lands in Task 5; for now the state exists and nothing renders it):

```jsx
  const [prefsOpen, setPrefsOpen] = useState(false);
```

Replace the `<div className="topbar">` block with one that leads with the sidebar toggle:

```jsx
        <div className="topbar">
          <button
            className="icon-btn"
            title="Toggle Sidebar"
            onClick={() => setSidebarOpen((o) => !o)}
          >
            <Icon d={ICONS.sidebar} />
          </button>
          <h1>{view === "project" && selected ? selected.name : titles[view] || "Dashboard"}</h1>
          <div className="right">
            <ThemeToggle theme={theme} onChange={setTheme} />
          </div>
        </div>
```

(`ThemeToggle` stays in the toolbar for now; Task 5 moves it into Preferences.)

- [ ] **Step 7: Verify no dangling references to the deleted component**

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -rn "IconRail\|side-item\|rail-btn\|sidebarCollapsed" frontend/src/
```

Expected: no output (exit code 1). If `sidebarCollapsed` still appears, an old `localStorage` key survived in `Sidebar.jsx` — it should have been removed by the Step 2 rewrite.

- [ ] **Step 8: Verify the build succeeds**

```bash
cd /Users/yasharyan/Projects/jumpstart/frontend && npm run build
```

Expected: `✓ built in ...` with no errors.

- [ ] **Step 9: Verify visually**

```bash
cd /Users/yasharyan/Projects/jumpstart && wails dev
```

Expected:
1. One sidebar, no circular icon rail. Dashboard and Ports are labeled rows at the top.
2. A small uppercase gray `PROJECTS` header sits above the project rows.
3. Clicking a row fills it solid accent-blue with white text, ~6px corners — not a pill.
4. The toolbar's leftmost button hides and shows the sidebar. With it hidden, the page title clears the traffic lights rather than sitting under them.
5. The footer shows an "Add Project" row and a gear button. The gear does nothing yet (Task 5).

Quit with `Ctrl+C`.

- [ ] **Step 10: Commit**

```bash
cd /Users/yasharyan/Projects/jumpstart
git add -A frontend/src main.go
git commit -m "feat: merge icon rail into unified macOS sidebar"
```

---

### Task 4: Component shape pass

Bring every remaining component in line with HIG shapes: control radii down from pills, cards opaque with hairline borders instead of `backdrop-filter`, macOS type scale, subtle shadows, a real segmented control.

**Files:**
- Modify: `frontend/src/styles.css` (rules from `/* Buttons */` onward)

**Interfaces:**
- Consumes: all tokens from Task 1; `.icon-btn` from Task 3.
- Produces: `.segmented` styling reused by Task 5's Preferences pane (`ThemeToggle` keeps its existing `.theme-toggle` class name, restyled here).

- [ ] **Step 1: Buttons**

Replace the `/* Buttons */` block (`.btn`, `.btn:hover`, `.btn:disabled`, `.btn.primary`, `.btn.primary:hover`, `.btn.danger`, `.btn.danger:hover`, `.btn.small`) with:

```css
/* Buttons — macOS push-button proportions */
.btn {
  padding: 5px 14px;
  border-radius: var(--r-control);
  border: 1px solid var(--border);
  background: var(--card-solid);
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
  font-family: inherit;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: background 0.12s;
}
.btn:hover { background: var(--card-soft); }
.btn:disabled { opacity: 0.45; cursor: default; box-shadow: none; }
.btn:disabled:hover { background: var(--card-solid); }
.btn.primary {
  background: var(--accent);
  border-color: transparent;
  color: var(--accent-fg);
  font-weight: 600;
}
.btn.primary:hover { background: var(--accent-hover); }
.btn.danger { color: var(--red); }
.btn.danger:hover { background: var(--red-soft); }
.btn.small { padding: 3px 10px; font-size: 12px; }
```

- [ ] **Step 2: Segmented controls (tabs + theme toggle)**

Replace both the `/* Theme toggle */` block (`.theme-toggle`, `.theme-toggle button`, `.theme-toggle button.active`) and the `/* Tabs */` block (`.tabs`, `.tabs button`, `.tabs button.active`) with a single shared implementation:

```css
/* Segmented control — macOS NSSegmentedControl proportions */
.theme-toggle,
.tabs {
  display: inline-flex;
  background: var(--card-soft);
  border: 1px solid var(--border);
  border-radius: var(--r-control);
  padding: 2px;
  gap: 2px;
}
.tabs { margin-bottom: 16px; }
.theme-toggle button,
.tabs button {
  border: none;
  background: none;
  color: var(--text);
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
}
.theme-toggle button:hover,
.tabs button:hover { background: var(--sidebar-hover); }
.theme-toggle button.active,
.tabs button.active {
  background: var(--card-solid);
  color: var(--text);
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}
.theme-toggle button.active:hover,
.tabs button.active:hover { background: var(--card-solid); }
```

Note the selected segment is a raised white chip on a gray track — that is the real macOS segmented control, not an accent fill.

- [ ] **Step 3: Cards, tiles, and panels**

These three rules all currently use `backdrop-filter: blur()` plus a big radius. Vibrancy is the sidebar's job now, so they become opaque. Replace `.card`, `.tile`, and `.panel` with:

```css
.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 14px; }
.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--r-card);
  padding: 14px;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.12s;
}
.card:hover { box-shadow: var(--shadow); }
.card.running { border-color: var(--green); }

.tiles { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px; margin-bottom: 18px; }
.tile {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--r-card);
  padding: 14px;
  box-shadow: var(--shadow-sm);
}

.panel {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: var(--r-card);
  padding: 16px;
  box-shadow: var(--shadow-sm);
}
```

Note `.card:hover` loses its `transform: translateY(-2px)` lift — Mac apps do not levitate list content on hover.

- [ ] **Step 4: Remove every remaining `backdrop-filter`**

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -n "backdrop-filter" frontend/src/styles.css
```

Expected after Step 3: matches on `.kb-col` and `.modal-overlay`. Delete the `backdrop-filter` declaration from `.kb-col`. **Keep** the one on `.modal-overlay` — a blurred scrim behind a sheet is correct macOS behavior.

- [ ] **Step 5: Badges, pills, and chips**

Status pills stay capsule-shaped (macOS uses capsules for status). Tag-like chips become small rounded rects. Replace `.status-pill`, `.port-badge`, `.usage-badge`, `.delta-pill`, `.kb-pill`, `.kb-count`, and `.dep-kind` with:

```css
/* Status = capsule (macOS uses capsules for status indicators) */
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 9px;
  border-radius: 999px;
}
.status-pill.on { background: var(--green-soft); color: var(--green); }
.status-pill.off { background: var(--card-soft); color: var(--dim); }
.dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

/* Tags = small rounded rects */
.port-badge {
  background: var(--accent-soft);
  color: var(--accent);
  border-radius: var(--r-chip);
  padding: 2px 7px;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  font-family: inherit;
}
.port-badge:hover { background: var(--accent); color: var(--accent-fg); }

.usage-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: var(--r-chip);
  background: var(--card-soft);
  color: var(--dim);
}
.usage-badge.hot { background: var(--red-soft); color: var(--red); }

.delta-pill {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: var(--r-chip);
  background: var(--green-soft);
  color: var(--green);
}
.delta-pill.neutral { background: var(--card-soft); color: var(--dim); }

.kb-count {
  font-size: 11px;
  font-weight: 600;
  background: var(--card-soft);
  color: var(--dim);
  border-radius: var(--r-chip);
  padding: 1px 7px;
}

.kb-pill {
  font-size: 10.5px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: var(--r-chip);
  background: var(--card-soft);
  color: var(--dim);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.kb-pill.prio-high { background: var(--red-soft); color: var(--red); }
.kb-pill.prio-medium { background: var(--yellow-soft); color: var(--yellow); }
.kb-pill.prio-low { background: var(--green-soft); color: var(--green); }
.kb-pill.label { background: var(--accent-soft); color: var(--accent); }
.kb-pill.label button {
  border: none;
  background: none;
  color: inherit;
  cursor: pointer;
  font-size: 12px;
  padding: 0;
  line-height: 1;
}

.dep-kind {
  font-size: 10px;
  color: var(--dim);
  border: 1px solid var(--border);
  border-radius: var(--r-chip);
  padding: 0 5px;
}
```

- [ ] **Step 6: Form controls**

Replace `.field input`, `.field input:focus`, `.env-row input`, `.task-add input`, `.task-add input:focus`, `.kb-detail select`, `.kb-detail select:focus`, `.kb-detail textarea`, `.kb-detail textarea:focus`, and `.kb-add-input` with:

```css
/* Text fields — macOS bezel + focus ring */
.field input,
.env-row input,
.task-add input,
.kb-detail select,
.kb-detail textarea,
.kb-add-input {
  width: 100%;
  padding: 5px 8px;
  border: 1px solid var(--border);
  border-radius: var(--r-control);
  font-size: 13px;
  font-family: inherit;
  background: var(--card-solid);
  color: var(--text);
}
.field input:focus,
.env-row input:focus,
.task-add input:focus,
.kb-detail select:focus,
.kb-detail textarea:focus,
.kb-add-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.env-row input { font-size: 12px; }
.task-add input { flex: 1; }
.kb-detail textarea { resize: vertical; }
.kb-add-input { margin-top: 8px; }
```

Delete the now-duplicated `.env-row input` / `.task-add input` declarations left behind elsewhere in the file, keeping only the shared rule above plus the three one-line overrides.

- [ ] **Step 7: Checkbox, modal, logs, and the remaining radii**

Replace `.task-check` (currently a 6px rounded square) with a macOS checkbox:

```css
.task-check {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: var(--card-solid);
  cursor: pointer;
  display: grid;
  place-items: center;
  color: var(--accent-fg);
  font-size: 10px;
  flex-shrink: 0;
}
.task-check:hover { border-color: var(--accent); }
.task-row.done .task-check { background: var(--accent); border-color: var(--accent); }
```

Replace `.modal`:

```css
.modal {
  background: var(--card-solid);
  border: 1px solid var(--border);
  border-radius: var(--r-modal);
  width: 660px;
  max-height: 85vh;
  overflow-y: auto;
  padding: 22px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
}
.modal h2 { font-size: 16px; font-weight: 600; margin-bottom: 14px; }
```

Then sweep the remaining oversized radii. Run:

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -n "border-radius: 1[2-9]px\|border-radius: 2[0-9]px\|border-radius: 999px" frontend/src/styles.css
```

For each match, apply this mapping (the `999px` matches on `.status-pill`, `.meter`, and `.meter > div` are correct — leave those three alone):
- `.logs`, `.kb-col` → `var(--r-card)`
- `.card .cmd`, `.conf-path`, `.proc-block`, `.env-import`, `.kb-card`, `.kb-empty`, `.kb-add-btn`, `.quick-row`, `.task-row`, `.toast` → `var(--r-control)`
- `.full-btn`, `.sidebar .add-btn` (if any survives) → `var(--r-control)`

`.full-btn` also loses its pill shape and gains accent-aware text:

```css
.full-btn {
  width: 100%;
  margin-top: 10px;
  padding: 6px;
  border: none;
  border-radius: var(--r-control);
  background: var(--accent);
  color: var(--accent-fg);
  font-weight: 600;
  font-size: 13px;
  font-family: inherit;
  cursor: pointer;
}
.full-btn:hover { background: var(--accent-hover); }
```

- [ ] **Step 8: Type scale**

Apply the macOS type scale to the headings that still carry the old sizes:

- `.main-header h1 { font-size: 22px; font-weight: 700; }` → keep 22px, but set `letter-spacing: -0.4px;`
- `.card-top h3 { font-size: 15px; font-weight: 700; }` → `font-size: 14px; font-weight: 600;`
- `.panel h3 { font-size: 14px; }` → `font-size: 14px; font-weight: 600;`
- `.tile .tile-value { font-size: 32px; ... }` → `font-size: 28px; font-weight: 600; letter-spacing: -0.5px;`
- `.tile .tile-label { font-size: 13.5px; font-weight: 600; }` → `font-size: 12px; font-weight: 500; color: var(--dim);`
- `.side-row`-adjacent leftovers: `.quick-row .q-name { font-size: 13px; font-weight: 600; }` → `font-weight: 500;`
- `.task-row .task-title { font-size: 13.5px; font-weight: 600; }` → `font-size: 13px; font-weight: 500;`
- `.empty h2 { font-size: 18px; }` → `font-size: 16px; font-weight: 600;`

- [ ] **Step 9: Verify the build succeeds**

```bash
cd /Users/yasharyan/Projects/jumpstart/frontend && npm run build
```

Expected: `✓ built in ...` with no errors.

- [ ] **Step 10: Verify visually**

```bash
cd /Users/yasharyan/Projects/jumpstart && wails dev
```

Walk every view — Dashboard, Ports, a project (its Processes / Tasks / Kanban tabs), the Add Project modal, and a Kanban task detail modal. Expected:
1. No pill-shaped buttons anywhere. Buttons are ~6px rounded rects.
2. Cards, tiles, and panels are opaque with a hairline border and a barely-there shadow — no frosted blur.
3. The Light/Dark/Auto control and the project tabs both render as a gray track with a raised white selected segment.
4. Focusing a text input draws a 3px accent-tinted ring.
5. Status pills (`Running` / `Stopped`) are still capsules; priority and label tags are now small rounded chips.
6. The modal scrim is still blurred.

Quit with `Ctrl+C`.

- [ ] **Step 11: Commit**

```bash
cd /Users/yasharyan/Projects/jumpstart
git add frontend/src/styles.css
git commit -m "style: HIG control shapes, type scale, and opaque cards"
```

---

### Task 5: Preferences pane with accent picker

Add a macOS-Settings-style Preferences modal holding the appearance control and an 8-swatch accent picker. Move `ThemeToggle` out of the main toolbar into it.

**Files:**
- Create: `frontend/src/components/Preferences.jsx`
- Modify: `frontend/src/App.jsx`
- Modify: `frontend/src/styles.css` (append the Preferences block)

**Interfaces:**
- Consumes: `prefsOpen` / `setPrefsOpen` state and the `onOpenPrefs` gear wired in Task 3; `.modal` / `.modal-overlay` / `.theme-toggle` from Task 4; the `[data-accent]` token contract from Task 1.
- Consumes: `ThemeToggle` from `./ThemeToggle` — props `{ theme: "light" | "dark" | "system", onChange: (mode) => void }`, unchanged from today.
- Produces: `Preferences` default export with props `{ theme, onThemeChange, accent, onAccentChange, onClose }` and a named export `ACCENTS: string[]` (the 8 accent names in display order).

- [ ] **Step 1: Create `Preferences.jsx`**

Create `frontend/src/components/Preferences.jsx`:

```jsx
import ThemeToggle from "./ThemeToggle";

export const ACCENTS = [
  "blue",
  "purple",
  "pink",
  "red",
  "orange",
  "yellow",
  "green",
  "graphite",
];

export default function Preferences({
  theme,
  onThemeChange,
  accent,
  onAccentChange,
  onClose,
}) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal prefs" onClick={(e) => e.stopPropagation()}>
        <h2>Preferences</h2>

        <div className="prefs-row">
          <label>Appearance</label>
          <ThemeToggle theme={theme} onChange={onThemeChange} />
        </div>

        <div className="prefs-row">
          <label>Accent color</label>
          <div className="swatches">
            {ACCENTS.map((a) => (
              <button
                key={a}
                className={`swatch ${accent === a ? "active" : ""}`}
                data-swatch={a}
                title={a}
                aria-label={a}
                aria-pressed={accent === a}
                onClick={() => onAccentChange(a)}
              />
            ))}
          </div>
        </div>

        <div className="modal-actions">
          <button className="btn primary" onClick={onClose}>
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Add the `useAccent` hook and wire the modal in `App.jsx`**

In `frontend/src/App.jsx`, add this hook directly below the existing `useTheme` hook:

```jsx
function useAccent() {
  const [accent, setAccent] = useState(
    () => localStorage.getItem("accent") || "blue"
  );
  useEffect(() => {
    document.documentElement.dataset.accent = accent;
    localStorage.setItem("accent", accent);
  }, [accent]);
  return [accent, setAccent];
}
```

Add the import:

```jsx
import Preferences from "./components/Preferences";
```

Remove the `ThemeToggle` import — it is now only used inside `Preferences.jsx`.

Inside `App()`, call the hook next to `useTheme`:

```jsx
  const [accent, setAccent] = useAccent();
```

Strip the theme toggle out of the toolbar. The `.topbar` block becomes:

```jsx
        <div className="topbar">
          <button
            className="icon-btn"
            title="Toggle Sidebar"
            onClick={() => setSidebarOpen((o) => !o)}
          >
            <Icon d={ICONS.sidebar} />
          </button>
          <h1>{view === "project" && selected ? selected.name : titles[view] || "Dashboard"}</h1>
        </div>
```

Finally, render the modal next to the existing `{modal && ...}` block, just before the toast:

```jsx
      {prefsOpen && (
        <Preferences
          theme={theme}
          onThemeChange={setTheme}
          accent={accent}
          onAccentChange={setAccent}
          onClose={() => setPrefsOpen(false)}
        />
      )}
```

- [ ] **Step 3: Style the Preferences pane**

Append to `frontend/src/styles.css`:

```css
/* Preferences — macOS Settings pane proportions */
.modal.prefs { width: 420px; }
.prefs-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border);
}
.prefs-row:last-of-type { border-bottom: none; }
.prefs-row label { font-size: 13px; font-weight: 500; color: var(--text); }

.swatches { display: flex; gap: 8px; }
.swatch {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid rgba(0, 0, 0, 0.12);
  cursor: pointer;
  padding: 0;
}
.swatch.active {
  box-shadow: 0 0 0 2px var(--card-solid), 0 0 0 4px var(--dim);
}
.swatch[data-swatch="blue"] { background: #007aff; }
.swatch[data-swatch="purple"] { background: #af52de; }
.swatch[data-swatch="pink"] { background: #ff2d55; }
.swatch[data-swatch="red"] { background: #ff3b30; }
.swatch[data-swatch="orange"] { background: #ff9500; }
.swatch[data-swatch="yellow"] { background: #ffcc00; }
.swatch[data-swatch="green"] { background: #28cd41; }
.swatch[data-swatch="graphite"] { background: #8e8e93; }
```

- [ ] **Step 4: Verify the toolbar no longer imports ThemeToggle**

```bash
cd /Users/yasharyan/Projects/jumpstart
grep -rn "ThemeToggle" frontend/src/
```

Expected: exactly two matches, both in `frontend/src/components/Preferences.jsx` (the import and the JSX usage), plus the component's own definition file `frontend/src/components/ThemeToggle.jsx`. No matches in `App.jsx`.

- [ ] **Step 5: Verify the build succeeds**

```bash
cd /Users/yasharyan/Projects/jumpstart/frontend && npm run build
```

Expected: `✓ built in ...` with no errors.

- [ ] **Step 6: Verify visually**

```bash
cd /Users/yasharyan/Projects/jumpstart && wails dev
```

Expected:
1. The main toolbar shows only the sidebar toggle and the page title — no theme control.
2. The sidebar's gear button opens a 420px Preferences sheet over a blurred scrim.
3. Clicking a swatch immediately recolors selected sidebar rows, primary buttons, focus rings, meters, and avatars across the whole app.
4. Picking **yellow** produces black text on the primary button and on selected sidebar rows (not unreadable white).
5. Picking **graphite** in dark mode yields the lighter `#98989d`, not the light-mode `#8e8e93`.
6. Quit the app entirely and relaunch: the chosen accent persists.
7. Switching Light/Dark inside Preferences still works and the accent shifts to its dark variant.

Quit with `Ctrl+C`.

- [ ] **Step 7: Full-app regression sweep**

Relaunch and click through every surface once more, confirming nothing broke: Dashboard tiles and panels, Ports table, a project's Processes / Tasks / Kanban tabs, drag a Kanban card between columns, open a task detail modal, open the Add Project modal, trigger a toast (e.g. delete a project). Everything should be legible in both Light and Dark.

- [ ] **Step 8: Commit**

```bash
cd /Users/yasharyan/Projects/jumpstart
git add frontend/src/components/Preferences.jsx frontend/src/App.jsx frontend/src/styles.css
git commit -m "feat: preferences pane with macOS accent color picker"
```

---

## Spec Coverage

| Spec section | Covered by |
|---|---|
| 1. Window chrome & vibrancy | Task 2 (Steps 1–4, 6) |
| 2. Unified sidebar & navigation | Task 3 (all steps); toolbar toggle in Task 2 Step 5 + Task 3 Step 6 |
| 3. Color system & tokens | Task 1 (all steps) |
| 4. Typography & component shapes | Task 4 (all steps) |
| 5. Preferences view | Task 5 (all steps) |
| 6. File-level change map | Tasks 1–5 collectively; `IconRail.jsx` deleted in Task 3 Step 3 |

**Spec deviation:** the spec's section 1 named a `Backdrop: mac.SidebarBackdrop` option. That field does not exist in Wails v2.13.0 (it is a Wails v3 API). Task 2 achieves the same result with `WebviewIsTransparent` + `WindowIsTranslucent` + a transparent `BackgroundColour`, which is the v2-supported path.
