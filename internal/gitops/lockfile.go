// Recovery for the classic "Unable to create '.../index.lock': File
// exists" failure: git leaves index.lock behind while it holds the index,
// and removes it when done. If the git process is killed, crashes, or the
// app is force-quit mid-operation, the lock file survives and blocks
// every subsequent git command in the repository until someone deletes it
// by hand. This file lets the UI do that deletion safely instead of
// sending the user to a terminal.
package gitops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// staleLockAge is how old index.lock must be before RemoveIndexLock will
// touch it. A real git process holds the lock only for the duration of
// one operation, so a lock younger than this is more likely to belong to
// something still running than to be abandoned.
const staleLockAge = 3 * time.Second

// IndexLockPath resolves the path to the index.lock file for dir's
// repository via `git rev-parse --git-dir`, so it resolves correctly
// whether dir is the repository root or a linked worktree (where ".git"
// is a file pointing at the real git directory rather than the directory
// itself).
func IndexLockPath(dir string) (string, error) {
	gitDir, err := gitCmd(dir, "rev-parse", "--git-dir")
	if err != nil {
		return "", fmt.Errorf("resolving git directory: %w", err)
	}
	gitDir = strings.TrimSpace(gitDir)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	return filepath.Join(gitDir, "index.lock"), nil
}

// HasIndexLock reports whether dir's repository currently has an
// index.lock file present, for callers that want to decide whether to
// offer a "remove lock" action without attempting the removal.
func HasIndexLock(dir string) bool {
	path, err := IndexLockPath(dir)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// RemoveIndexLock deletes a stale .git/index.lock left behind by a
// crashed or force-quit git process. It refuses to remove a lock file
// younger than staleLockAge, since that one may belong to a git process
// that is still genuinely running — removing it out from under that
// process could corrupt the index.
func RemoveIndexLock(dir string) error {
	path, err := IndexLockPath(dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no lock file found — the repository isn't locked")
		}
		return fmt.Errorf("checking lock file: %w", err)
	}
	if age := time.Since(info.ModTime()); age < staleLockAge {
		return fmt.Errorf("the lock file was created %s ago — a git process may still be running; wait a moment and try again", age.Round(100*time.Millisecond))
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing lock file: %w", err)
	}
	return nil
}

// IsIndexLockErr reports whether err (or its string form) looks like the
// "index.lock ... File exists" failure git raises when a stale lock file
// is blocking the operation. Used by callers that want to offer a
// "remove lock and retry" action alongside the raw error.
func IsIndexLockErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "index.lock") && strings.Contains(msg, "file exists")
}
