package main

import (
	"strings"
	"time"

	"devdeck/internal/analytics"
	"devdeck/internal/detect"
	"devdeck/internal/model"
)

// Feature-adoption events (taxonomy 4.3 and 4.4).
//
// Git and Docker each get ONE event with an `action` property rather than
// one event per verb. Fourteen separate git events would mean fourteen
// charts to maintain and no way to ask the question that actually matters:
// "does this user use git at all?"

// gitOp runs a git binding and reports it as git_action_performed.
func (a *App) gitOp(action string, fn func() error) error {
	start := time.Now()
	err := fn()
	a.track("git_action_performed", outcome(start, err, map[string]any{"action": action}))
	return err
}

// trackGit is the deferred form, for bindings that need extra properties or
// return a value alongside the error. errp is read after the function
// returns, so the reported outcome is the real one rather than the attempt.
func (a *App) trackGit(action string, start time.Time, errp *error, extra map[string]any) {
	var err error
	if errp != nil {
		err = *errp
	}
	props := outcome(start, err, extra)
	props["action"] = action
	a.track("git_action_performed", props)
}

// dockerOp runs a docker binding and reports it as docker_action_performed.
func (a *App) dockerOp(action string, fn func() error) error {
	start := time.Now()
	err := fn()
	a.track("docker_action_performed", outcome(start, err, map[string]any{"action": action}))
	return err
}

// --- Processes (taxonomy 4.3) ---

// trackProcessStarted reports one process launch. startup_ms measures how
// long the manager took to hand back control, not how long the dev server
// took to boot: the latter is not observable from here.
func (a *App) trackProcessStarted(projectID string, p model.Process, trigger string, start time.Time, err error) {
	if err == nil {
		a.procStarts.Store(p.ID, time.Now())
		// A fresh run clears any stop flag left over from the last one.
		a.stopping.Delete(p.ID)
	}
	a.track("process_started", outcome(start, err, map[string]any{
		"runtime":     analytics.Runtime("", p.Command),
		"trigger":     trigger,
		"project_ref": a.ref(projectID),
		"has_env":     len(p.Env) > 0,
		"path_depth":  analytics.PathDepth(p.Dir),
	}))
}

// trackProcessStopped reports one process stop, with how long it had been
// running. uptime_s is the number that separates "started it and it works"
// from "started it, it crashed in two seconds, tried again".
func (a *App) trackProcessStopped(procID, trigger string, err error) {
	uptime := int64(0)
	if v, ok := a.procStarts.LoadAndDelete(procID); ok {
		if started, ok := v.(time.Time); ok {
			uptime = int64(time.Since(started).Seconds())
		}
	}
	a.track("process_stopped", map[string]any{
		"trigger":        trigger,
		"uptime_s":       uptime,
		"succeeded":      err == nil,
		"failure_reason": analytics.FailureReason(err),
	})
}

// noteIntentionalStop marks a process as being stopped on purpose, before
// the stop is issued.
//
// Terminating a process makes it exit non-zero, which is indistinguishable
// from a crash by the time the exit event arrives — and the exit event can
// arrive before Stop even returns, so ordering alone cannot separate them.
// Without this flag, every "Stop" click would show up as a crash and the
// stability dashboard would be pure noise.
func (a *App) noteIntentionalStop(procID string) {
	a.stopping.Store(procID, true)
}

// trackProcessEvent listens on the process manager's event stream for
// unexpected exits. A process that dies on its own never returns through a
// Wails binding, so this is the only place a crash is observable.
func (a *App) trackProcessEvent(event string, data ...interface{}) {
	procID, ok := strings.CutPrefix(event, "exit:")
	if !ok || len(data) == 0 {
		return
	}
	exitCode, ok := data[0].(int)
	if !ok || exitCode == 0 {
		return
	}
	// Quitting the app stops everything; none of that is a crash.
	if a.shuttingDown.Load() {
		return
	}
	// Checked before procStarts is touched, so an intentional stop leaves
	// the start time in place for process_stopped's uptime_s.
	if _, intentional := a.stopping.Load(procID); intentional {
		return
	}
	started, ok := a.procStarts.LoadAndDelete(procID)
	if !ok {
		return
	}
	uptime := int64(0)
	if at, ok := started.(time.Time); ok {
		uptime = int64(time.Since(at).Seconds())
	}
	a.track("process_crashed", map[string]any{
		"exit_code": exitCode,
		"uptime_s":  uptime,
		// A crash in the first few seconds is a configuration problem; one
		// after an hour is a different bug entirely.
		"died_immediately": uptime < 5,
	})
}

