// Package opener hands a directory to the host desktop: the platform file
// manager (Finder, Explorer, Nautilus, ...) or a terminal emulator opened at
// that directory. It also detects installed code editors and extracts real
// application icons (file manager, terminal, editors) so the UI can show the
// actual icon of whatever it is about to launch instead of a generic glyph.
//
// Every open call shells out to a program owned by the desktop environment
// and returns as soon as it has been started. JumpStart never waits for the
// window to close, and never treats the child's exit code as its own — some
// launchers (notably Windows Explorer) exit non-zero even on success.
package opener

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppIcon describes one external application the UI can offer: a stable id
// used to invoke it again, a display name, and its real application icon
// rendered as a "data:image/...;base64,..." URI. Icon is "" when JumpStart
// could not extract one (a conversion tool is missing, the icon theme
// doesn't have it, ...) — every other field is still valid, so callers
// should fall back to a generic glyph rather than treat that as an error.
type AppIcon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// Reveal opens dir in the platform's file manager.
func Reveal(dir string) error {
	d, err := resolveDir(dir)
	if err != nil {
		return err
	}
	return reveal(d)
}

// OpenTerminal opens a terminal window whose working directory is dir.
func OpenTerminal(dir string) error {
	d, err := resolveDir(dir)
	if err != nil {
		return err
	}
	return openTerminal(d)
}

// ListEditors returns every code editor JumpStart found installed on this
// machine, each with its own real application icon, so the UI can offer a
// proper "open with" choice instead of hardcoding a single editor.
func ListEditors() []AppIcon {
	return listEditors()
}

// OpenInEditor opens dir in the editor identified by id (one of the ids
// ListEditors returned).
func OpenInEditor(id, dir string) error {
	d, err := resolveDir(dir)
	if err != nil {
		return err
	}
	return openEditor(id, d)
}

// FileManagerIcon returns the real icon of the platform's file manager
// (Finder, File Explorer, Nautilus, ...) — the same one Reveal opens.
func FileManagerIcon() AppIcon {
	return fileManagerIcon()
}

// TerminalIcon returns the real icon of the terminal OpenTerminal would
// launch.
func TerminalIcon() AppIcon {
	return terminalIcon()
}

// resolveDir normalises a stored path into an existing absolute directory.
// A path pointing at a file resolves to its parent folder, so a project root
// that was saved as a file path still opens somewhere sensible instead of
// failing.
func resolveDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("no directory set for this item")
	}

	if home, err := os.UserHomeDir(); err == nil {
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~"+string(filepath.Separator)) {
			path = filepath.Join(home, path[2:])
		}
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s does not exist", abs)
		}
		return "", err
	}
	if !info.IsDir() {
		return filepath.Dir(abs), nil
	}
	return abs, nil
}
