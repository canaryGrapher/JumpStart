package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"encoding/json"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
	"devdeck/internal/banner"
	"devdeck/internal/chatstore"
	"devdeck/internal/codectx"
	"devdeck/internal/config"
	"devdeck/internal/deps"
	"devdeck/internal/detect"
	"devdeck/internal/docker"
	"devdeck/internal/github"
	"devdeck/internal/gitops"
	"devdeck/internal/model"
	"devdeck/internal/procman"
	"devdeck/internal/release"
	"devdeck/internal/secrets"
	"devdeck/internal/store"
	"devdeck/internal/sysinfo"
	"devdeck/internal/testrunner"
	"devdeck/internal/update"
)

// App is the Wails-bound API used by the frontend.
type App struct {
	ctx        context.Context
	store      *store.Store
	manager    *procman.Manager
	scriptRuns *scriptRuns
	analytics  *analytics.Client
	// procStarts records when each process was started, so process_stopped
	// can report uptime without the frontend having to pass it back in.
	procStarts sync.Map // procID -> time.Time
	// stopping marks processes the user asked to stop, so their non-zero
	// exit is not misreported as a crash.
	stopping sync.Map // procID -> bool
	// shuttingDown suppresses crash reporting while the app is quitting.
	shuttingDown atomic.Bool
	// lastDetect remembers the most recent auto-detect scan, so a project
	// saved right after one can be attributed to detection.
	lastDetect atomic.Value // detection
	// updating guards overlapping InstallUpdate calls (auto-download from
	// the banner plus a Settings re-check must not race the swap).
	updating atomic.Bool
	// ghOnce/ghShared hold the GitHub sync scheduler, built lazily so a
	// user who never links a board pays nothing for it.
	ghOnce   sync.Once
	ghShared *ghState
}

func NewApp() *App {
	return &App{scriptRuns: newScriptRuns()}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.manager = procman.New(func(event string, data ...interface{}) {
		// Hooking the emitter is how process_crashed gets reported: a
		// process that dies on its own never goes back through a binding,
		// so there is no call site to instrument.
		a.trackProcessEvent(event, data...)
		runtime.EventsEmit(a.ctx, event, data...)
	})
	s, err := store.New()
	if err != nil {
		panic(err)
	}
	a.store = s

	// Analytics comes up after the store, because app_launched reports how
	// many projects the install has.
	a.initAnalytics()
	a.trackLaunch()
}

func (a *App) Shutdown(ctx context.Context) {
	// Set before StopAll: every process is about to exit non-zero, and none
	// of those are crashes.
	a.shuttingDown.Store(true)
	a.manager.StopAll()
	a.trackClose()
	// Bounded so an unreachable network on quit costs the app_closed event,
	// not the user's patience. Anything undelivered is queued to disk and
	// goes out on the next launch.
	a.analytics.Close(2 * time.Second)
}

// --- Projects ---

func (a *App) GetProjects() ([]model.Project, error) {
	return a.store.Load()
}

func (a *App) SaveProject(p model.Project) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	found := false
	var before model.Project
	for i := range projects {
		if projects[i].ID == p.ID {
			before = projects[i]
			projects[i] = p
			found = true
			break
		}
	}
	if !found {
		projects = append(projects, p)
	}
	err = a.store.Save(projects)
	a.trackProjectSaved(p, before, found, err)
	return err
}

func (a *App) DeleteProject(id string) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	out := projects[:0]
	var removed model.Project
	for _, p := range projects {
		if p.ID != id {
			out = append(out, p)
			continue
		}
		removed = p
	}
	// Drop the project's sidecar files too, so a deleted project leaves no
	// orphaned code index or chat history behind in ~/.jumpstart.
	codectx.Evict(id)
	_ = codectx.Delete(id)
	_ = chatstore.DeleteProject(id)
	err = a.store.Save(out)
	a.track("project_deleted", map[string]any{
		"project_ref":   a.ref(id),
		"process_count": len(removed.Processes),
		"use_count":     removed.UseCount,
		"succeeded":     err == nil,
	})
	return err
}

// SetProjectFavorite pins or unpins a project. Favorited projects are grouped
// at the top of the sidebar. It is a targeted write so toggling a star never
// races with an in-flight edit of the rest of the project.
func (a *App) SetProjectFavorite(id string, favorite bool) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	for i := range projects {
		if projects[i].ID == id {
			projects[i].Favorite = favorite
			err := a.store.Save(projects)
			a.track("project_favorited", map[string]any{
				"project_ref": a.ref(id),
				"favorite":    favorite,
				"succeeded":   err == nil,
			})
			return err
		}
	}
	return fmt.Errorf("project %s not found", id)
}

