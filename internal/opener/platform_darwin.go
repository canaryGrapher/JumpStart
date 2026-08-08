//go:build darwin

package opener

import (
	"os"
	"path/filepath"
)

// macTerminals are the terminal bundles we prefer ahead of the built-in
// Terminal.app. The list is deliberately short: `open -a App <dir>` only
// lands in the right folder for terminals that register as folder handlers,
// and one that silently opens in $HOME instead is worse than not using it.
// `open -a` also errors on a missing app rather than falling back, so each
// bundle is probed on disk before it is chosen.
var macTerminals = []string{
	"iTerm.app",
	"Ghostty.app",
}

// macAppDirs are the locations a terminal bundle can be installed in.
var macAppDirs = []string{
	"/Applications",
	"/Applications/Utilities",
}

const noOpen = "could not find the macOS `open` command"

func reveal(dir string) error {
	return launch(dir, noOpen, []candidate{
		{bin: "open", args: []string{dirToken}},
	})
}

func openTerminal(dir string) error {
	return launch(dir, noOpen, []candidate{
		{bin: "open", args: []string{"-a", preferredMacTerminal(), dirToken}},
	})
}

// preferredMacTerminal returns the first installed app from macTerminals,
// falling back to Terminal.app, which ships with the OS.
func preferredMacTerminal() string {
	dirs := macAppDirs
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append([]string{filepath.Join(home, "Applications")}, dirs...)
	}
	for _, app := range macTerminals {
		for _, d := range dirs {
			if _, err := os.Stat(filepath.Join(d, app)); err == nil {
				return app
			}
		}
	}
	return "Terminal"
}
