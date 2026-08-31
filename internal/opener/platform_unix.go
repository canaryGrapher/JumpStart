//go:build !darwin && !windows

package opener

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// fileManagers are probed in order. xdg-open comes first because it routes to
// whatever the desktop environment has registered for inode/directory; the
// named managers are the fallback for minimal setups without xdg-utils.
var fileManagers = []candidate{
	{bin: "xdg-open", args: []string{dirToken}},
	{bin: "nautilus", args: []string{dirToken}},
	{bin: "dolphin", args: []string{dirToken}},
	{bin: "nemo", args: []string{dirToken}},
	{bin: "thunar", args: []string{dirToken}},
	{bin: "caja", args: []string{dirToken}},
	{bin: "pcmanfm", args: []string{dirToken}},
	{bin: "gio", args: []string{"open", dirToken}},
}

// terminals are probed in order, each with its own way of being told a
// working directory. x-terminal-emulator leads on Debian-family systems
// because it points at the user's configured default, but it takes no cwd
// flag, so it relies on the inherited working directory that start() sets.
var terminals = []candidate{
	{bin: "x-terminal-emulator", args: nil},
	{bin: "gnome-terminal", args: []string{"--working-directory=" + dirToken}},
	{bin: "konsole", args: []string{"--workdir", dirToken}},
	{bin: "xfce4-terminal", args: []string{"--working-directory=" + dirToken}},
	{bin: "mate-terminal", args: []string{"--working-directory=" + dirToken}},
	{bin: "tilix", args: []string{"-w", dirToken}},
	{bin: "terminator", args: []string{"--working-directory=" + dirToken}},
	{bin: "ghostty", args: []string{"--working-directory=" + dirToken}},
	{bin: "kitty", args: []string{"--directory", dirToken}},
	{bin: "alacritty", args: []string{"--working-directory", dirToken}},
	{bin: "wezterm", args: []string{"start", "--cwd", dirToken}},
	{bin: "foot", args: []string{"--working-directory=" + dirToken}},
	{bin: "urxvt", args: []string{"-cd", dirToken}},
	{bin: "xterm", args: nil},
}

// terminalIconHints maps a terminal binary to the icon names its own
// .desktop file is likely to declare, most-specific first, with
// "utilities-terminal" (the icon-theme standard name for "a terminal") as
// the last resort every icon theme is expected to provide.
var terminalIconHints = map[string][]string{
	"gnome-terminal":      {"org.gnome.Terminal", "utilities-terminal"},
	"konsole":             {"org.kde.konsole", "utilities-terminal"},
	"xfce4-terminal":      {"org.xfce.terminal", "utilities-terminal"},
	"mate-terminal":       {"mate-terminal", "utilities-terminal"},
	"tilix":               {"com.gexperts.Tilix", "utilities-terminal"},
	"terminator":          {"terminator", "utilities-terminal"},
	"ghostty":             {"com.mitchellh.ghostty", "utilities-terminal"},
	"kitty":               {"kitty", "utilities-terminal"},
	"alacritty":           {"Alacritty", "utilities-terminal"},
	"wezterm":             {"org.wezfurlong.wezterm", "utilities-terminal"},
	"foot":                {"foot", "utilities-terminal"},
	"urxvt":               {"utilities-terminal"},
	"xterm":               {"xterm", "utilities-terminal"},
	"x-terminal-emulator": {"utilities-terminal"},
}

// terminalDisplayNames gives a handful of common terminals a proper name;
// anything not listed just falls back to its bin name (e.g. "urxvt"),
// which is a fine label for the less common ones.
var terminalDisplayNames = map[string]string{
	"gnome-terminal":      "GNOME Terminal",
	"konsole":             "Konsole",
	"xfce4-terminal":      "Xfce Terminal",
	"mate-terminal":       "MATE Terminal",
	"tilix":               "Tilix",
	"terminator":          "Terminator",
	"ghostty":             "Ghostty",
	"kitty":               "Kitty",
	"alacritty":           "Alacritty",
	"wezterm":             "WezTerm",
	"foot":                "Foot",
	"xterm":               "XTerm",
	"x-terminal-emulator": "Terminal",
}

// unixEditor describes one code editor JumpStart knows how to find on
// Linux/BSD: bin is the launcher on PATH, iconHints are the icon-theme
// names its .desktop file is likely to use (most specific first).
type unixEditor struct {
	id, name, bin string
	iconHints     []string
}

