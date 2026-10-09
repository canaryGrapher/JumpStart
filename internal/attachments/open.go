package attachments

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// run starts a viewer process without waiting for it to exit (viewers such as
// Quick Look stay open until the user closes them) and reaps it in the
// background. Arguments are passed as an argv list, never through a shell.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not launch %s: %w", name, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Open opens a stored file with the system's default application.
func Open(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return run("open", path)
	case "windows":
		return run("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return run("xdg-open", path)
	}
}

// Preview opens a quick look at the file. On macOS images and PDFs open in
// Preview; every other type uses Quick Look, since Preview rejects them. Other
// platforms fall back to the default application.
func Preview(path, mimeType string) error {
	if runtime.GOOS != "darwin" {
		return Open(path)
	}
	if IsImage(mimeType) || strings.EqualFold(mimeType, "application/pdf") {
		return run("open", "-a", "Preview", path)
	}
	return run("qlmanage", "-p", path)
}

// Reveal shows the file in Finder / Explorer / the file manager.
func Reveal(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return run("open", "-R", path)
	case "windows":
		return run("explorer", "/select,", path)
	default:
		return run("xdg-open", pathDir(path))
	}
}

func pathDir(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i > 0 {
		return p[:i]
	}
	return "."
}
