// This file adds richer, read/write git features on top of the go-git
// helpers in gitops.go by shelling out to the system `git` binary:
// branch listing, a commit graph for the timeline UI, branch
// switch/create/delete, diffs across working/staging/HEAD/remote/stash,
// and stash listing. The CLI is used here (rather than go-git) because
// merge-topology graphs, working-tree diffs and stashes are far more
// reliable through git itself, which is always present in a dev setup.
package gitops

import (
	"fmt"
	"os/exec"
	"strings"
)

// Branch describes one local (or remote-tracking) branch for the UI.
type Branch struct {
	Name      string `json:"name"`
	Current   bool   `json:"current"`
	Remote    bool   `json:"remote"`
	Upstream  string `json:"upstream"`
	Subject   string `json:"subject"`
	CommitISO string `json:"commitISO"`
}

// GraphCommit is a single node in the branch/merge timeline.
type GraphCommit struct {
	Hash    string   `json:"hash"`
	Short   string   `json:"short"`
	Parents []string `json:"parents"`
	Refs    []string `json:"refs"`
	Author  string   `json:"author"`
	ISO     string   `json:"iso"`
	Subject string   `json:"subject"`
	Merge   bool     `json:"merge"`
}

// DiffFile summarises one changed file in a diff.
type DiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// DiffResult is a parsed diff for one comparison mode.
type DiffResult struct {
	Mode  string     `json:"mode"`
	Label string     `json:"label"`
	Files []DiffFile `json:"files"`
	Patch string     `json:"patch"`
	Empty bool       `json:"empty"`
}

// Stash describes one entry from `git stash list`.
type Stash struct {
	Ref     string `json:"ref"`
	Message string `json:"message"`
}

// git runs a git subcommand in dir and returns trimmed stdout, mapping
// common failures to friendly errors.
func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimRight(string(out), "\n"), nil
}
