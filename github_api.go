package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/ghsync"
	"devdeck/internal/github"
	"devdeck/internal/gitops"
	"devdeck/internal/model"
	"devdeck/internal/secrets"
)

// ghState holds the pieces of GitHub sync that outlive a single call:
// the polling scheduler and the in-flight device authorization.
type ghState struct {
	mu        sync.Mutex
	scheduler *ghsync.Scheduler
	deviceID  string
	syncing   sync.Map // projectID -> bool, so passes never overlap
}

// --- Connection ---

// GitHubStatus describes the current connection for Settings.
type GitHubStatus struct {
	Connected  bool   `json:"connected"`
	Login      string `json:"login"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	DeviceFlow bool   `json:"deviceFlow"` // whether this build can run the device flow
	Error      string `json:"error,omitempty"`
}

// GitHubGetStatus reports whether a usable token is stored and who it
// belongs to.
func (a *App) GitHubGetStatus() (*GitHubStatus, error) {
	st := &GitHubStatus{DeviceFlow: strings.TrimSpace(GitHubClientID) != ""}
	token, err := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
	if err != nil || token == "" {
		return st, nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()

	viewer, err := github.New(token).Whoami(ctx)
	if err != nil {
		st.Error = err.Error()
		return st, nil
	}
	st.Connected = true
	st.Login = viewer.Login
	st.Name = viewer.Name
	st.AvatarURL = viewer.AvatarURL
	return st, nil
}

// GitHubStartDeviceAuth begins the OAuth device flow and returns the code
// the user types on github.com.
func (a *App) GitHubStartDeviceAuth() (*github.DeviceCode, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	code, err := github.StartDeviceFlow(ctx, GitHubClientID)
	if err != nil {
		return nil, err
	}
	a.gh().mu.Lock()
	a.gh().deviceID = code.DeviceCode
	a.gh().mu.Unlock()
	return code, nil
}

// GitHubPollDeviceAuth checks once whether the user has finished
// authorizing. It returns false while the flow is still pending, so the
// frontend can poll at the interval GitHub asked for.
func (a *App) GitHubPollDeviceAuth() (bool, error) {
	a.gh().mu.Lock()
	deviceID := a.gh().deviceID
	a.gh().mu.Unlock()
	if deviceID == "" {
		return false, errors.New("no authorization in progress")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	token, err := github.PollDeviceFlow(ctx, GitHubClientID, deviceID)
	if errors.Is(err, github.ErrAuthPending) || errors.Is(err, github.ErrSlowDown) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := secrets.SaveToken(secrets.Service, secrets.KeyGitHubToken, token); err != nil {
		return false, err
	}
	a.gh().mu.Lock()
	a.gh().deviceID = ""
	a.gh().mu.Unlock()
	return true, nil
}

// GitHubSaveToken stores a pasted personal access token, for builds
// without an OAuth client id or users who prefer a PAT.
func (a *App) GitHubSaveToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token is empty")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	if _, err := github.New(token).Whoami(ctx); err != nil {
		return err
	}
	return secrets.SaveToken(secrets.Service, secrets.KeyGitHubToken, token)
}

// GitHubDisconnect removes the stored token and stops polling.
func (a *App) GitHubDisconnect() error {
	a.gh().scheduler.Stop()
	return secrets.DeleteToken(secrets.Service, secrets.KeyGitHubToken)
}

// --- Discovery ---

// GitHubListProjects lists the boards visible to the token. An empty
// owner means the signed-in account's own boards.
func (a *App) GitHubListProjects(owner string) ([]github.Project, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()
	return client.ListOwnerProjects(ctx, owner)
}

// GitHubGetProject returns one board with every field definition, which
// is what the field editors in the task modal are built from.
func (a *App) GitHubGetProject(projectID string) (*github.Project, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()
	return client.GetProject(ctx, projectID)
}

// GitHubListRepositories lists repos the token can open issues in.
func (a *App) GitHubListRepositories() ([]github.Repository, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()
	return client.ListRepositories(ctx)
}

// GitHubSuggestRepo reads the project's git remote and returns the
// "owner/name" it points at, so linking usually needs no typing.
func (a *App) GitHubSuggestRepo(projectID string) (string, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return "", err
	}
	st, err := gitops.GetStatus(proj.Root)
	if err != nil || st == nil {
		return "", nil
	}
	return github.ParseRemote(st.RemoteURL), nil
}

// GitHubDetection is what the connection modal shows while it figures
// out how (or whether) the current project already relates to GitHub:
// is this a git checkout, does it have a remote, does that remote point
// at GitHub, and does the repository it names actually exist and is it
// reachable with the stored token.
type GitHubDetection struct {
	Connected bool `json:"connected"` // a GitHub account/token is stored

	IsGitRepo bool   `json:"isGitRepo"`
	HasRemote bool   `json:"hasRemote"`
	RemoteURL string `json:"remoteUrl,omitempty"`
	IsGitHub  bool   `json:"isGitHub"` // the remote, if any, points at github.com

	Owner    string `json:"owner,omitempty"`    // parsed from the remote
	Repo     string `json:"repo,omitempty"`     // parsed from the remote
	FullName string `json:"fullName,omitempty"` // "owner/repo"

	RepoExists bool   `json:"repoExists"` // the repo exists and the token can see it
	RepoURL    string `json:"repoUrl,omitempty"`

	// SuggestedName is the local folder's name, sanitized into something
	// GitHub will accept, for the "create a repository" flow.
	SuggestedName string `json:"suggestedName"`
}

// GitHubDetectRepo inspects a project's local git state and, when a
// GitHub remote is configured, checks whether that repository exists and
// is reachable. It is the first thing the connection modal calls, and it
// never errors for an ordinary "not connected yet" project: every check
// that cannot run just leaves its field false so the UI can show the
// next appropriate step instead of an error screen.
func (a *App) GitHubDetectRepo(projectID string) (*GitHubDetection, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return nil, err
	}
	det := &GitHubDetection{SuggestedName: github.SanitizeRepoName(filepath.Base(proj.Root))}

	if token, terr := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken); terr == nil {
		det.Connected = token != ""
	}

	st, serr := gitops.GetStatus(proj.Root)
	if serr != nil || st == nil {
		return det, nil
	}
	det.IsGitRepo = st.Initialized
	det.HasRemote = st.HasRemote
	det.RemoteURL = st.RemoteURL

	if st.HasRemote {
		if full := github.ParseRemote(st.RemoteURL); full != "" {
			det.IsGitHub = true
			det.FullName = full
			if owner, name, ok := strings.Cut(full, "/"); ok {
				det.Owner, det.Repo = owner, name
			}
		}
	}

	if det.IsGitHub && det.Connected {
		if client, cerr := a.ghClient(); cerr == nil {
			ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
			defer cancel()
			if repo, rerr := client.GetRepository(ctx, det.FullName); rerr == nil && repo != nil {
				det.RepoExists = true
				det.RepoURL = repo.URL
			}
		}
	}

	return det, nil
}

// GitHubRepoURL returns the github.com URL for a project's origin remote,
// or "" when the project has no remote or its remote isn't GitHub. It
// takes a raw directory rather than a project ID (see opener_api.go)
// because it is a cheap, local-only check: unlike GitHubDetectRepo, it
// never touches the network or requires a signed-in token, so callers
// like the project header can use it just to decide whether to show an
// "Open on GitHub" button.
func (a *App) GitHubRepoURL(dir string) (string, error) {
	st, err := gitops.GetStatus(dir)
	if err != nil || st == nil || !st.HasRemote {
		return "", nil
	}
	full := github.ParseRemote(st.RemoteURL)
	if full == "" {
		return "", nil
	}
	return "https://github.com/" + full, nil
}

// GitHubListOwners lists the signed-in account and every organization it
// belongs to, for the account/organization picker in the "connect
// GitHub" flow.
func (a *App) GitHubListOwners() ([]github.Owner, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	return client.ListOwners(ctx)
}

// GitHubBoardPresets lists JumpStart's starter layouts for a new board,
// shown when creating one. These are JumpStart's own presets, not
// GitHub's built-in template gallery — the public API has no way to
// copy one of GitHub's own template projects.
func (a *App) GitHubBoardPresets() []github.BoardPreset {
	return github.BoardPresets()
}

// GitHubCreateProject creates a new Projects v2 board under ownerID and,
// when presetKey matches one of GitHubBoardPresets, seeds its Status
// field with that preset's columns.
func (a *App) GitHubCreateProject(ownerID, title, presetKey string) (*github.Project, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	proj, err := client.CreateProject(ctx, ownerID, title)
	if err != nil {
		return nil, err
	}
	if preset, ok := github.BoardPresetByKey(presetKey); ok {
		if perr := client.ApplyStatusPreset(ctx, proj.ID, preset.Columns); perr != nil {
			// The board itself was created successfully; only the column
			// relabeling failed, so surface that distinctly rather than
			// losing the board the user just made.
			return proj, fmt.Errorf("board created, but could not apply the %s layout: %w", preset.Label, perr)
		}
	}
	return proj, nil
}

// GitHubImportRepoItems adds a repository's open issues and/or pull
// requests to a board, for the "import items from repository" step when
// creating one.
func (a *App) GitHubImportRepoItems(projectID, owner, repo string, includeIssues, includePRs bool) (*github.ImportResult, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	return client.ImportRepoItems(ctx, projectID, owner, repo, includeIssues, includePRs)
}

// GitHubCreateRepository creates a new repository under owner and wires
// it up as the project's git remote. When the project already has local
// commits they are pushed immediately, so the freshly created repository
// is never left empty and the modal can continue straight into syncing.
// description and gitignoreTemplate are both optional (pass "" to skip
// either) — see CreateRepository's doc comment for why a gitignore
// template should only be offered when the project has no local commits
// yet.
func (a *App) GitHubCreateRepository(projectID, owner, ownerType, name, description, gitignoreTemplate string, private bool) (*github.CreatedRepository, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return nil, err
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	repo, err := client.CreateRepository(ctx, owner, ownerType, name, description, gitignoreTemplate, private)
	if err != nil {
		return nil, err
	}

	st, serr := gitops.GetStatus(proj.Root)
	if serr == nil && st != nil {
		if !st.Initialized {
			if ierr := gitops.Init(proj.Root); ierr != nil {
				return repo, fmt.Errorf("repository created, but could not initialize the local project: %w", ierr)
			}
		}
		if aerr := gitops.AddRemote(proj.Root, "origin", repo.CloneURL); aerr != nil {
			return repo, fmt.Errorf("repository created, but could not connect the local project to it: %w", aerr)
		}
		if st.LastCommit != "" || st.Branch != "" {
			token, _ := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
			// Bounded separately from the 30s repo-creation timeout above:
			// pushing existing local history can legitimately take longer,
			// but it must still be bounded, or a stalled network/auth
			// handshake would hang this call (and the "Creating..." button
			// driving it) forever.
			pushCtx, pushCancel := context.WithTimeout(a.ctx, 2*time.Minute)
			perr := gitops.Push(pushCtx, proj.Root, token)
			pushCancel()
			if perr != nil {
				return repo, fmt.Errorf("repository created and connected, but pushing your local history failed: %w", perr)
			}
		}
	}

	return repo, nil
}

// GitHubSetRemote points a project's local "origin" remote at an
// existing GitHub repository. It is only ever called after the user has
// explicitly confirmed the link in the UI — unlike GitHubCreateRepository
// (which is always wiring up a repo it just created), this can silently
// replace an existing remote, so it never runs on its own.
func (a *App) GitHubSetRemote(projectID, fullName string) error {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return err
	}
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return fmt.Errorf("repository is required")
	}

	st, serr := gitops.GetStatus(proj.Root)
	if serr == nil && st != nil && !st.Initialized {
		if ierr := gitops.Init(proj.Root); ierr != nil {
			return fmt.Errorf("could not initialize the local project: %w", ierr)
		}
	}
	return gitops.AddRemote(proj.Root, "origin", "https://github.com/"+fullName+".git")
}

// ghSyncRepo resolves the "owner/name" and cached node id of the
// repository a project's GitHub sync currently points at. The Issues/PRs
// panel and "new issue"/"new pull request" both key off this rather than
// the git remote, since Repo is what the user explicitly confirmed as
// this project's GitHub home.
func (a *App) ghSyncRepo(projectID string) (owner, name, repoID string, err error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return "", "", "", err
	}
	if proj.GitHub == nil || proj.GitHub.Repo == "" {
		return "", "", "", errors.New("this project is not linked to a GitHub repository yet")
	}
	o, n, ok := strings.Cut(proj.GitHub.Repo, "/")
	if !ok {
		return "", "", "", fmt.Errorf("invalid repository %q", proj.GitHub.Repo)
	}
	return o, n, proj.GitHub.RepoID, nil
}

// GitHubListIssues lists issues from the project's linked repository,
// most recently updated first. states filters by "OPEN"/"CLOSED"; pass
// an empty slice for both.
func (a *App) GitHubListIssues(projectID string, states []string, limit int) ([]github.Issue, error) {
	owner, name, _, err := a.ghSyncRepo(projectID)
	if err != nil {
		return nil, err
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	return client.ListIssues(ctx, owner, name, states, limit)
}

// GitHubListPullRequests lists pull requests from the project's linked
// repository the same way.
func (a *App) GitHubListPullRequests(projectID string, states []string, limit int) ([]github.PullRequest, error) {
	owner, name, _, err := a.ghSyncRepo(projectID)
	if err != nil {
		return nil, err
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	return client.ListPullRequests(ctx, owner, name, states, limit)
}

// ghRepoID resolves a repository's node id, using the cached one on the
// sync config when present so creating an issue or PR is usually a
// single request rather than two.
func (a *App) ghRepoID(ctx context.Context, client *github.Client, owner, name, cached string) (string, error) {
	if cached != "" {
		return cached, nil
	}
	repo, err := client.GetRepository(ctx, owner+"/"+name)
	if err != nil {
		return "", err
	}
	return repo.ID, nil
}

// GitHubCreateIssue opens a new issue on the project's linked repository.
func (a *App) GitHubCreateIssue(projectID, title, body string) (*github.CreatedIssue, error) {
	owner, name, repoID, err := a.ghSyncRepo(projectID)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("a title is required")
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	repoID, err = a.ghRepoID(ctx, client, owner, name, repoID)
	if err != nil {
		return nil, err
	}
	return client.CreateIssue(ctx, repoID, title, body)
}

// BranchRef is one branch relevant to opening a pull request, merged
// from two sources: the local clone (which knows whether it exists here
// and whether this is the current checkout) and a live query against
// GitHub (which knows whether it has actually been pushed). Status
// summarizes whether the two copies, when a branch exists in both
// places, agree.
type BranchRef struct {
	Name    string `json:"name"`
	Local   bool   `json:"local"`
	Remote  bool   `json:"remote"`
	Current bool   `json:"current"` // the checked-out local branch

	LocalSHA  string `json:"localSha,omitempty"`
	RemoteSHA string `json:"remoteSha,omitempty"`

	// Status is "in-sync", "local-ahead", "remote-ahead", or "diverged"
	// when the branch exists both locally and on GitHub; empty when it
	// only exists in one place, since there is nothing to compare.
	Status string `json:"status,omitempty"`
}

// GitHubListBranches lists every branch relevant to opening a pull
// request: local branches from the clone, live branches from GitHub, and
// which of those is which — so the base/head pickers can show a branch
// that only exists on GitHub (pushed from elsewhere), one that only
// exists locally (not pushed yet), and whether a branch present in both
// places is actually in sync, rather than only offering what happens to
// be checked out here.
func (a *App) GitHubListBranches(projectID string) ([]BranchRef, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return nil, err
	}
	owner, name, _, err := a.ghSyncRepo(projectID)
	if err != nil {
		return nil, err
	}

	local, lerr := gitops.ListBranches(proj.Root)
	if lerr != nil {
		local = nil
	}

	var remote []github.RemoteBranch
	if client, cerr := a.ghClient(); cerr == nil {
		ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
		defer cancel()
		// Best-effort: if the live lookup fails (offline, rate limited),
		// the picker still works off local branches alone.
		remote, _ = client.ListBranches(ctx, owner, name)
	}

	byName := map[string]*BranchRef{}
	var order []string
	entry := func(n string) *BranchRef {
		if b, ok := byName[n]; ok {
			return b
		}
		b := &BranchRef{Name: n}
		byName[n] = b
		order = append(order, n)
		return b
	}

	localDates := map[string]string{}
	for _, b := range local {
		if b.Remote || b.Name == "" {
			continue // remote-tracking refs are what the live query below replaces
		}
		br := entry(b.Name)
		br.Local = true
		br.Current = b.Current
		br.LocalSHA = b.SHA
		localDates[b.Name] = b.CommitISO
	}
	for _, r := range remote {
		br := entry(r.Name)
		br.Remote = true
		br.RemoteSHA = shortSHA(r.OID)
		if br.Local {
			br.Status = compareBranchState(br.LocalSHA, br.RemoteSHA, localDates[r.Name], r.CommittedDate)
		}
	}

	out := make([]BranchRef, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out, nil
}

// compareBranchState summarizes whether a branch's local and remote
// copies agree, using commit dates to guess a direction when the SHAs
// differ. It is a heuristic, not a true ahead/behind count — good enough
// to warn "these have diverged, pull or push first" without walking the
// commit graph for a branch that may not even be checked out.
func compareBranchState(localSHA, remoteSHA, localDate, remoteDate string) string {
	if localSHA != "" && localSHA == remoteSHA {
		return "in-sync"
	}
	lt, lerr := time.Parse(time.RFC3339, localDate)
	rt, rerr := time.Parse(time.RFC3339, remoteDate)
	switch {
	case lerr != nil || rerr != nil:
		return "diverged"
	case lt.After(rt):
		return "local-ahead"
	case rt.After(lt):
		return "remote-ahead"
	default:
		return "diverged"
	}
}

// shortSHA truncates a commit hash to the 7-character form used
// throughout JumpStart's git UI.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// GitHubCreatePullRequest opens a pull request on the project's linked
// repository from an already-pushed branch.
func (a *App) GitHubCreatePullRequest(projectID, base, head, title, body string, draft bool) (*github.CreatedPullRequest, error) {
	owner, name, repoID, err := a.ghSyncRepo(projectID)
	if err != nil {
		return nil, err
	}
	base, head, title = strings.TrimSpace(base), strings.TrimSpace(head), strings.TrimSpace(title)
	if base == "" || head == "" {
		return nil, errors.New("base and head branches are required")
	}
	if title == "" {
		return nil, errors.New("a title is required")
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	repoID, err = a.ghRepoID(ctx, client, owner, name, repoID)
	if err != nil {
		return nil, err
	}
	return client.CreatePullRequest(ctx, repoID, base, head, title, body, draft)
}

func (a *App) gh() *ghState {
	a.ghOnce.Do(func() {
		state := &ghState{}
		state.scheduler = ghsync.NewScheduler(func(ctx context.Context, projectID string) (*ghsync.Result, error) {
			return a.runSync(projectID, false)
		})
		a.ghShared = state
	})
	return a.ghShared
}

func (a *App) ghClient() (*github.Client, error) {
	token, err := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, errors.New("not connected to GitHub, connect in Settings")
	}
	return github.New(token), nil
}

func (a *App) ghProject(projectID string) (*model.Project, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ID == projectID {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project not found")
}

// emit is a thin wrapper so sync events survive a nil context in tests.
func (a *App) emit(event string, data ...any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, event, data...)
}
