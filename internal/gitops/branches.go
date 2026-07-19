package gitops

import "strings"

// unit separator used to join fields inside a git --format string; safe
// because it will not appear in branch names or commit subjects.
const fieldSep = "\x1f"

// ListBranches returns local branches (with the current one flagged)
// followed by remote-tracking branches.
func ListBranches(dir string) ([]Branch, error) {
	format := strings.Join([]string{
		"%(HEAD)", "%(refname:short)", "%(upstream:short)",
		"%(contents:subject)", "%(committerdate:iso-strict)",
	}, fieldSep)

	local, err := gitCmd(dir, "branch", "--format="+format)
	if err != nil {
		return nil, err
	}
	branches := parseBranches(local, false)

	remote, err := gitCmd(dir, "branch", "-r", "--format="+format)
	if err == nil {
		branches = append(branches, parseBranches(remote, true)...)
	}
	return branches, nil
}

func parseBranches(out string, remote bool) []Branch {
	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, fieldSep)
		for len(f) < 5 {
			f = append(f, "")
		}
		name := f[1]
		// Skip the symbolic "origin/HEAD -> origin/main" entry.
		if remote && strings.Contains(name, "->") {
			continue
		}
		branches = append(branches, Branch{
			Name:      name,
			Current:   strings.TrimSpace(f[0]) == "*",
			Remote:    remote,
			Upstream:  f[2],
			Subject:   f[3],
			CommitISO: f[4],
		})
	}
	return branches
}

// GraphLog returns up to limit commits across all branches in topological
// order, with parent hashes and ref decorations, for the timeline UI.
func GraphLog(dir string, limit int) ([]GraphCommit, error) {
	if limit <= 0 {
		limit = 80
	}
	format := strings.Join([]string{
		"%H", "%h", "%P", "%D", "%an", "%aI", "%s",
	}, fieldSep)

	out, err := gitCmd(dir,
		"log", "--all", "--date-order",
		"--pretty=format:"+format, "-n", itoa(limit),
	)
	if err != nil {
		return nil, err
	}

	var commits []GraphCommit
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, fieldSep)
		for len(f) < 7 {
			f = append(f, "")
		}
		parents := splitFields(f[2])
		commits = append(commits, GraphCommit{
			Hash:    f[0],
			Short:   f[1],
			Parents: parents,
			Refs:    parseRefs(f[3]),
			Author:  f[4],
			ISO:     f[5],
			Subject: f[6],
			Merge:   len(parents) > 1,
		})
	}
	return commits, nil
}

func splitFields(s string) []string {
	var out []string
	for _, p := range strings.Fields(s) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseRefs turns git's "%D" decoration string into clean ref labels,
// dropping the "HEAD -> " prefix and tag/ref qualifiers.
func parseRefs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var refs []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		p = strings.TrimPrefix(p, "HEAD -> ")
		p = strings.TrimPrefix(p, "tag: ")
		if p != "" && p != "HEAD" {
			refs = append(refs, p)
		}
	}
	return refs
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// Checkout switches the working tree to an existing branch.
func Checkout(dir, branch string) error {
	if strings.TrimSpace(branch) == "" {
		return errEmpty("branch")
	}
	_, err := gitCmd(dir, "checkout", branch)
	return err
}

// CreateBranch creates a new branch from HEAD; when checkout is true it
// also switches to it.
func CreateBranch(dir, name string, checkout bool) error {
	if strings.TrimSpace(name) == "" {
		return errEmpty("branch name")
	}
	if checkout {
		_, err := gitCmd(dir, "checkout", "-b", name)
		return err
	}
	_, err := gitCmd(dir, "branch", name)
	return err
}

// DeleteBranch deletes a local branch. force uses -D (drops unmerged
// work) instead of the safe -d.
func DeleteBranch(dir, name string, force bool) error {
	if strings.TrimSpace(name) == "" {
		return errEmpty("branch name")
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := gitCmd(dir, "branch", flag, name)
	return err
}