// --- Processes ---

func (a *App) StartProcess(projectID, procID string) error {
	p, err := a.findProcess(projectID, procID)
	if err != nil {
		return err
	}
	start := time.Now()
	err = a.manager.Start(*p)
	a.trackProcessStarted(projectID, *p, "single", start, err)
	if err != nil {
		return err
	}
	a.touchUsage(projectID)
	return nil
}

func (a *App) StopProcess(procID string) error {
	a.noteIntentionalStop(procID)
	err := a.manager.Stop(procID)
	a.trackProcessStopped(procID, "single", err)
	return err
}

func (a *App) StartAll(projectID string) []string {
	var errs []string
	projects, err := a.store.Load()
	if err != nil {
		return []string{err.Error()}
	}
	for _, proj := range projects {
		if proj.ID != projectID {
			continue
		}
		for _, proc := range proj.Processes {
			start := time.Now()
			err := a.manager.Start(proc)
			a.trackProcessStarted(projectID, proc, "start_all", start, err)
			if err != nil {
				errs = append(errs, proc.Name+": "+err.Error())
			}
		}
	}
	a.touchUsage(projectID)
	return errs
}

// touchUsage records that a project was just used (for dashboard
// "recent" and "most used" lists).
func (a *App) touchUsage(projectID string) {
	projects, err := a.store.Load()
	if err != nil {
		return
	}
	for i := range projects {
		if projects[i].ID == projectID {
			projects[i].LastUsedAt = time.Now().UnixMilli()
			projects[i].UseCount++
			_ = a.store.Save(projects)
			return
		}
	}
}

func (a *App) StopAll(projectID string) {
	projects, err := a.store.Load()
	if err != nil {
		return
	}
	for _, proj := range projects {
		if proj.ID != projectID {
			continue
		}
		for _, proc := range proj.Processes {
			a.noteIntentionalStop(proc.ID)
			err := a.manager.Stop(proc.ID)
			a.trackProcessStopped(proc.ID, "stop_all", err)
		}
	}
}

func (a *App) GetStatus(procID string) model.Status {
	return a.manager.Status(procID)
}

func (a *App) GetLogs(procID string) []string {
	return a.manager.Logs(procID)
}

// --- Resource usage ---

// Usage bundles system stats with per-running-subprocess stats.
type Usage struct {
	System sysinfo.SystemUsage          `json:"system"`
	Procs  map[string]sysinfo.ProcUsage `json:"procs"`
}

func (a *App) GetUsage() Usage {
	sys, procs := sysinfo.Snapshot(a.manager.RunningPIDs())
	return Usage{System: sys, Procs: procs}
}

// --- Ports ---

