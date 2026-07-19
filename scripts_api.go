package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"devdeck/internal/model"
	"devdeck/internal/scripts"
)

// maxScriptRuns is how many past runs are kept per script. Runs live in
// memory only, so history resets when the app restarts.
const maxScriptRuns = 10

// ScriptRun is one execution of a Script. RunID doubles as the process
// manager key, so the frontend reads its output via GetLogs(runID) and
// listens on "log:<runID>" / "exit:<runID>".
type ScriptRun struct {
	RunID     string `json:"runId"`
	ScriptID  string `json:"scriptId"`
	ProcID    string `json:"procId"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	StartedAt int64  `json:"startedAt"` // unix ms
}

// scriptRuns keeps the recent run history, newest first, keyed by script ID.
type scriptRuns struct {
	mu   sync.Mutex
	runs map[string][]ScriptRun
}

func newScriptRuns() *scriptRuns {
	return &scriptRuns{runs: map[string][]ScriptRun{}}
}

func (s *scriptRuns) add(run ScriptRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := append([]ScriptRun{run}, s.runs[run.ScriptID]...)
	if len(list) > maxScriptRuns {
		list = list[:maxScriptRuns]
	}
	s.runs[run.ScriptID] = list
}

func (s *scriptRuns) forScript(scriptID string) []ScriptRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ScriptRun, len(s.runs[scriptID]))
	copy(out, s.runs[scriptID])
	return out
}

func (s *scriptRuns) forProcess(procID string) []ScriptRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ScriptRun
	for _, list := range s.runs {
		for _, r := range list {
			if r.ProcID == procID {
				out = append(out, r)
			}
		}
	}
	if out == nil {
		out = []ScriptRun{}
	}
	return out
}

// DetectScripts scans a directory for one-off commands it can offer as
// script buttons (npm scripts, Makefile targets, Go flags, and so on).
func (a *App) DetectScripts(dir string) ([]scripts.Found, error) {
	return scripts.Detect(dir)
}

// RunScript executes a process's script as a one-off through the process
// manager (mirroring RunTests) and returns the run ID the frontend uses to
// read logs. The script's dir and env fall back to the parent process's.
func (a *App) RunScript(projectID, procID, scriptID string) (string, error) {
	proc, err := a.findProcess(projectID, procID)
	if err != nil {
		return "", err
	}
	script, err := findScript(proc, scriptID)
	if err != nil {
		return "", err
	}
	command := strings.TrimSpace(script.Command)
	if command == "" {
		return "", fmt.Errorf("script %q has no command", script.Name)
	}

	dir := strings.TrimSpace(script.Dir)
	if dir == "" {
		dir = proc.Dir
	}

	env := map[string]string{}
	for k, v := range proc.Env {
		env[k] = v
	}
	for k, v := range script.Env {
		env[k] = v
	}

	startedAt := time.Now()
	runID := fmt.Sprintf("%s:script-%d", scriptID, startedAt.UnixMilli())
	if err := a.manager.Start(model.Process{
		ID:      runID,
		Name:    script.Name,
		Dir:     dir,
		Command: command,
		Env:     env,
	}); err != nil {
		return "", err
	}

	a.scriptRuns.add(ScriptRun{
		RunID:     runID,
		ScriptID:  scriptID,
		ProcID:    procID,
		Name:      script.Name,
		Command:   command,
		StartedAt: startedAt.UnixMilli(),
	})
	return runID, nil
}

// StopScriptRun terminates a still-running script execution.
func (a *App) StopScriptRun(runID string) error {
	return a.manager.Stop(runID)
}

// ListScriptRuns returns the recent runs of one script, newest first.
func (a *App) ListScriptRuns(scriptID string) []ScriptRun {
	return a.scriptRuns.forScript(scriptID)
}

// ListProcessScriptRuns returns every recent run across a process's scripts.
func (a *App) ListProcessScriptRuns(procID string) []ScriptRun {
	return a.scriptRuns.forProcess(procID)
}

// GetScriptRunStatus reports whether a run is still going and its exit code.
func (a *App) GetScriptRunStatus(runID string) model.Status {
	return a.manager.Status(runID)
}

func findScript(proc *model.Process, scriptID string) (*model.Script, error) {
	for i := range proc.Scripts {
		if proc.Scripts[i].ID == scriptID {
			return &proc.Scripts[i], nil
		}
	}
	return nil, fmt.Errorf("script not found")
}
