//go:build windows

package opener

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