var unixEditorCatalog = []unixEditor{
	{"vscode", "Visual Studio Code", "code", []string{"vscode", "code"}},
	{"vscode-insiders", "VS Code Insiders", "code-insiders", []string{"code-insiders"}},
	{"vscodium", "VSCodium", "codium", []string{"vscodium"}},
	{"cursor", "Cursor", "cursor", []string{"cursor"}},
	{"windsurf", "Windsurf", "windsurf", []string{"windsurf"}},
	{"antigravity", "Antigravity", "antigravity", []string{"antigravity"}},
	{"zed", "Zed", "zed", []string{"dev.zed.Zed", "zed"}},
	{"sublime", "Sublime Text", "subl", []string{"sublime-text", "com.sublimetext.three"}},
	{"kate", "Kate", "kate", []string{"kate"}},
	{"gnome-text-editor", "Text Editor", "gnome-text-editor", []string{"org.gnome.TextEditor"}},
	{"gedit", "Text Editor", "gedit", []string{"org.gnome.gedit", "gedit"}},
}

func reveal(dir string) error {
	return launch(dir, "no file manager found (install xdg-utils or a file manager such as Nautilus)", fileManagers)
}

func openTerminal(dir string) error {
	// $TERMINAL is the conventional override on Linux and BSD; honouring it
	// means a user with an unusual emulator does not need us to know about it.
	return launch(dir, "no terminal emulator found (set $TERMINAL to the one you use)",
		append(envCandidate("TERMINAL"), terminals...))
}

func listEditors() []AppIcon {
	var out []AppIcon
	for _, e := range unixEditorCatalog {
		if _, err := exec.LookPath(e.bin); err == nil {
			out = append(out, AppIcon{ID: e.id, Name: e.name, Icon: findLinuxIcon(e.iconHints...)})
		}
	}
	return out
}

func openEditor(id, dir string) error {
	for _, e := range unixEditorCatalog {
		if e.id != id {
			continue
		}
		return launch(dir, fmt.Sprintf("could not find %s", e.name), []candidate{
			{bin: e.bin, args: []string{dirToken}},
		})
	}
	return fmt.Errorf("unknown editor %q", id)
}

// fileManagerIcon always reports Nautilus (GNOME's "Files"), since that is
// what most desktop environments in JumpStart's Linux support matrix use —
// reveal() itself still goes through xdg-open first and may launch a
// different manager on a non-GNOME desktop, so this is a best-effort label
// rather than a guarantee of which app actually opens.
func fileManagerIcon() AppIcon {
	return AppIcon{ID: "nautilus", Name: "Files", Icon: findLinuxIcon("org.gnome.Nautilus", "nautilus", "system-file-manager")}
}

func terminalIcon() AppIcon {
	for _, c := range append(envCandidate("TERMINAL"), terminals...) {
		if _, err := exec.LookPath(c.bin); err != nil {
			continue
		}
		hints := terminalIconHints[c.bin]
		if len(hints) == 0 {
			hints = []string{"utilities-terminal"}
		}
		name := terminalDisplayNames[c.bin]
		if name == "" {
			name = c.bin
		}
		return AppIcon{ID: "terminal", Name: name, Icon: findLinuxIcon(hints...)}
	}
	return AppIcon{ID: "terminal", Name: "Terminal"}
}

// linuxIconThemeDirs are searched in order for a freedesktop icon theme
// (hicolor is the spec-mandated fallback theme every implementation ships);
// pixmaps is the older, flat convention some packages still install into.
var linuxIconThemeDirs = []string{
	"/usr/share/icons/hicolor",
	"/usr/local/share/icons/hicolor",
	"/usr/share/icons/Adwaita",
}

var linuxIconSizes = []string{"256x256", "128x128", "96x96", "64x64", "48x48", "32x32"}

// findLinuxIcon looks for any of names (most specific first) across the
// usual icon-theme locations and returns the first match as a data URI, or
// "" if none of the themes installed on this machine have it.
func findLinuxIcon(names ...string) string {
	searchDirs := append([]string{}, linuxIconThemeDirs...)
	if home, err := os.UserHomeDir(); err == nil {
		searchDirs = append(searchDirs,
			filepath.Join(home, ".local/share/icons/hicolor"),
			filepath.Join(home, ".icons/hicolor"),
		)
	}

	for _, base := range searchDirs {
		for _, sz := range linuxIconSizes {
			for _, n := range names {
				if d := readIconFile(filepath.Join(base, sz, "apps", n+".png")); d != "" {
					return d
				}
			}
		}
		for _, n := range names {
			if d := readIconFile(filepath.Join(base, "scalable", "apps", n+".svg")); d != "" {
				return d
			}
		}
	}

	// Flat, unthemed fallback some distros still populate. (XPM icons show
	// up here too on older systems, but browsers can't render XPM as an
	// <img>, so only PNG is worth trying.)
	for _, n := range names {
		if d := readIconFile(filepath.Join("/usr/share/pixmaps", n+".png")); d != "" {
			return d
		}
	}
	return ""
}

func readIconFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	mime := "image/png"
	if strings.HasSuffix(path, ".svg") {
		mime = "image/svg+xml"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}
