package main

import (
	"context"
	"fmt"
	"time"

	"encoding/json"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/ai"
	"devdeck/internal/config"
	"devdeck/internal/deps"
	"devdeck/internal/detect"
	"devdeck/internal/gitops"
	"devdeck/internal/model"
	"devdeck/internal/procman"
	"devdeck/internal/release"
	"devdeck/internal/secrets"
	"devdeck/internal/store"
	"devdeck/internal/sysinfo"
	"devdeck/internal/testrunner"
)

// App is the Wails-bound API used by the frontend.
type App struct {
	ctx     context.Context
	store   *store.Store
	manager *procman.Manager
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.manager = procman.New(func(event string, data ...interface{}) {
		runtime.EventsEmit(a.ctx, event, data...)
	})
	s, err := store.New()
	if err != nil {
		panic(err)
	}
	a.store = s
}

func (a *App) Shutdown(ctx context.Context) {
	a.manager.StopAll()
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
	for i := range projects {
		if projects[i].ID == p.ID {
			projects[i] = p
			found = true
			break
		}
	}
	if !found {
		projects = append(projects, p)
	}
	return a.store.Save(projects)
}

func (a *App) DeleteProject(id string) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	out := projects[:0]
	for _, p := range projects {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return a.store.Save(out)
}

// --- Processes ---

func (a *App) StartProcess(projectID, procID string) error {
	p, err := a.findProcess(projectID, procID)
	if err != nil {
		return err
	}
	if err := a.manager.Start(*p); err != nil {
		return err
	}
	a.touchUsage(projectID)
	return nil
}

func (a *App) StopProcess(procID string) error {
	return a.manager.Stop(procID)
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
			if err := a.manager.Start(proc); err != nil {
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
			_ = a.manager.Stop(proc.ID)
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
			projects[i].Tasks = tasks
			return a.store.Save(projects)
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
		return "", err
	}
	return a.applyImport(imported)
}

// ImportConfigText parses conf JSON pasted in the app, merges it, and
// saves a copy to the import path.
func (a *App) ImportConfigText(text string) (string, error) {
	data := []byte(text)
	imported, err := config.Parse(data)
	if err != nil {
		return "", err
	}
	msg, err := a.applyImport(imported)
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
		return "", err
	}
	imported, err := config.Parse(data)
	if err != nil {
		return "", err
	}
	msg, err := a.applyImport(imported)
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
	p, err := a.findProcess(projectID, procID)
	if err != nil {
		return "", err
	}
	info, err := deps.Inspect(p.Dir)
	if err != nil {
		return "", err
	}
	if info.InstallCommand == "" {
		return "", fmt.Errorf("no package manager detected in %s", p.Dir)
	}
	depsID := procID + ":deps"
	err = a.manager.Start(model.Process{
		ID:      depsID,
		Name:    p.Name + " (install deps)",
		Dir:     p.Dir,
		Command: info.InstallCommand,
	})
	return depsID, err
}

// DetectProcesses scans root and its subfolders for runnable
// subprocesses (languages, frameworks, env files).
func (a *App) DetectProcesses(root string) ([]detect.Detected, error) {
	return detect.Scan(root)
}

// ReadEnvFile parses a dotenv file (.env, .env.local, ...) into a map.
func (a *App) ReadEnvFile(path string) (map[string]string, error) {
	return detect.ParseEnvFile(path)
}

// PickDirectory opens a native folder picker and returns the path.
func (a *App) PickDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select folder",
	})
}

// --- AI (local Ollama) ---