// --- Onboarding funnel (taxonomy 4.2) ---

// trackProjectSaved turns one SaveProject call into the funnel events it
// actually represents: a project_created the first time an ID is seen, and
// a process_added for every subprocess that was not there before.
//
// Diffing here rather than instrumenting the UI means the funnel cannot
// drift when a new code path starts saving projects.
func (a *App) trackProjectSaved(p, before model.Project, existed bool, err error) {
	if err != nil {
		return
	}

	if !existed {
		a.track("project_created", map[string]any{
			"project_ref":     a.ref(p.ID),
			"source":          a.projectSource(p.Root),
			"process_count":   len(p.Processes),
			"path_depth":      analytics.PathDepth(p.Root),
			"has_description": p.Description != "",
			"tasks_enabled":   p.TasksEnabled,
		})
	}

	known := make(map[string]bool, len(before.Processes))
	for _, proc := range before.Processes {
		known[proc.ID] = true
	}
	for _, proc := range p.Processes {
		if known[proc.ID] {
			continue
		}
		a.track("process_added", map[string]any{
			"project_ref":  a.ref(p.ID),
			"runtime":      analytics.Runtime("", proc.Command),
			"detected":     a.wasDetected(proc.Dir),
			"has_env":      len(proc.Env) > 0,
			"script_count": len(proc.Scripts),
			"path_depth":   analytics.PathDepth(proc.Dir),
		})
	}
}

// trackDetection reports one auto-detect scan.
//
// processes_detected with accepted_count 0 is the highest-signal failure
// event in the app: detection ran, found something, and the user rejected
// all of it. accepted_count is filled in later, when the resulting
// process_added events land, so the two are joined in PostHog rather than
// held open here.
func (a *App) trackDetection(root string, found []detect.Detected, start time.Time, err error) {
	runtimes := make([]string, 0, len(found))
	seen := map[string]bool{}
	for _, d := range found {
		r := analytics.Runtime(d.Language, d.Command)
		if !seen[r] {
			seen[r] = true
			runtimes = append(runtimes, r)
		}
	}

	if err == nil && len(found) > 0 {
		a.noteDetection(root)
	}
	a.track("processes_detected", outcome(start, err, map[string]any{
		"count":      len(found),
		"runtimes":   runtimes,
		"path_depth": analytics.PathDepth(root),
	}))
}

// trackImport reports a conf-file import. source separates the three entry
// points, because a failure rate that is concentrated in one of them is a
// UI problem rather than a parser problem.
func (a *App) trackImport(source string, projectCount int, err error) {
	a.track("config_imported", map[string]any{
		"source":         source,
		"project_count":  projectCount,
		"succeeded":      err == nil,
		"failure_reason": analytics.FailureReason(err),
	})
}

// trackDepsInstall reports a dependency install launching.
func (a *App) trackDepsInstall(manager string, start time.Time, err error) {
	a.track("deps_installed", outcome(start, err, map[string]any{
		"manager": analytics.PackageManager(manager),
	}))
}

// --- Detection hint ---
//
// DetectProcesses and SaveProject are two separate round trips, so the Go
// side cannot otherwise tell a hand-typed project from one the wizard
// auto-detected. Remembering the last scanned root for a few minutes closes
// that gap without adding a parameter to a binding the frontend already
// calls from several places.

const detectionWindow = 15 * time.Minute

type detection struct {
	root string
	at   time.Time
}

func (a *App) noteDetection(root string) {
	a.lastDetect.Store(detection{root: root, at: time.Now()})
}

func (a *App) wasDetected(dir string) bool {
	v := a.lastDetect.Load()
	d, ok := v.(detection)
	if !ok || d.root == "" {
		return false
	}
	if time.Since(d.at) > detectionWindow {
		return false
	}
	// Detected subprocesses live at or under the scanned root.
	return dir == d.root || strings.HasPrefix(dir, d.root)
}

// projectSource classifies how a project came to exist. Imports report
// themselves through config_imported, so the only distinction left here is
// whether auto-detection was involved.
func (a *App) projectSource(root string) string {
	if a.wasDetected(root) {
		return "detect"
	}
	return "manual"
}

// themeMode bounds the theme value; anything unexpected becomes "system"
// rather than a new property value nobody will recognise in a breakdown.
func themeMode(mode string) string {
	switch mode {
	case "light", "dark":
		return mode
	}
	return "system"
}
