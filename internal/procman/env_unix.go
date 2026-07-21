//go:build !windows

package procman

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// GUI apps launched from Finder inherit a minimal PATH and miss user tool
// locations (homebrew, nvm, asdf, etc.). resolvedEnv returns os.Environ with
// PATH replaced by the user's real login-shell PATH, so commands like `npm`
// resolve the same way they do in a terminal.
func resolvedEnv() []string {
	env := os.Environ()
	path := userPath()
	if path == "" {
		return env
	}

	out := env[:0]
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "PATH="+path)
}

var (
	pathOnce   sync.Once
	cachedPath string
)

// userPath resolves PATH once by asking the login shell, then merges in common
// fallback bin directories so tools are found even if the shell probe fails.
func userPath() string {
	pathOnce.Do(func() {
		cachedPath = mergePaths(loginShellPath(), fallbackDirs())
	})
	return cachedPath
}

// loginShellPath runs the user's login+interactive shell to capture the PATH
// their profile/rc files build (where homebrew, nvm, asdf, etc. are added).
func loginShellPath() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// -l -i loads both login and interactive config. Printing a sentinel lets
	// us ignore any banner/prompt noise the shell may emit.
	cmd := exec.CommandContext(ctx, shell, "-l", "-i", "-c", `printf "__JS_PATH__%s__JS_END__" "$PATH"`)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	s := string(out)
	start := strings.Index(s, "__JS_PATH__")
	end := strings.Index(s, "__JS_END__")
	if start == -1 || end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(s[start+len("__JS_PATH__") : end])
}

// fallbackDirs are the usual homes for user-installed CLI tools on macOS/Linux.
func fallbackDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
	}
	if home != "" {
		dirs = append(dirs,
			home+"/.local/bin",
			home+"/.cargo/bin",
			home+"/go/bin",
		)
	}
	return dirs
}

// mergePaths joins a base PATH with extra dirs, dropping duplicates and
// non-existent directories while preserving order.
func mergePaths(base string, extra []string) string {
	seen := map[string]bool{}
	var out []string
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" || seen[dir] {
			return
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			return
		}
		seen[dir] = true
		out = append(out, dir)
	}

	if base != "" {
		for _, dir := range strings.Split(base, string(os.PathListSeparator)) {
			add(dir)
		}
	}
	for _, dir := range extra {
		add(dir)
	}
	return strings.Join(out, string(os.PathListSeparator))
}
