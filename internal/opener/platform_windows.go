//go:build windows

package opener

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// winEditor describes one code editor JumpStart knows how to find on
// Windows: paths lists every install location worth checking, in order,
// since the same installer can land in either a per-user or a machine-wide
// location depending on how it was run.
type winEditor struct {
	id, name string
	paths    []string
}

// winCandidatePaths joins each rel install-relative path onto every base
// directory an installer might have used (skipping any base the current
// environment doesn't define).
func winCandidatePaths(rel ...string) []string {
	var out []string
	for _, base := range []string{
		os.Getenv("LOCALAPPDATA"),
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
	} {
		if base == "" {
			continue
		}
		for _, r := range rel {
			out = append(out, filepath.Join(base, r))
		}
	}
	return out
}

var winEditorCatalog = []winEditor{
	{"vscode", "Visual Studio Code", winCandidatePaths(
		filepath.Join("Programs", "Microsoft VS Code", "Code.exe"),
		filepath.Join("Microsoft VS Code", "Code.exe"),
	)},
	{"vscode-insiders", "VS Code Insiders", winCandidatePaths(
		filepath.Join("Programs", "Microsoft VS Code Insiders", "Code - Insiders.exe"),
	)},
	{"cursor", "Cursor", winCandidatePaths(
		filepath.Join("Programs", "cursor", "Cursor.exe"),
	)},
	{"windsurf", "Windsurf", winCandidatePaths(
		filepath.Join("Programs", "Windsurf", "Windsurf.exe"),
	)},
	{"antigravity", "Antigravity", winCandidatePaths(
		filepath.Join("Programs", "Antigravity", "Antigravity.exe"),
	)},
	{"sublime", "Sublime Text", winCandidatePaths(
		filepath.Join("Sublime Text", "sublime_text.exe"),
	)},
	{"notepadpp", "Notepad++", winCandidatePaths(
		filepath.Join("Notepad++", "notepad++.exe"),
	)},
}

func reveal(dir string) error {
	// explorer.exe exits with code 1 even when it opened the window, which is
	// why start() never inspects the exit code.
	return launch(dir, "could not find Windows Explorer", []candidate{
		{bin: "explorer.exe", args: []string{dirToken}},
	})
}

func openTerminal(dir string) error {
	// Windows Terminal first (it honours the user's default profile), then a
	// plain console. `start` is a cmd.exe builtin, so it needs the /C wrapper;
	// the empty "" is start's title argument, without which start would treat
	// a quoted path as the title.
	return launch(dir, "could not find a terminal to open", []candidate{
		{bin: "wt.exe", args: []string{"-d", dirToken}},
		{bin: "cmd.exe", args: []string{"/C", "start", "", "/D", dirToken, "cmd.exe"}, viaConsole: true},
	})
}

func listEditors() []AppIcon {
	var out []AppIcon
	for _, e := range winEditorCatalog {
		if p := findWinEditorPath(e); p != "" {
			out = append(out, AppIcon{ID: e.id, Name: e.name, Icon: winExeIconDataURI(p)})
		}
	}
	return out
}

func openEditor(id, dir string) error {
	for _, e := range winEditorCatalog {
		if e.id != id {
			continue
		}
		p := findWinEditorPath(e)
		if p == "" {
			return fmt.Errorf("%s is not installed", e.name)
		}
		return launch(dir, "could not launch "+e.name, []candidate{
			{bin: p, args: []string{dirToken}},
		})
	}
	return fmt.Errorf("unknown editor %q", id)
}

func fileManagerIcon() AppIcon {
	exe := filepath.Join(os.Getenv("SystemRoot"), "explorer.exe")
	return AppIcon{ID: "explorer", Name: "File Explorer", Icon: winExeIconDataURI(exe)}
}

func terminalIcon() AppIcon {
	if p, err := exec.LookPath("wt.exe"); err == nil {
		return AppIcon{ID: "terminal", Name: "Windows Terminal", Icon: winExeIconDataURI(p)}
	}
	cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	return AppIcon{ID: "terminal", Name: "Command Prompt", Icon: winExeIconDataURI(cmdPath)}
}

func findWinEditorPath(e winEditor) string {
	for _, p := range e.paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// winExeIconDataURI extracts an .exe's own associated icon via a short
// PowerShell script (PowerShell ships with every supported Windows
// version, so this needs no extra dependency or cgo) and returns it as a
// PNG data URI, or "" if extraction failed for any reason.
func winExeIconDataURI(exePath string) string {
	tmpFile, err := os.CreateTemp("", "jumpstart-icon-*.png")
	if err != nil {
		return ""
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	script := fmt.Sprintf(
		`Add-Type -AssemblyName System.Drawing; `+
			`$icon = [System.Drawing.Icon]::ExtractAssociatedIcon(%s); `+
			`if ($icon) { $icon.ToBitmap().Save(%s, [System.Drawing.Imaging.ImageFormat]::Png) }`,
		psQuote(exePath), psQuote(tmpPath),
	)
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	hideConsole(cmd)
	if err := cmd.Run(); err != nil {
		return ""
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// psQuote wraps s in single quotes for a PowerShell -Command string,
// escaping any single quote it contains (PowerShell's escape for ' inside
// a single-quoted string is two of them, '').
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
