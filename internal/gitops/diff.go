package gitops

import (
	"fmt"
	"strconv"
	"strings"
)

func errEmpty(what string) error { return fmt.Errorf("%s is required", what) }

// diffArgs maps a UI mode to the `git diff` arguments that produce it.
// Modes:
//
//	working  — unstaged changes (working tree vs index)
//	staging  — staged changes (index vs HEAD)
//	worktree — all uncommitted changes (working tree vs HEAD)
//	remote   — local branch vs its upstream / origin
//	stash    — the latest stash vs its base
func diffArgs(mode string) (label string, args []string) {
	switch mode {
	case "working":
		return "Working tree vs staging", []string{"diff"}
	case "staging":
		return "Staging vs last commit", []string{"diff", "--cached"}
	case "worktree":
		return "All changes vs last commit", []string{"diff", "HEAD"}
	case "remote":
		return "Local vs remote", []string{"diff", "@{upstream}...HEAD"}
	case "stash":
		return "Latest stash", []string{"stash", "show", "-p", "stash@{0}"}
	default:
		return "Working tree vs staging", []string{"diff"}
	}
}

// Diff returns a parsed diff for the given comparison mode, including a
// per-file summary and the raw unified patch text.
func Diff(dir, mode string) (*DiffResult, error) {
	label, base := diffArgs(mode)

	res := &DiffResult{Mode: mode, Label: label}

	// Numstat summary (skip for stash, which handles its own args).
	if mode != "stash" {
		statArgs := append(append([]string{}, base...), "--numstat")
		if stat, err := gitCmd(dir, statArgs...); err == nil {
			res.Files = parseNumstat(stat)
		}
	}

	patch, err := gitCmd(dir, base...)
	if err != nil {
		// A missing upstream or empty stash should read as "nothing",
		// not a hard error.
		if isNoDiffErr(err) {
			res.Empty = true
			return res, nil
		}
		return nil, err
	}
	res.Patch = patch
	res.Empty = strings.TrimSpace(patch) == ""

	if mode == "stash" && res.Patch != "" {
		res.Files = parseStatNames(res.Patch)
	}
	return res, nil
}

func isNoDiffErr(err error) bool {
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "no upstream") ||
		strings.Contains(m, "no stash entries") ||
		strings.Contains(m, "unknown revision") ||
		strings.Contains(m, "ambiguous argument")
}

func parseNumstat(out string) []DiffFile {
	var files []DiffFile
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		add, _ := strconv.Atoi(parts[0])
		del, _ := strconv.Atoi(parts[1])
		files = append(files, DiffFile{
			Path:      parts[2],
			Additions: add,
			Deletions: del,
			Status:    "modified",
		})
	}
	return files
}

// parseStatNames is a fallback file list derived from a unified patch
// (used for stash, where numstat isn't collected separately).
func parseStatNames(patch string) []DiffFile {
	var files []DiffFile
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			files = append(files, DiffFile{
				Path:   strings.TrimPrefix(line, "+++ b/"),
				Status: "modified",
			})
		}
	}
	return files
}

// ListStashes returns the entries from `git stash list`.
func ListStashes(dir string) ([]Stash, error) {
	out, err := gitCmd(dir, "stash", "list", "--pretty=%gd"+fieldSep+"%s")
	if err != nil {
		return nil, err
	}
	var stashes []Stash
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.SplitN(line, fieldSep, 2)
		s := Stash{Ref: f[0]}
		if len(f) > 1 {
			s.Message = f[1]
		}
		stashes = append(stashes, s)
	}
	return stashes, nil
}
