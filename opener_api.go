package main

import (
	"time"

	"devdeck/internal/opener"
)

// Reveal / terminal bindings. Both take a raw directory — the project root or
// a process's working directory — rather than an ID, because the frontend
// already holds those paths and the two call sites (project header, process
// card) would otherwise need two lookups each.

// OpenInFileManager shows a directory in the platform's file manager
// (Finder, Explorer, Nautilus, ...).
func (a *App) OpenInFileManager(dir string) error {
	return a.openOp("file_manager", func() error { return opener.Reveal(dir) })
}

// OpenInTerminal opens a terminal window at a directory.
func (a *App) OpenInTerminal(dir string) error {
	return a.openOp("terminal", func() error { return opener.OpenTerminal(dir) })
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
