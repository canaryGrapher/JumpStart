// Per-file working tree state: the staged/unstaged status list behind the
// Git Changes modal's file lists, plus the staging and per-file diff
// operations the modal's Stage/Unstage buttons and diff pane call.
package gitops

import "strings"

// FileChange describes one path's git status. A path can be both staged
// and unstaged at once (e.g. staged a modification, then edited the file
// again), so the two are independent flags rather than a single state.
type FileChange struct {
	Path          string `json:"path"`
	OldPath       string `json:"oldPath,omitempty"` // rename/copy source
	Staged        bool   `json:"staged"`
	Unstaged      bool   `json:"unstaged"`
	StagedLabel   string `json:"stagedLabel,omitempty"`
	UnstagedLabel string `json:"unstagedLabel,omitempty"`
	Conflicted    bool   `json:"conflicted"`
}

// WorkingChanges lists every path with staged and/or unstaged changes,
// parsed from `git status --porcelain=v1 -z`.
func WorkingChanges(dir string) ([]FileChange, error) {
	entries, err := gitCmdZ(dir, "status", "--porcelain=v1", "-z")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(entries), nil
}

// parsePorcelain turns the NUL-separated entries from `git status -z`
// into FileChanges. Each entry is "XY PATH"; renames and copies are
// followed by a second entry holding the original path.
func parsePorcelain(entries []string) []FileChange {
	var changes []FileChange
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		x, y := e[0], e[1]
		path := e[3:]

		var oldPath string
		if x == 'R' || x == 'C' {
			i++
			if i < len(entries) {
				oldPath = entries[i]
			}
		}

		fc := FileChange{Path: path, OldPath: oldPath}

		// Unmerged (conflicted) paths use a distinct set of XY codes
		// rather than the usual index/worktree split.
		if isConflictCode(x, y) {
			fc.Conflicted = true
			fc.Unstaged = true
			fc.UnstagedLabel = "conflicted"
			changes = append(changes, fc)
			continue
		}

		if x == '?' && y == '?' {
			fc.Unstaged = true
			fc.UnstagedLabel = "untracked"
			changes = append(changes, fc)
			continue
		}

		if x != ' ' {
			fc.Staged = true
			fc.StagedLabel = statusLabel(x)
		}
		if y != ' ' {
			fc.Unstaged = true
			fc.UnstagedLabel = statusLabel(y)
		}
		changes = append(changes, fc)
	}
	return changes
}

func isConflictCode(x, y byte) bool {
	switch {
	case x == 'U' || y == 'U':
		return true
	case x == 'A' && y == 'A':
		return true
	case x == 'D' && y == 'D':
		return true
	}
	return false
}

func statusLabel(code byte) string {
	switch code {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "type-changed"
	default:
		return "modified"
	}
}

// StageFile adds a single path to the index.
func StageFile(dir, path string) error {
	if strings.TrimSpace(path) == "" {
		return errEmpty("path")
	}
	_, err := gitCmd(dir, "add", "--", path)
	return err
}

// UnstageFile removes a single path from the index without touching the
// working tree. Falls back to `git rm --cached` when HEAD doesn't exist
// yet (nothing committed), where `git reset` has nothing to reset to.
func UnstageFile(dir, path string) error {
	if strings.TrimSpace(path) == "" {
		return errEmpty("path")
	}
	if _, err := gitCmd(dir, "reset", "-q", "--", path); err != nil {
		if isNoHeadErr(err) {
			_, err2 := gitCmd(dir, "rm", "--cached", "-q", "--", path)
			return err2
		}
		return err
	}
	return nil
}

// StageAll stages every pending change, including untracked files.
func StageAll(dir string) error {
	_, err := gitCmd(dir, "add", "-A")
	return err
}

// UnstageAll clears the index back to HEAD without touching the working
// tree (or, before the first commit, un-adds everything).
func UnstageAll(dir string) error {
	if _, err := gitCmd(dir, "reset", "-q"); err != nil {
		if isNoHeadErr(err) {
			_, err2 := gitCmd(dir, "rm", "--cached", "-q", "-r", ".")
			return err2
		}
		return err
	}
	return nil
}

func isNoHeadErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "ambiguous argument 'head'")
}

// FileDiff returns the unified diff for one file: staged (index vs HEAD)
// when staged is true, otherwise unstaged (working tree vs index). A
// brand-new untracked file has nothing in the index to diff against, so
// it's compared with /dev/null instead to show it as all-additions.
func FileDiff(dir, path string, staged bool) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errEmpty("path")
	}

	if !staged {
		changes, err := WorkingChanges(dir)
		if err == nil {
			for _, c := range changes {
				if c.Path == path && c.UnstagedLabel == "untracked" {
					return gitCmdAllowDiff(dir, "diff", "--no-index", "--", "/dev/null", path)
				}
			}
		}
	}

	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	return gitCmd(dir, args...)
}
