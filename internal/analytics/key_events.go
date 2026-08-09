package analytics

// IsKeyEvent reports whether this event should be treated as a GA4 key /
// conversion candidate. Operators still mark key events in GA4 Admin; this
// flag keeps DebugView and explorations consistent.
func IsKeyEvent(name string, props map[string]any) bool {
	switch name {
	case "project_created", "process_started", "processes_accepted", "update_installed":
		return true
	case "consent_decided":
		g, _ := props["granted"].(bool)
		return g
	}
	return false
}