// PortEntry maps one listening port to its process and project.
type PortEntry struct {
	Port        int    `json:"port"`
	PID         int    `json:"pid"`
	ProcID      string `json:"procId"`
	ProcName    string `json:"procName"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
}

// GetPortMap lists every port in use by managed processes.
func (a *App) GetPortMap() ([]PortEntry, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	var out []PortEntry
	for _, proj := range projects {
		for _, proc := range proj.Processes {
			st := a.manager.Status(proc.ID)
			if !st.Running {
				continue
			}
			for _, port := range st.Ports {
				out = append(out, PortEntry{
					Port: port, PID: st.PID,
					ProcID: proc.ID, ProcName: proc.Name,
					ProjectID: proj.ID, ProjectName: proj.Name,
				})
			}
		}
	}
	return out, nil
}

// --- Tasks (project management) ---

func (a *App) UpdateTasks(projectID string, tasks []model.Task) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	for i := range projects {
		if projects[i].ID == projectID {
			before := projects[i].Tasks
			projects[i].Tasks = tasks
			linked := projects[i].GitHub != nil && projects[i].GitHub.Enabled
			err := a.store.Save(projects)
			if err == nil {
				a.trackTaskChanges(projectID, before, tasks)
				// A local edit pushes straight away rather than waiting for
				// the next poll, which is what makes the board feel live in
				// the direction the user can actually see.
				if linked {
					go func() { _, _ = a.runSync(projectID, false) }()
				}
			}
			return err
		}
	}
	return fmt.Errorf("project not found")
}

// UpdateSprints replaces a project's sprint list. Sprint order is taken
// from the slice position so the roadmap sequence stays authoritative.
func (a *App) UpdateSprints(projectID string, sprints []model.Sprint) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	for i := range sprints {
		sprints[i].Order = i
	}
	for i := range projects {
		if projects[i].ID == projectID {
			before := projects[i].Sprints
			projects[i].Sprints = sprints
			err := a.store.Save(projects)
			if err == nil {
				a.trackSprintChanges(projectID, before, sprints)
			}
			return err
		}
	}
	return fmt.Errorf("project not found")
}

// --- Conf file import ---

// GetImportPath returns the conf file location shown in the UI.
func (a *App) GetImportPath() (string, error) {
	return config.ImportPath()
}

// SetNativeTheme syncs the native window/vibrancy appearance to the
// frontend's resolved theme ("light", "dark", or anything else for
// system-default), so AppKit's sidebar vibrancy matches the CSS theme
// instead of tracking the OS appearance independently of it.
func (a *App) SetNativeTheme(mode string) {
	// Once per session per mode: the frontend calls this on every resolved
	// theme change, including the ones the system triggers at sunset.
	a.trackOnce("theme:"+mode, "theme_changed", map[string]any{"to": themeMode(mode)})

	// macOS: pin NSApp.appearance directly — the Wails calls below are
	// Windows-only no-ops (this was the sidebar-follows-system bug).
	setNativeAppearance(mode)

	switch mode {
	case "dark":
		runtime.WindowSetDarkTheme(a.ctx)
	case "light":
		runtime.WindowSetLightTheme(a.ctx)
	default:
		runtime.WindowSetSystemDefaultTheme(a.ctx)
	}
}

// ImportConfig reads the conf file and merges its projects in.
func (a *App) ImportConfig() (string, error) {
	imported, err := config.Load()
	if err != nil {
		a.trackImport("path", 0, err)
		return "", err
	}
	msg, err := a.applyImport(imported)
	a.trackImport("path", len(imported), err)
	return msg, err
}

// ImportConfigText parses conf JSON pasted in the app, merges it, and
// saves a copy to the import path.
func (a *App) ImportConfigText(text string) (string, error) {
	data := []byte(text)
	imported, err := config.Parse(data)
	if err != nil {
		a.trackImport("paste", 0, err)
		return "", err
	}
	msg, err := a.applyImport(imported)
	a.trackImport("paste", len(imported), err)
	if err != nil {
		return "", err
	}
	_ = config.WriteImportFile(data) // keep a copy with the app
	return msg, nil
}

// PickConfigFile opens a native file picker for a JSON conf file.
func (a *App) PickConfigFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select configuration file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Config files (*.json, *.conf)", Pattern: "*.json;*.conf"},
		},
	})
}

// ImportConfigFile reads a picked conf file, merges it, and saves a
// copy to the import path.
func (a *App) ImportConfigFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		a.trackImport("file", 0, err)
		return "", err
	}
	imported, err := config.Parse(data)
	if err != nil {
		a.trackImport("file", 0, err)
		return "", err
	}
	msg, err := a.applyImport(imported)
	a.trackImport("file", len(imported), err)
	if err != nil {
		return "", err
	}
	_ = config.WriteImportFile(data)
	return msg, nil
}

// ReadConfigFile returns the raw contents of a picked conf file so the
// frontend can load it into the editor.
func (a *App) ReadConfigFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// applyImport merges imported projects into the store and returns a
// summary of what changed.
func (a *App) applyImport(imported []model.Project) (string, error) {
	existing, err := a.store.Load()
	if err != nil {
		return "", err
	}
	merged, added, updated := config.Merge(existing, imported)
	if err := a.store.Save(merged); err != nil {
		return "", err
	}
	return fmt.Sprintf("Imported %d project(s): %d added, %d updated", len(imported), added, updated), nil
}

// GetDependencies lists a folder's dependencies and their install state.
func (a *App) GetDependencies(dir string) (*deps.Info, error) {
	return deps.Inspect(dir)
}

// InstallDeps runs the install command for a process's folder through
// the process manager. Returns the ID used for log/exit events.
func (a *App) InstallDeps(projectID, procID string) (string, error) {
	start := time.Now()
	p, err := a.findProcess(projectID, procID)
	if err != nil {
		a.trackDepsInstall("", start, err)
		return "", err
	}
	info, err := deps.Inspect(p.Dir)
	if err != nil {
		a.trackDepsInstall("", start, err)
		return "", err
	}
	if info.InstallCommand == "" {
		err = fmt.Errorf("no package manager detected in %s", p.Dir)
		a.trackDepsInstall(info.Manager, start, err)
		return "", err
	}
	depsID := procID + ":deps"
	err = a.manager.Start(model.Process{
		ID:      depsID,
		Name:    p.Name + " (install deps)",
		Dir:     p.Dir,
		Command: info.InstallCommand,
	})
	// This reports that the install *launched*, not that it succeeded: the
	// command runs through the process manager and finishes asynchronously.
	a.trackDepsInstall(info.Manager, start, err)
	return depsID, err
}

// DetectProcesses scans root and its subfolders for runnable
// subprocesses (languages, frameworks, env files).
func (a *App) DetectProcesses(root string) ([]detect.Detected, error) {
	start := time.Now()
	found, err := detect.Scan(root)
	a.trackDetection(root, found, start, err)
	return found, err
}

// ReadEnvFile parses a dotenv file (.env, .env.local, ...) into a map.
func (a *App) ReadEnvFile(path string) (map[string]string, error) {
	return detect.ParseEnvFile(path)
}

// PickDirectory opens a native folder picker and returns the path.
// defaultDir, when non-empty, is where the dialog opens (typically the
// current project's root), so browsing for a process directory or a new
// project root starts nearby instead of wherever the dialog last was.
func (a *App) PickDirectory(defaultDir string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select folder",
		DefaultDirectory: defaultDir,
	})
}

// AI (local Ollama) bindings live in ai_api.go.
// Code-context indexing lives in codectx_api.go.
// Chat session persistence lives in chat_api.go.

// --- Docker / Containers ---

// DockerInfo reports what Docker assets (compose file, Dockerfile) a project
// root contains and whether the docker CLI is available on this machine.
func (a *App) DockerInfo(projectRoot string) docker.Info {
	return docker.Inspect(projectRoot)
}

// ComposeUp starts the project's compose stack in detached mode.
func (a *App) ComposeUp(projectRoot string) error {
	return a.dockerOp("compose_up", func() error { return docker.ComposeUp(projectRoot) })
}

// ComposeDown stops and removes the project's compose stack.
func (a *App) ComposeDown(projectRoot string) error {
	return a.dockerOp("compose_down", func() error { return docker.ComposeDown(projectRoot) })
}

// ListContainers returns the compose project's containers (running and stopped).
func (a *App) ListContainers(projectRoot string) ([]docker.Container, error) {
	return docker.Containers(projectRoot)
}

// ListImages returns the images backing the compose project's services.
func (a *App) ListImages(projectRoot string) ([]docker.Image, error) {
	return docker.Images(projectRoot)
}

// ListVolumes returns the Docker volumes owned by the compose project.
func (a *App) ListVolumes(projectRoot string) ([]docker.Volume, error) {
	return docker.Volumes(projectRoot)
}

// StartContainer starts a single container by ID or name.
func (a *App) StartContainer(id string) error {
	return a.dockerOp("container_start", func() error { return docker.StartContainer(id) })
}

// StopContainer stops a single container by ID or name.
func (a *App) StopContainer(id string) error {
	return a.dockerOp("container_stop", func() error { return docker.StopContainer(id) })
}

// RemoveContainer force-removes a single container by ID or name.
func (a *App) RemoveContainer(id string) error {
	return a.dockerOp("container_remove", func() error { return docker.RemoveContainer(id) })
}

// --- Git integration ---

// GitStatus reports the git state of a project's root directory.
func (a *App) GitStatus(projectRoot string) (*gitops.Status, error) {
	return gitops.GetStatus(projectRoot)
}

// GitInit initializes a new git repository at projectRoot.
func (a *App) GitInit(projectRoot string) error {
	return a.gitOp("init", func() error { return gitops.Init(projectRoot) })
}

// GitFetch fetches from origin using the stored GitHub token (falls back
// to GitLab token if no GitHub token is set).
func (a *App) GitFetch(projectRoot string) error {
	return a.gitOp("fetch", func() error {
		token, err := a.gitTokenFor(projectRoot)
		if err != nil {
			return err
		}
		return gitops.Fetch(projectRoot, token)
	})
}

// GitPull pulls the current branch from origin.
func (a *App) GitPull(projectRoot string) error {
	return a.gitOp("pull", func() error {
		token, err := a.gitTokenFor(projectRoot)
		if err != nil {
			return err
		}
		return gitops.Pull(projectRoot, token)
	})
}

// GitPush pushes the current branch to origin.
func (a *App) GitPush(projectRoot string) error {
	return a.gitOp("push", func() error {
		token, err := a.gitTokenFor(projectRoot)
		if err != nil {
			return err
		}
		return gitops.Push(a.ctx, projectRoot, token)
	})
}

// GitCommit stages all changes and commits them, returning the short hash.
func (a *App) GitCommit(projectRoot, message string) (hash string, err error) {
	// message_length, never the message: commit subjects describe the
	// user's private work and are the classic analytics leak.
	defer a.trackGit("commit", time.Now(), &err, map[string]any{
		"message_length": len(strings.TrimSpace(message)),
	})

	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit message is required")
	}
	name, email := gitops.Author(projectRoot)
	if name == "" {
		name = "JumpStart"
	}
	if email == "" {
		email = "jumpstart@local"
	}
	return gitops.Commit(projectRoot, message, name, email)
}

// GitAddRemote adds (or replaces) the "origin" remote for a project.
func (a *App) GitAddRemote(projectRoot, url string) error {
	return a.gitOp("remote_add", func() error {
		return gitops.AddRemote(projectRoot, "origin", url)
	})
}

// GitListBranches lists local and remote-tracking branches.
func (a *App) GitListBranches(projectRoot string) ([]gitops.Branch, error) {
	return gitops.ListBranches(projectRoot)
}

// GitGraphLog returns the commit graph (across all branches) for the
// timeline view; limit caps how many commits are returned.
func (a *App) GitGraphLog(projectRoot string, limit int) ([]gitops.GraphCommit, error) {
	return gitops.GraphLog(projectRoot, limit)
}

// GitCheckout switches the working tree to an existing branch.
func (a *App) GitCheckout(projectRoot, branch string) error {
	return a.gitOp("checkout", func() error { return gitops.Checkout(projectRoot, branch) })
}

// GitCreateBranch creates a new branch; when checkout is true it also
// switches to it.
func (a *App) GitCreateBranch(projectRoot, name string, checkout bool) error {
	return a.gitOp("branch_create", func() error {
		return gitops.CreateBranch(projectRoot, name, checkout)
	})
}

// GitDeleteBranch deletes a local branch (force uses -D).
func (a *App) GitDeleteBranch(projectRoot, name string, force bool) error {
	return a.gitOp("branch_delete", func() error {
		return gitops.DeleteBranch(projectRoot, name, force)
	})
}

// GitDiff returns a parsed diff for the given comparison mode
// (working, staging, worktree, remote, stash).
func (a *App) GitDiff(projectRoot, mode string) (res *gitops.DiffResult, err error) {
	defer a.trackGit("diff", time.Now(), &err, map[string]any{"mode": mode})
	return gitops.Diff(projectRoot, mode)
}

// GitListStashes returns the project's git stash entries.
func (a *App) GitListStashes(projectRoot string) ([]gitops.Stash, error) {
	return gitops.ListStashes(projectRoot)
}

// GitWorkingChanges lists every file with staged and/or unstaged changes,
// for the Git Changes modal's file lists.
func (a *App) GitWorkingChanges(projectRoot string) ([]gitops.FileChange, error) {
	return gitops.WorkingChanges(projectRoot)
}

// GitStageFile stages a single file.
func (a *App) GitStageFile(projectRoot, path string) error {
	return a.gitOp("stage_file", func() error { return gitops.StageFile(projectRoot, path) })
}

// GitUnstageFile removes a single file from the index.
func (a *App) GitUnstageFile(projectRoot, path string) error {
	return a.gitOp("unstage_file", func() error { return gitops.UnstageFile(projectRoot, path) })
}

// GitStageAll stages every pending change.
func (a *App) GitStageAll(projectRoot string) error {
	return a.gitOp("stage_all", func() error { return gitops.StageAll(projectRoot) })
}

// GitUnstageAll clears the index back to HEAD.
func (a *App) GitUnstageAll(projectRoot string) error {
	return a.gitOp("unstage_all", func() error { return gitops.UnstageAll(projectRoot) })
}

// GitRemoveIndexLock removes a stale .git/index.lock left behind by a
// crashed or force-quit git process. Every git operation fails with
// "Unable to create '.../index.lock': File exists" until that file is
// gone, so the Git Changes modal offers this as a one-click recovery
// instead of sending the user to a terminal. Refuses (see
// gitops.RemoveIndexLock) if the lock looks fresh enough to belong to a
// git process that's still actually running.
func (a *App) GitRemoveIndexLock(projectRoot string) error {
	return a.gitOp("remove_index_lock", func() error { return gitops.RemoveIndexLock(projectRoot) })
}

// GitFileDiff returns the unified diff for one file, staged or unstaged.
func (a *App) GitFileDiff(projectRoot, path string, staged bool) (string, error) {
	return gitops.FileDiff(projectRoot, path, staged)
}

// GitPublishBranch pushes a branch to origin for the first time and sets
// it as the branch's upstream, for a branch with no remote tracking yet.
func (a *App) GitPublishBranch(projectRoot, branch string) error {
	return a.gitOp("publish_branch", func() error {
		token, err := a.gitTokenFor(projectRoot)
		if err != nil {
			return err
		}
		return gitops.PublishBranch(projectRoot, branch, token)
	})
}

// SaveGitToken stores a personal access token for provider ("github" or
// "gitlab") in the OS keychain.
func (a *App) SaveGitToken(provider, token string) error {
	key, err := gitTokenKey(provider)
	if err != nil {
		return err
	}
	if key == secrets.KeyGitHubToken {
		// Keep the two GitHub keychain entries consistent: a pasted PAT
		// replaces any stored OAuth set, refresh token and all, rather
		// than leaving a stale set that would be preferred on read.
		err = saveGitHubTokenSet(github.StaticTokenSet(token))
	} else {
		err = secrets.SaveToken(secrets.Service, key, token)
	}
	a.track("git_token_saved", map[string]any{
		"provider":  analytics.Provider(provider),
		"succeeded": err == nil,
	})
	return err
}

// HasGitToken reports whether a token is stored for provider.
func (a *App) HasGitToken(provider string) (bool, error) {
	key, err := gitTokenKey(provider)
	if err != nil {
		return false, err
	}
	token, err := secrets.GetToken(secrets.Service, key)
	if err != nil {
		return false, err
	}
	return token != "", nil
}

// DeleteGitToken removes the stored token for provider.
func (a *App) DeleteGitToken(provider string) error {
	key, err := gitTokenKey(provider)
	if err != nil {
		return err
	}
	if key == secrets.KeyGitHubToken {
		return deleteGitHubTokenSet()
	}
	return secrets.DeleteToken(secrets.Service, key)
}

func gitTokenKey(provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github":
		return secrets.KeyGitHubToken, nil
	case "gitlab":
		return secrets.KeyGitLabToken, nil
	default:
		return "", fmt.Errorf("unknown git provider %q (expected \"github\" or \"gitlab\")", provider)
	}
}

// gitTokenFor picks the right stored token for a project's remote host.
// It falls back to trying both tokens if the host can't be determined,
// so Fetch/Pull/Push still work with whichever token is configured.
func (a *App) gitTokenFor(projectRoot string) (string, error) {
	remoteURL, err := gitops.RemoteURL(projectRoot)
	if err != nil {
		return "", err
	}
	host, _, _, perr := release.ParseRemote(remoteURL)
	if perr == nil {
		switch {
		case strings.Contains(host, "gitlab"):
			return secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
		case strings.Contains(host, "github"):
			return a.ghAccessToken(a.ctx, false)
		}
	}
	// Unknown host: prefer GitHub token, then GitLab token.
	if tok, _ := a.ghAccessToken(a.ctx, false); tok != "" {
		return tok, nil
	}
	return secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
}

// --- Release publishing ---

// CreateRelease publishes a release for projectRoot's "origin" remote on
// GitHub or GitLab (detected automatically) and returns the release URL.
func (a *App) CreateRelease(projectRoot string, opts release.ReleaseOptions) (url string, err error) {
	start := time.Now()
	provider := "unknown"
	defer func() {
		a.track("release_created", outcome(start, err, map[string]any{
			"provider":    provider,
			"prerelease":  opts.Prerelease,
			"draft":       opts.Draft,
			"has_notes":   strings.TrimSpace(opts.Body) != "",
			"notes_chars": len(opts.Body),
		}))
	}()

	remoteURL, err := gitops.RemoteURL(projectRoot)
	if err != nil {
		return "", err
	}
	host, _, _, err := release.ParseRemote(remoteURL)
	if err != nil {
		return "", err
	}
	provider = analytics.Provider(host)
	var token string
	switch {
	case strings.Contains(host, "gitlab"):
		token, err = secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
	case strings.Contains(host, "github"):
		token, err = a.ghAccessToken(a.ctx, false)
	default:
		return "", fmt.Errorf("unsupported git host: %s", host)
	}
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("no access token configured — add one in Preferences")
	}
	return release.CreateRelease(remoteURL, token, opts)
}

// --- Updates & remote banner ---

// GetAppVersion returns the running build's version string, without any
// leading "v" (the tag is injected as e.g. "v1.2.4").
func (a *App) GetAppVersion() string {
	return strings.TrimPrefix(Version, "v")
}

// CheckForUpdate queries GitHub Releases for a newer version of JumpStart.
// When beta is true the beta channel is included, so pre-release tags such as
// "v1.5.0-beta.2" become eligible; otherwise only stable releases are offered.
func (a *App) CheckForUpdate(beta bool) (update.Info, error) {
	info, err := update.Check(UpdateOwner, UpdateRepo, Version, beta)
	// Once per session: the frontend re-checks on a timer, and a polled
	// binding is the fastest way to blow through the event budget.
	a.trackOnce("update_checked", "update_checked", map[string]any{
		"channel":          update.ChannelName(beta),
		"update_available": info.Available,
		"latest_version":   info.LatestVersion,
		"succeeded":        err == nil,
		"failure_reason":   analytics.FailureReason(err),
	})
	return info, err
}

// InstallUpdate downloads the latest release for this platform and replaces
// the running app in place. Download progress is emitted to the frontend as
// "update:progress" (0-100), and "update:ready" fires on success. Call
// RestartApp afterwards to launch the new version. Concurrent calls return
// an error so the banner auto-download and a manual retry cannot race.
func (a *App) InstallUpdate(beta bool) (err error) {
	if !a.updating.CompareAndSwap(false, true) {
		return fmt.Errorf("update already in progress")
	}
	defer a.updating.Store(false)

	start := time.Now()
	// Version fragmentation is the metric to watch here: a long tail of old
	// app_version values weeks after a release means the updater is failing
	// silently, and every other number in GA4 is polluted by it.
	defer func() {
		a.track("update_installed", outcome(start, err, map[string]any{
			"channel":      update.ChannelName(beta),
			"from_version": strings.TrimPrefix(Version, "v"),
		}))
	}()

	asset, err := update.LatestAsset(UpdateOwner, UpdateRepo, beta)
	if err != nil {
		return err
	}
	if asset.URL == "" {
		return fmt.Errorf("no downloadable build found for this platform")
	}
	if err = update.Apply(asset, func(pct int) {
		runtime.EventsEmit(a.ctx, "update:progress", pct)
	}); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "update:ready")
	return nil
}

// RestartApp launches the updated binary and quits the current process.
func (a *App) RestartApp() error {
	if err := update.Relaunch(); err != nil {
		return err
	}
	runtime.Quit(a.ctx)
	return nil
}

// GetRemoteBanner fetches the remote overlay banner config from the
// built-in default location and returns the active announcements.
func (a *App) GetRemoteBanner() ([]banner.Banner, error) {
	return banner.Fetch()
}

// --- Project description (AI) ---

// GenerateProjectDescription asks the local model for a concise 1-3
// sentence description of the project, using lightweight signals gathered
// from the project directory (package.json, README, top-level files).
func (a *App) GenerateProjectDescription(host, aiModel, projectRoot, projectName string) (desc string, err error) {
	start := time.Now()
	defer func() {
		a.track("ai_description_generated", outcome(start, err, map[string]any{
			"model_family": analytics.AIModelFamily(aiModel),
			"param_size":   analytics.AIParamSize(aiModel),
			"reply_chars":  len(desc),
		}))
	}()

	projectContext := gatherProjectContext(projectRoot, projectName)
	system := "You are a helpful assistant that writes concise, plain-English descriptions of software projects. " +
		"Given some signals about a project (its name, package metadata, README excerpt, top-level files, and detected language/framework), " +
		"reply with ONLY a 1-3 sentence description of what the project is/does. No markdown, no quotes, no preamble."
	out, err := ai.New(host).Chat(a.ctx, aiModel, []ai.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: projectContext},
	}, false)
	if err != nil {
		return "", err
	}
	return cleanDescription(out), nil
}

// DetectTestConfig and RunTests are instrumented below; the detection call
// itself is cheap and is reported as part of tests_run rather than on its
// own, so opening the test panel does not emit an event per render.

// gatherProjectContext collects lightweight, non-sensitive signals about
// a project directory to ground the AI's description.
func gatherProjectContext(projectRoot, projectName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Project name: %s\n", projectName)

	if data, err := os.ReadFile(projectRoot + "/package.json"); err == nil {
		var pkg struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			if pkg.Name != "" {
				fmt.Fprintf(&b, "package.json name: %s\n", pkg.Name)
			}
			if pkg.Description != "" {
				fmt.Fprintf(&b, "package.json description: %s\n", pkg.Description)
			}
		}
	}

	for _, readme := range []string{"README.md", "Readme.md", "readme.md"} {
		if data, err := os.ReadFile(projectRoot + "/" + readme); err == nil {
			text := string(data)
			if len(text) > 500 {
				text = text[:500]
			}
			fmt.Fprintf(&b, "README excerpt:\n%s\n", text)
			break
		}
	}

	if entries, err := os.ReadDir(projectRoot); err == nil {
		var names []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			names = append(names, e.Name())
			if len(names) >= 25 {
				break
			}
		}
		if len(names) > 0 {
			fmt.Fprintf(&b, "Top-level entries: %s\n", strings.Join(names, ", "))
		}
	}

	if scanned, err := detect.Scan(projectRoot); err == nil && len(scanned) > 0 {
		d := scanned[0]
		if d.Language != "" {
			fmt.Fprintf(&b, "Detected language: %s\n", d.Language)
		}
		if d.Framework != "" {
			fmt.Fprintf(&b, "Detected framework: %s\n", d.Framework)
		}
	}

	return b.String()
}

// cleanDescription strips surrounding quotes/code fences the model
// sometimes adds despite instructions.
func cleanDescription(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	return strings.TrimSpace(s)
}

// --- Test runner ---

// DetectTestConfig inspects projectRoot and returns the detected test
// command, if any.
func (a *App) DetectTestConfig(projectRoot string) (*testrunner.TestConfig, error) {
	return testrunner.Detect(projectRoot)
}

// RunTests runs tests through the process manager as a one-off process
// (mirroring InstallDeps), returning the log ID the frontend can read via
// GetLogs / listen to via "log:<id>" and "exit:<id>" events.
//
// When procID is empty this runs the project's "global directory" test:
// it resolves against the project root, using the stored
// Project.TestCommand override. When procID names a subprocess, this runs
// that process's own tests instead: it resolves against the process's
// working directory, using the stored Process.TestCommand override.
//
// Resolution order for the command: customCommand param, then the
// relevant stored override, then auto-detection.
func (a *App) RunTests(projectID, procID, customCommand string) (id string, err error) {
	start := time.Now()
	detected := false
	kind := ""
	command := strings.TrimSpace(customCommand)
	defer func() {
		a.track("tests_run", outcome(start, err, map[string]any{
			"framework":   analytics.TestFramework(kind, command),
			"detected":    detected,
			"scope":       testScope(procID),
			"project_ref": a.ref(projectID),
		}))
	}()

	var dir, testKey string

	if procID == "" {
		projects, err := a.store.Load()
		if err != nil {
			return "", err
		}
		found := false
		for _, proj := range projects {
			if proj.ID == projectID {
				dir = proj.Root
				if command == "" {
					command = strings.TrimSpace(proj.TestCommand)
				}
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("project %s not found", projectID)
		}
		testKey = projectID
	} else {
		p, err := a.findProcess(projectID, procID)
		if err != nil {
			return "", err
		}
		dir = p.Dir
		if command == "" {
			command = strings.TrimSpace(p.TestCommand)
		}
		testKey = procID
	}

	if command == "" {
		cfg, derr := testrunner.Detect(dir)
		if derr != nil {
			return "", derr
		}
		if !cfg.Detected {
			return "", fmt.Errorf("no test command detected in %s — set one in project settings", dir)
		}
		detected = true
		kind = cfg.Kind
		command = cfg.Command
	}

	testID := testKey + ":test-" + fmt.Sprint(time.Now().UnixMilli())
	err = a.manager.Start(model.Process{
		ID:      testID,
		Name:    "Run tests",
		Dir:     dir,
		Command: command,
	})
	return testID, err
}

// testScope distinguishes a whole-project test run from one scoped to a
// single subprocess. Which one people reach for decides whether the
// per-process test button earns its place in the UI.
func testScope(procID string) string {
	if procID == "" {
		return "project"
	}
	return "process"
}

func (a *App) findProcess(projectID, procID string) (*model.Process, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	for _, proj := range projects {
		if proj.ID != projectID {
			continue
		}
		for i := range proj.Processes {
			if proj.Processes[i].ID == procID {
				return &proj.Processes[i], nil
			}
		}
	}
	return nil, fmt.Errorf("process not found")
}
