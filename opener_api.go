package main

import (
	"time"

	"devdeck/internal/opener"
)

// Reveal / terminal / editor bindings. Directory-taking ones use a raw
// directory — the project root or a process's working directory — rather
// than an ID, because the frontend already holds those paths and the call
// sites (project header, process card) would otherwise need two lookups
// each.

// OpenInFileManager shows a directory in the platform's file manager
// (Finder, Explorer, Nautilus, ...).
func (a *App) OpenInFileManager(dir string) error {
	return a.openOp("file_manager", func() error { return opener.Reveal(dir) })
}

// OpenInTerminal opens a terminal window at a directory.
func (a *App) OpenInTerminal(dir string) error {
	return a.openOp("terminal", func() error { return opener.OpenTerminal(dir) })
}

// ListEditors returns every code editor installed on this machine, each
// with its own real application icon, for the "open in editor" dropdown.
func (a *App) ListEditors() []opener.AppIcon {
	return opener.ListEditors()
}

// OpenInEditor opens dir in the editor identified by id (one of the ids
// ListEditors returned).
func (a *App) OpenInEditor(id, dir string) error {
	return a.openOp("editor:"+id, func() error { return opener.OpenInEditor(id, dir) })
}

// FileManagerIcon returns the real icon of the platform's file manager, for
// the "open in Finder/Explorer/Files" button.
func (a *App) FileManagerIcon() opener.AppIcon {
	return opener.FileManagerIcon()
}

// TerminalIcon returns the real icon of the terminal OpenInTerminal would
// launch.
func (a *App) TerminalIcon() opener.AppIcon {
	return opener.TerminalIcon()
}

// openOp runs one launcher binding and reports it as a single
// external_open_performed event with a `target` property, following the same
// one-event-per-feature rule as git and docker. The directory itself is never
// reported: it is a user path.
func (a *App) openOp(target string, fn func() error) error {
	start := time.Now()
	err := fn()
	a.track("external_open_performed", outcome(start, err, map[string]any{"target": target}))
	return err
}
