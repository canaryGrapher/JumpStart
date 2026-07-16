// Package gitops wraps go-git to provide a small, app-friendly API for
// inspecting and operating on a project's git repository: status, init,
// fetch/pull/push (with token auth), commit, and remote management.
package gitops

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Status describes a project's repository state for the UI.
type Status struct {
	Initialized    bool   `json:"initialized"`
	HasRemote      bool   `json:"hasRemote"`
	RemoteURL      string `json:"remoteUrl"`
	RemoteName     string `json:"remoteName"`
	Branch         string `json:"branch"`
	Clean          bool   `json:"clean"`
	Ahead          int    `json:"ahead"`
	Behind         int    `json:"behind"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
}

// GetStatus reports the git state of dir. If dir is not a git repository
// it returns a non-error Status{Initialized: false} so callers can show a
// friendly "not a repo yet" state instead of handling an error.
//
// Ahead/Behind are computed against the upstream tracking branch when one
// is configured; they are best-effort and may read 0 until Fetch has been
// run to update remote-tracking refs (this function does not touch the
// network).
func GetStatus(dir string) (*Status, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return &Status{Initialized: false}, nil
		}
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	st := &Status{Initialized: true}

	head, err := repo.Head()
	if err == nil {
		if head.Name().IsBranch() {
			st.Branch = head.Name().Short()
		}
		if commit, cerr := repo.CommitObject(head.Hash()); cerr == nil {
			msg := commit.Message
			if len(msg) > 72 {
				msg = msg[:72] + "…"
			}
			// collapse to first line
			for i, c := range msg {
				if c == '\n' {
					msg = msg[:i]
					break
				}
			}
			st.LastCommit = fmt.Sprintf("%s %s", commit.Hash.String()[:7], msg)
			st.LastCommitTime = commit.Author.When.Format(time.RFC3339)
		}
	}

	if remote, rerr := repo.Remote("origin"); rerr == nil {
		st.HasRemote = true
		st.RemoteName = "origin"
		if urls := remote.Config().URLs; len(urls) > 0 {
			st.RemoteURL = urls[0]
		}
	}

	wt, werr := repo.Worktree()
	if werr == nil {
		if wst, sterr := wt.Status(); sterr == nil {
			st.Clean = wst.IsClean()
		}
	}

	// Best-effort ahead/behind against the configured upstream, using
	// only local refs (no network access here — run Fetch first for a
	// fresh comparison).
	if head != nil && st.Branch != "" {
		if ahead, behind, ok := aheadBehind(repo, st.Branch); ok {
			st.Ahead = ahead
			st.Behind = behind
		}
	}

	return st, nil
}

func aheadBehind(repo *git.Repository, branch string) (ahead, behind int, ok bool) {
	localRef, err := repo.Reference(refBranch(branch), true)
	if err != nil {
		return 0, 0, false
	}
	remoteRef, err := repo.Reference(refRemoteBranch("origin", branch), true)
	if err != nil {
		return 0, 0, false
	}
	if localRef.Hash() == remoteRef.Hash() {
		return 0, 0, true
	}

	localCommits, err := commitSet(repo, localRef.Hash())
	if err != nil {
		return 0, 0, false
	}
	remoteCommits, err := commitSet(repo, remoteRef.Hash())
	if err != nil {
		return 0, 0, false
	}
	for h := range localCommits {
		if !remoteCommits[h] {
			ahead++
		}
	}
	for h := range remoteCommits {
		if !localCommits[h] {
			behind++
		}
	}
	return ahead, behind, true
}

// commitSet walks back from hash and returns the set of commit hashes
// reachable from it, capped to avoid unbounded work on huge histories.
func commitSet(repo *git.Repository, hash plumbing.Hash) (map[plumbing.Hash]bool, error) {
	const maxCommits = 500
	out := map[plumbing.Hash]bool{}
	iter, err := repo.Log(&git.LogOptions{From: hash})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	count := 0
	for count < maxCommits {
		c, err := iter.Next()
		if err != nil {
			break
		}
		out[c.Hash] = true
		count++
	}
	return out, nil
}

// Init creates a new git repository at dir.
func Init(dir string) error {
	_, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("initializing repository: %w", err)
	}
	return nil
}

// Fetch fetches from origin using token as an HTTPS PAT.
func Fetch(dir, token string) error {
	repo, err := openRepo(dir)
	if err != nil {
		return err
	}
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       tokenAuth(token),
	})
	return wrapGitErr(err)
}

// Pull pulls the current branch from origin using token as an HTTPS PAT.
func Pull(dir, token string) error {
	repo, err := openRepo(dir)
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       tokenAuth(token),
	})
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	return wrapGitErr(err)
}

// Push pushes the current branch to origin using token as an HTTPS PAT.
func Push(dir, token string) error {
	repo, err := openRepo(dir)
	if err != nil {
		return err
	}
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       tokenAuth(token),
	})
	if errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	return wrapGitErr(err)
}

// Commit stages all changes and commits them, returning the short hash.
func Commit(dir, message, authorName, authorEmail string) (string, error) {
	repo, err := openRepo(dir)
	if err != nil {
		return "", err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	if _, err := wt.Add("."); err != nil {
		return "", fmt.Errorf("staging changes: %w", err)
	}
	if authorName == "" {
		authorName = "JumpStart"
	}
	if authorEmail == "" {
		authorEmail = "jumpstart@local"
	}
	hash, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("committing: %w", err)
	}
	return hash.String()[:7], nil
}

// Author returns the best available author name/email for dir: the
// repository's local git config if set, otherwise the user's global git
// config, otherwise empty strings (callers should fall back to a
// default).
func Author(dir string) (name, email string) {
	repo, err := git.PlainOpen(dir)
	if err == nil {
		if cfg, cerr := repo.Config(); cerr == nil {
			if cfg.User.Name != "" || cfg.User.Email != "" {
				return cfg.User.Name, cfg.User.Email
			}
		}
	}
	if global, gerr := config.LoadConfig(config.GlobalScope); gerr == nil && global != nil {
		return global.User.Name, global.User.Email
	}
	return "", ""
}

// AddRemote adds (or replaces) a remote named name pointing at url.
func AddRemote(dir, name, url string) error {
	repo, err := openRepo(dir)
	if err != nil {
		return err
	}
	// Remove existing remote of the same name, if any, so this call is
	// idempotent (acts as "connect/replace remote").
	_ = repo.DeleteRemote(name)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		return fmt.Errorf("adding remote: %w", err)
	}
	return nil
}

// RemoteURL returns the URL configured for "origin", for display/linking.
func RemoteURL(dir string) (string, error) {
	repo, err := openRepo(dir)
	if err != nil {
		return "", err
	}
	remote, err := repo.Remote("origin")
	if err != nil {
		return "", fmt.Errorf("no remote configured")
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", fmt.Errorf("no remote configured")
	}
	return urls[0], nil
}

// --- helpers ---

func openRepo(dir string) (*git.Repository, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("not a git repository")
		}
		return nil, fmt.Errorf("opening repository: %w", err)
	}
	return repo, nil
}

func tokenAuth(token string) *http.BasicAuth {
	if token == "" {
		return nil
	}
	return &http.BasicAuth{Username: "x-access-token", Password: token}
}

func wrapGitErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case errors.Is(err, git.ErrRemoteNotFound):
		return fmt.Errorf("no remote configured")
	case containsAny(msg, "authentication required", "authorization failed", "401", "403"):
		return fmt.Errorf("authentication failed — check your token")
	}
	return err
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) && indexOf(s, sub) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	n, m := len(s), len(sub)
	for i := 0; i+m <= n; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}

func refBranch(branch string) plumbing.ReferenceName {
	return plumbing.ReferenceName("refs/heads/" + branch)
}

func refRemoteBranch(remote, branch string) plumbing.ReferenceName {
	return plumbing.ReferenceName("refs/remotes/" + remote + "/" + branch)
}
