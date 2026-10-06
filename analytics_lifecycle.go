package main

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"
)

// Lifecycle events (taxonomy 4.1). These are what make a user count as
// active at all: DAU/WAU/MAU, session length, and stability by version.

// trackLaunch reports one app start. project_count and process_count answer
// "how big does a real install get?", which is the number that decides
// whether the sidebar and dashboard designs still hold up.
func (a *App) trackLaunch() {
	var projects, favorites, processes, withTasks int
	if list, err := a.store.Load(); err == nil {
		projects = len(list)
		for _, p := range list {
			if p.Favorite {
				favorites++
			}
			if p.TasksEnabled {
				withTasks++
			}
			processes += len(p.Processes)
		}
	}

	// session_start helps GA4 attribute MP events to a session in Realtime.
	a.trackOnce("analytics:session_start", "session_start", nil)
	a.track("app_launched", map[string]any{
		"cold_start_ms":        time.Since(processStart).Milliseconds(),
		"project_count":        projects,
		"favorite_count":       favorites,
		"process_count":        processes,
		"kanban_project_count": withTasks,
	})
}

// trackClose reports a clean shutdown. Sessions that end in a crash or a
// force-quit never reach this, which is exactly why app_launched and not
// app_closed is the denominator for every rate.
func (a *App) trackClose() {
	a.track("app_closed", map[string]any{
		"session_duration_s":  int64(time.Since(processStart).Seconds()),
		"events_this_session": a.analytics.EventCount(),
	})
}

// trackPanic reports a crash. panic_type is the value's Go type rather than
// its message: a panic message routinely embeds a path or a project name,
// while the type is bounded and still groups crashes usefully.
//
// crash_site is narrowed to the package the panic came from, for the same
// reason: a full frame would carry the module path and arguments.
func (a *App) trackPanic(recovered any) {
	a.track("app_crashed", map[string]any{
		"panic_type":         fmt.Sprintf("%T", recovered),
		"crash_package":      crashPackage(),
		"session_duration_s": int64(time.Since(processStart).Seconds()),
	})
	// A crash gets an explicit short flush. The process is about to die, so
	// this event has one chance to leave, and the offline queue on the next
	// launch is the only fallback.
	a.analytics.Close(2 * time.Second)
}

// crashPackage returns the short package name of the topmost stack frame
// inside this module, e.g. "procman" for
// "devdeck/internal/procman.(*Manager).Start". Frames from the runtime and
// from dependencies are skipped so the value points at our own code.
func crashPackage() string {
	for _, line := range strings.Split(string(debug.Stack()), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "devdeck/") {
			continue
		}
		// "devdeck/internal/procman.(*Manager).Start(0x14000...)"
		//                    ^^^^^^^ is all we keep.
		trimmed := strings.TrimPrefix(line, "devdeck/")
		if slash := strings.LastIndex(trimmed, "/"); slash >= 0 {
			trimmed = trimmed[slash+1:]
		}
		if dot := strings.Index(trimmed, "."); dot > 0 {
			trimmed = trimmed[:dot]
		}
		if trimmed != "" {
			return trimmed
		}
	}
	return "unknown"
}
