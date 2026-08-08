// Package opener hands a directory to the host desktop: the platform file
// manager (Finder, Explorer, Nautilus, ...) or a terminal emulator opened at
// that directory.
//
// Every call shells out to a program owned by the desktop environment and
// returns as soon as it has been started. JumpStart never waits for the
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
