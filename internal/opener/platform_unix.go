//go:build !darwin && !windows

package opener

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

func reveal(dir string) error {
	return launch(dir, "no file manager found (install xdg-utils or a file manager such as Nautilus)", fileManagers)
}

func openTerminal(dir string) error {
	// $TERMINAL is the conventional override on Linux and BSD; honouring it
	// means a user with an unusual emulator does not need us to know about it.
	return launch(dir, "no terminal emulator found (set $TERMINAL to the one you use)",
		append(envCandidate("TERMINAL"), terminals...))
}
