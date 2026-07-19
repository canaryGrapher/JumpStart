package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// downloadClient allows large binaries and slow links; separate from the
// short-timeout client used for the JSON update check.
var downloadClient = &http.Client{Timeout: 10 * time.Minute}

// Apply downloads the given release asset and replaces the running app in
// place. On macOS the whole .app bundle is swapped (to keep its signature
// intact); on Windows and Linux the executable file is replaced. Call
// Relaunch afterwards to start the new version.
//
// progress, if non-nil, is called with an integer percentage (0-100).
func Apply(asset Asset, progress func(pct int)) error {
	exe, err := currentExecutable()
	if err != nil {
		return err
	}
	archivePath, err := downloadToTemp(asset, progress)
	if err != nil {
		return err
	}
	defer os.Remove(archivePath)

	if runtime.GOOS == "darwin" {
		return applyDarwin(archivePath, exe)
	}
	return applyBinary(archivePath, exe)
}

// currentExecutable resolves the running binary path through any symlinks.
func currentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// downloadToTemp streams the asset to a temp file, reporting progress.
func downloadToTemp(asset Asset, progress func(pct int)) (string, error) {
	req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: server returned %s", resp.Status)
	}

	total := asset.Size
	if total <= 0 {
		total = resp.ContentLength
	}

	tmp, err := os.CreateTemp("", "jumpstart-update-*"+archiveExt(asset.Name))
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	pw := &progressWriter{total: total, report: progress}
	if _, err := io.Copy(tmp, io.TeeReader(resp.Body, pw)); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if progress != nil {
		progress(100)
	}
	return tmp.Name(), nil
}

func archiveExt(name string) string {
	if strings.HasSuffix(name, ".tar.gz") {
		return ".tar.gz"
	}
	return filepath.Ext(name)
}

// progressWriter reports download progress as a percentage.
type progressWriter struct {
	total   int64
	written int64
	last    int
	report  func(pct int)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.written += int64(n)
	if w.report != nil && w.total > 0 {
		pct := int(w.written * 100 / w.total)
		if pct > w.last {
			w.last = pct
			w.report(pct)
		}
	}
	return n, nil
}

// applyBinary replaces a single executable (Windows/Linux) with the one
// extracted from the archive.
func applyBinary(archivePath, exe string) error {
	dir := filepath.Dir(exe)
	newBin := filepath.Join(dir, "."+filepath.Base(exe)+".new")
	defer os.Remove(newBin)

	var err error
	if strings.HasSuffix(archivePath, ".tar.gz") {
		err = extractTarEntry(archivePath, filepath.Base(exe), newBin)
	} else {
		err = extractZipEntry(archivePath, filepath.Base(exe), newBin)
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(newBin, 0o755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		// A running .exe can't be overwritten, but it can be renamed.
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
		if err := os.Rename(newBin, exe); err != nil {
			os.Rename(old, exe) // roll back
			return err
		}
		return nil
	}
	// Unix: renaming over the running binary is safe (the process keeps
	// the old inode open until it exits).
	return os.Rename(newBin, exe)
}

// applyDarwin swaps the entire .app bundle so the new build's code signature
// stays valid.
func applyDarwin(archivePath, exe string) error {
	appPath := bundlePath(exe)
	if appPath == "" {
		// Not running from a bundle (e.g. bare binary): fall back.
		return applyBinary(archivePath, exe)
	}
	parent := filepath.Dir(appPath)
	work, err := os.MkdirTemp(parent, ".jumpstart-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	if err := unzipAll(archivePath, work); err != nil {
		return err
	}
	newApp, err := findApp(work)
	if err != nil {
		return err
	}

	backup := appPath + ".old"
	os.RemoveAll(backup)
	if err := os.Rename(appPath, backup); err != nil {
		return err
	}
	if err := os.Rename(newApp, appPath); err != nil {
		os.Rename(backup, appPath) // roll back
		return err
	}
	os.RemoveAll(backup)
	return nil
}

// bundlePath returns the .app path for an executable inside a macOS bundle,
// or "" if the executable is not inside one.
func bundlePath(exe string) string {
	const marker = ".app/Contents/MacOS/"
	if i := strings.Index(exe, marker); i >= 0 {
		return exe[:i+len(".app")]
	}
	return ""
}

// findApp returns the first top-level *.app directory inside dir.
func findApp(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .app bundle found in update archive")
}

// Relaunch starts the (now updated) app as a new process. The caller should
// quit the current process immediately after.
func Relaunch() error {
	exe, err := currentExecutable()
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		if app := bundlePath(exe); app != "" {
			return exec.Command("open", "-n", app).Start()
		}
	}
	cmd := exec.Command(exe)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Start()
}
