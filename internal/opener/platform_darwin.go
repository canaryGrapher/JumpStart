//go:build darwin

package opener

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// macAppDirs are the locations an app bundle can be installed in, checked
// ahead of the user's own ~/Applications by findMacApp.
var macAppDirs = []string{
	"/Applications",
	"/Applications/Utilities",
}

// macEditor describes one code editor JumpStart knows how to find on macOS.
type macEditor struct {
	id, name, appName string
}

// macEditorCatalog is deliberately broad — editors ship in wildly different
// bundle names, and every entry here is otherwise inert: an editor that
// isn't installed just never shows up in ListEditors.
var macEditorCatalog = []macEditor{
	{"vscode", "Visual Studio Code", "Visual Studio Code.app"},
	{"vscode-insiders", "VS Code Insiders", "Visual Studio Code - Insiders.app"},
	{"cursor", "Cursor", "Cursor.app"},
	{"windsurf", "Windsurf", "Windsurf.app"},
	{"antigravity", "Antigravity IDE", "Antigravity IDE.app"},
	{"zed", "Zed", "Zed.app"},
	{"sublime", "Sublime Text", "Sublime Text.app"},
	{"textmate", "TextMate", "TextMate.app"},
	{"bbedit", "BBEdit", "BBEdit.app"},
	{"nova", "Nova", "Nova.app"},
	{"webstorm", "WebStorm", "WebStorm.app"},
	{"intellij", "IntelliJ IDEA", "IntelliJ IDEA.app"},
	{"pycharm", "PyCharm", "PyCharm.app"},
	{"goland", "GoLand", "GoLand.app"},
	{"rubymine", "RubyMine", "RubyMine.app"},
	{"androidstudio", "Android Studio", "Android Studio.app"},
	{"xcode", "Xcode", "Xcode.app"},
}

const noOpen = "could not find the macOS `open` command"

func reveal(dir string) error {
	return launch(dir, noOpen, []candidate{
		{bin: "open", args: []string{dirToken}},
	})
}

func openTerminal(dir string) error {
	name, _ := resolvedMacTerminal()
	return launch(dir, noOpen, []candidate{
		{bin: "open", args: []string{"-a", name, dirToken}},
	})
}

func listEditors() []AppIcon {
	var out []AppIcon
	for _, e := range macEditorCatalog {
		if p := findMacApp(e.appName); p != "" {
			out = append(out, AppIcon{ID: e.id, Name: e.name, Icon: macAppIconDataURI(p)})
		}
	}
	return out
}

func openEditor(id, dir string) error {
	for _, e := range macEditorCatalog {
		if e.id != id {
			continue
		}
		p := findMacApp(e.appName)
		if p == "" {
			return fmt.Errorf("%s is not installed", e.name)
		}
		return launch(dir, noOpen, []candidate{
			{bin: "open", args: []string{"-a", p, dirToken}},
		})
	}
	return fmt.Errorf("unknown editor %q", id)
}

func fileManagerIcon() AppIcon {
	return AppIcon{ID: "finder", Name: "Finder", Icon: macAppIconDataURI("/System/Library/CoreServices/Finder.app")}
}

func terminalIcon() AppIcon {
	name, path := resolvedMacTerminal()
	icon := ""
	if path != "" {
		icon = macAppIconDataURI(path)
	}
	return AppIcon{ID: "terminal", Name: name, Icon: icon}
}

// macAppSearchDirs is macAppDirs plus the user's own ~/Applications, which
// takes precedence since a per-user install shadows a system-wide one.
func macAppSearchDirs() []string {
	dirs := macAppDirs
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append([]string{filepath.Join(home, "Applications")}, dirs...)
	}
	return dirs
}

// findMacApp returns the full path to appName (e.g. "Cursor.app") under the
// usual Application directories, or "" if it isn't installed in any of them.
func findMacApp(appName string) string {
	for _, d := range macAppSearchDirs() {
		p := filepath.Join(d, appName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// resolvedMacTerminal returns both the bare app name openTerminal passes to
// `open -a` and the full bundle path (for icon extraction), preferring the
// first installed app from macTerminals and falling back to whichever
// location ships Terminal.app on this OS version.
func resolvedMacTerminal() (name, path string) {
	for _, app := range macTerminals {
		if p := findMacApp(app); p != "" {
			return strings.TrimSuffix(app, ".app"), p
		}
	}
	for _, p := range []string{
		"/System/Applications/Utilities/Terminal.app",
		"/Applications/Utilities/Terminal.app",
	} {
		if _, err := os.Stat(p); err == nil {
			return "Terminal", p
		}
	}
	return "Terminal", ""
}

// macAppIconDataURI extracts an app bundle's own icon and returns it as a
// PNG data URI, or "" if the icon couldn't be found or converted (missing
// `sips`/`defaults`, an unusual bundle layout, ...) — callers treat that as
// "no icon available", not an error.
func macAppIconDataURI(appPath string) string {
	icns := macBundleIconPath(appPath)
	if icns == "" {
		return ""
	}
	return pngDataURIFromICNS(icns)
}

// macBundleIconPath resolves an app bundle's icon file. It asks the
// bundle's Info.plist for CFBundleIconFile via `defaults read`, which
// handles both XML and binary plists (a plain file read would only handle
// the former); if that lookup fails it falls back to the first .icns in
// Resources, since most apps ship only one for the app itself (document-type
// icons are named differently).
func macBundleIconPath(appPath string) string {
	infoPlist := filepath.Join(appPath, "Contents", "Info")
	if out, err := exec.Command("defaults", "read", infoPlist, "CFBundleIconFile").Output(); err == nil {
		name := strings.TrimSpace(string(out))
		if name != "" {
			if !strings.HasSuffix(strings.ToLower(name), ".icns") {
				name += ".icns"
			}
			p := filepath.Join(appPath, "Contents", "Resources", name)
			if _, serr := os.Stat(p); serr == nil {
				return p
			}
		}
	}
	matches, _ := filepath.Glob(filepath.Join(appPath, "Contents", "Resources", "*.icns"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// pngDataURIFromICNS converts a .icns icon to a small PNG via macOS's
// built-in `sips` tool (no extra dependency) and base64-encodes it for
// direct use as an <img src>.
func pngDataURIFromICNS(icnsPath string) string {
	tmpFile, err := os.CreateTemp("", "jumpstart-icon-*.png")
	if err != nil {
		return ""
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// -Z 64 caps the largest side at 64px — plenty for a toolbar icon and
	// far smaller than the multi-megapixel images some .icns files embed.
	if err := exec.Command("sips", "-s", "format", "png", "-Z", "64", icnsPath, "--out", tmpPath).Run(); err != nil {
		return ""
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}
