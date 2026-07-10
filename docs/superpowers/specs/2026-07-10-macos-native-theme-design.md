# macOS-Native Theme Conversion — Design

Date: 2026-07-10
App: DevDeck (Wails v2 + React), local dev process manager

## Goal

Convert the current "Mind Bridge" design language (blue-lavender gradient background,
frosted-glass pill buttons, circular icon rail) into a design that follows Apple's
Human Interface Guidelines for Mac apps — full native depth, not just a color swap.

## 1. Window chrome & vibrancy (Go layer)

`main.go` gains a `Mac` options block on the `options.App` struct:

- `TitleBar: mac.TitleBarHiddenInset()` — hides the title text, keeps native traffic
  lights floating with inset spacing (Mail/Notes/Xcode pattern).
- `WebviewIsTransparent: true`, `WindowIsTranslucent: true`, and a sidebar-appropriate
  `Backdrop` — makes the whole window translucent so the sidebar can show real desktop
  blur (NSVisualEffectView-equivalent).

Compensating in CSS: only the sidebar panel stays visually transparent (letting the
native blur through); the main content pane gets a solid opaque background so it
doesn't look see-through. The top ~28px of the sidebar becomes a Wails-draggable
region (`--wails-draggable: drag` or equivalent) so the window remains movable with no
title bar left to grab.

This block only affects macOS builds — Wails ignores the `Mac` options struct on other
platforms, so no cross-platform regression.

## 2. Unified sidebar & navigation

The icon rail (`IconRail.jsx`) and the projects panel (`Sidebar.jsx`) merge into one
macOS-style sidebar, matching Mail/Notes/Slack:

- **Top** (below the drag zone): unlabeled rows for **Dashboard** and **Ports**, each
  with a small line icon — top-level sidebar destinations, no section header.
- **PROJECTS** section header (small uppercase gray label, Finder "Favorites" style),
  followed by project rows — monogram avatar + name + subprocess count (unchanged
  content from today).
- **Row selection**: solid accent-color fill with white text when selected (standard
  sidebar/table selection style), rounded-rect ~6px corner radius, not a pill.
- **Bottom**: a plain "+ Add Project" row (not a pill button) and a small gear icon
  that opens Preferences.
- **Collapse**: a sidebar-toggle button moves to the main content toolbar (top-left,
  next to the title) and fully hides/shows the sidebar (width → 0), replacing the
  current icon-only collapsed mode. Matches Finder/Xcode/Mail `⌘⌥S` behavior.

`IconRail.jsx` is deleted; its two nav rows move into `Sidebar.jsx`. `App.jsx` gains
the sidebar-toggle button and a `collapsed` boolean driving a width transition (not an
icon-mode layout).

## 3. Color system & design tokens

Replace the blue-lavender gradient palette with macOS system semantic colors:

- **Background**: no gradient — main content is a flat system background
  (`#f6f6f7` light / `#1e1e1e` dark). Sidebar stays near-transparent to show native
  blur.
- **Text**: `labelColor`-equivalents — `#1d1d1f` / `#f5f5f5` primary,
  `#6e6e73` / `#98989d` secondary/dim.
- **Separators**: hairline `#d1d1d6` (light) / `#38383a` (dark), replacing the current
  soft-blue borders.
- **Accent**: user-selectable — the 8 standard macOS accent colors (blue `#007AFF`
  light / `#0A84FF` dark, purple, pink, red, orange, yellow, green, graphite),
  defaulting to blue. Every current `--navy` usage (buttons, selected rows, links)
  switches to `var(--accent)` so it responds to the picker.
- **Shadows**: subtler — `0 1px 3px rgba(0,0,0,.12)` range, replacing the current
  large soft blue-tinted shadows.
- Status colors (green/red/yellow for running/error/warning) keep their meaning, retuned
  to macOS system green/red/yellow hex values.

## 4. Typography & component shapes

- **Type scale**: large title ~22px (topbar), headline ~15px semibold (card titles),
  body 13px (macOS default 13pt), caption 11px (sub-labels, badges). Font stack stays
  `-apple-system, BlinkMacSystemFont, ...`.
- **Corner radii**: buttons/inputs/segmented controls ~6-7px, cards/panels/modals
  ~10-12px — down from the current pill-everywhere look. Pills are kept only where
  macOS actually uses capsules (status/notification-style badges); priority tags, dep
  badges etc. become small rounded-rect chips (~5px).
- **Buttons**: primary = solid accent fill, white text; secondary = flat gray fill with
  hairline border (not white-on-white); destructive keeps red text with a subtle red
  hover fill. None are pill-shaped anymore.
- **Cards**: solid opaque fill with a hairline border instead of
  `backdrop-filter: blur()` — blur becomes exclusively the sidebar's job (native
  vibrancy), so cards no longer fake it.
- **Segmented control** (theme toggle): tightens to macOS's actual look — gray track,
  sliding rounded-rect selected segment (~6px radius), not a full pill.
- Kanban board, log viewer, tables, and forms inherit the same token set — no
  structural changes beyond radius/shadow/color updates.

## 5. Preferences view

A new modal styled like a macOS Settings pane (single "Appearance" section, ~420px
wide, not full-screen), opened from the gear icon in the sidebar footer. Mirrors
System Settings → Appearance:

- **Appearance**: the existing Light/Dark/Auto segmented control, moved here from the
  main toolbar.
- **Accent color**: 8 circular swatches (blue/purple/pink/red/orange/yellow/green/
  graphite); the selected one gets a ring/check. Clicking sets `--accent` and persists
  to `localStorage` via a new `useAccent` hook (mirrors the existing `useTheme` hook
  pattern in `App.jsx`).

The main toolbar loses `ThemeToggle` entirely; it's replaced by the sidebar-toggle
button (section 2) and the view title.

## 6. File-level change map

- `main.go` — add `Mac` window options (titlebar inset, translucency)
- `frontend/src/styles.css` — full token rewrite + shape/radius/shadow updates across
  all existing rules (same class names throughout, so no JSX churn beyond what's
  listed below)
- `frontend/src/components/Sidebar.jsx` — absorb nav rows from IconRail, drop
  icon-only collapsed mode, add gear/Preferences trigger
- `frontend/src/components/IconRail.jsx` — deleted
- `frontend/src/components/ThemeToggle.jsx` — kept, relocated into new Preferences
  component
- `frontend/src/components/Preferences.jsx` — new: appearance segmented control +
  accent swatches
- `frontend/src/App.jsx` — merge sidebar-toggle into toolbar, add `useAccent` hook,
  wire Preferences modal, remove `IconRail` usage

## Out of scope

- Per-window vibrancy variation (only the sidebar gets native blur; no other panel
  does)
- A full separate OS-level Preferences window (kept as an in-app modal instead)
- Any change to app behavior/logic — this is a visual/theme conversion only