// EnrichResult is the AI-suggested fill for one task/story.
type EnrichResult struct {
	Description string   `json:"description"`
	Acceptance  []string `json:"acceptance"`
	Subtasks    []string `json:"subtasks"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
}

// GeneratedTask is a child task suggested for a generated story.
type GeneratedTask struct {
	Title string `json:"title"`
}

// GeneratedStory is one user story proposed by the chat assistant.
type GeneratedStory struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Acceptance  []string        `json:"acceptance"`
	Priority    string          `json:"priority"`
	Labels      []string        `json:"labels"`
	StoryPoints int             `json:"storyPoints"`
	Tasks       []GeneratedTask `json:"tasks"`
}

// ChatResult bundles the assistant's prose reply with any stories it
// generated in the same turn.
type ChatResult struct {
	Reply   string           `json:"reply"`
	Stories []GeneratedStory `json:"stories"`
}

// OllamaListModels returns the models installed on the given Ollama host
// (empty host uses the default localhost:11434).
func (a *App) OllamaListModels(host string) ([]string, error) {
	return ai.New(host).ListModels(a.ctx)
}

// OllamaEnrichTask asks the model to flesh out a single item: a
// description, acceptance criteria, subtasks, a priority, and labels.
func (a *App) OllamaEnrichTask(host, model, title, kind, projectContext string) (EnrichResult, error) {
	var res EnrichResult
	if strings.TrimSpace(title) == "" {
		return res, fmt.Errorf("title is required")
	}
	if kind == "" {
		kind = "task"
	}
	system := "You are a product management assistant. Given a work item, return ONLY a JSON object " +
		"with keys: description (string, one short paragraph; for a story write it as \"As a <role>, I want <goal>, so that <benefit>\"), " +
		"acceptance (array of short acceptance-criteria strings), subtasks (array of short actionable subtask strings), " +
		"priority (one of \"low\", \"medium\", \"high\"), labels (array of 1-3 short lowercase tags). Do not include any prose outside the JSON."
	user := fmt.Sprintf("Item type: %s\nTitle: %s", kind, title)
	if strings.TrimSpace(projectContext) != "" {
		user += "\nProject context: " + projectContext
	}
	out, err := ai.New(host).Chat(a.ctx, model, []ai.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, true)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal([]byte(extractJSON(out)), &res); err != nil {
		return res, fmt.Errorf("could not parse model output: %w", err)
	}
	return res, nil
}

// OllamaChat drives the story-generating chat. It receives the full
// conversation and returns a prose reply plus any stories the user asked
// for (each with child tasks). history roles are "user"/"assistant".
func (a *App) OllamaChat(host, model string, history []ai.ChatMessage, projectContext string) (ChatResult, error) {
	var res ChatResult
	system := "You are a product manager helping plan software work. Reply conversationally, and whenever the user asks you to " +
		"create, draft, or break down work, produce user stories. Return ONLY a JSON object with keys: " +
		"reply (string, your conversational message to the user), and stories (array). Each story object has: " +
		"title (string), description (string, written as \"As a <role>, I want <goal>, so that <benefit>\"), " +
		"acceptance (array of criteria strings), priority (\"low\"|\"medium\"|\"high\"), labels (array of short tags), " +
		"storyPoints (integer 1-13), and tasks (array of objects each with a title string). " +
		"When the user is only chatting and not asking for stories, return an empty stories array. Never put text outside the JSON."
	if strings.TrimSpace(projectContext) != "" {
		system += "\nProject context: " + projectContext
	}
	msgs := append([]ai.ChatMessage{{Role: "system", Content: system}}, history...)
	out, err := ai.New(host).Chat(a.ctx, model, msgs, true)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal([]byte(extractJSON(out)), &res); err != nil {
		// Fall back to treating the whole output as a plain reply.
		return ChatResult{Reply: out}, nil
	}
	return res, nil
}

// extractJSON pulls the first {...} block out of a model response, in
// case the model wrapped it in code fences or stray prose.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// --- Git integration ---

// GitStatus reports the git state of a project's root directory.
func (a *App) GitStatus(projectRoot string) (*gitops.Status, error) {
	return gitops.GetStatus(projectRoot)
}

// GitInit initializes a new git repository at projectRoot.
func (a *App) GitInit(projectRoot string) error {
	return gitops.Init(projectRoot)
}

// GitFetch fetches from origin using the stored GitHub token (falls back
// to GitLab token if no GitHub token is set).
func (a *App) GitFetch(projectRoot string) error {
	token, err := a.gitTokenFor(projectRoot)
	if err != nil {
		return err
	}
	return gitops.Fetch(projectRoot, token)
}

// GitPull pulls the current branch from origin.
func (a *App) GitPull(projectRoot string) error {
	token, err := a.gitTokenFor(projectRoot)
	if err != nil {
		return err
	}
	return gitops.Pull(projectRoot, token)
}

// GitPush pushes the current branch to origin.
func (a *App) GitPush(projectRoot string) error {
	token, err := a.gitTokenFor(projectRoot)
	if err != nil {
		return err
	}
	return gitops.Push(projectRoot, token)
}

// GitCommit stages all changes and commits them, returning the short hash.
func (a *App) GitCommit(projectRoot, message string) (string, error) {
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
	return gitops.AddRemote(projectRoot, "origin", url)
}

// SaveGitToken stores a personal access token for provider ("github" or
// "gitlab") in the OS keychain.
func (a *App) SaveGitToken(provider, token string) error {
	key, err := gitTokenKey(provider)
	if err != nil {
		return err
	}
	return secrets.SaveToken(secrets.Service, key, token)
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
			return secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
		}
	}
	// Unknown host: prefer GitHub token, then GitLab token.
	if tok, _ := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken); tok != "" {
		return tok, nil
	}
	return secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
}

// --- Release publishing ---

// CreateRelease publishes a release for projectRoot's "origin" remote on
// GitHub or GitLab (detected automatically) and returns the release URL.
func (a *App) CreateRelease(projectRoot string, opts release.ReleaseOptions) (string, error) {
	remoteURL, err := gitops.RemoteURL(projectRoot)
	if err != nil {
		return "", err
	}
	host, _, _, err := release.ParseRemote(remoteURL)
	if err != nil {
		return "", err
	}
	var token string
	switch {
	case strings.Contains(host, "gitlab"):
		token, err = secrets.GetToken(secrets.Service, secrets.KeyGitLabToken)
	case strings.Contains(host, "github"):
		token, err = secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
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

// --- Project description (AI) ---

// GenerateProjectDescription asks the local model for a concise 1-3
// sentence description of the project, using lightweight signals gathered
// from the project directory (package.json, README, top-level files).
func (a *App) GenerateProjectDescription(host, aiModel, projectRoot, projectName string) (string, error) {
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

// RunTests runs the project's tests through the process manager as a
// one-off process (mirroring InstallDeps), returning the log ID the
// frontend can read via GetLogs / listen to via "log:<id>" and
// "exit:<id>" events. Resolution order: customCommand param, then the
// project's stored TestCommand override, then auto-detection.
func (a *App) RunTests(projectID, projectRoot, customCommand string) (string, error) {
	command := strings.TrimSpace(customCommand)
	if command == "" {
		projects, err := a.store.Load()
		if err != nil {
			return "", err
		}
		for _, proj := range projects {
			if proj.ID == projectID {
				command = strings.TrimSpace(proj.TestCommand)
				break
			}
		}
	}
	if command == "" {
		cfg, err := testrunner.Detect(projectRoot)
		if err != nil {
			return "", err
		}
		if !cfg.Detected {
			return "", fmt.Errorf("no test command detected in %s — set one in project settings", projectRoot)
		}
		command = cfg.Command
	}

	testID := projectID + ":test-" + fmt.Sprint(time.Now().UnixMilli())
	err := a.manager.Start(model.Process{
		ID:      testID,
		Name:    "Run tests",
		Dir:     projectRoot,
		Command: command,
	})
	return testID, err
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
