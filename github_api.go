package main

import (
	"context"
	"errors"
	"fmt"
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
