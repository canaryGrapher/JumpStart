package analytics

import "testing"

func TestIsKeyEvent(t *testing.T) {
	cases := []struct {
		name  string
		props map[string]any
		want  bool
	}{
		{"project_created", nil, true},
		{"process_started", nil, true},
		{"processes_accepted", nil, true},
		{"update_installed", nil, true},
		{"consent_decided", map[string]any{"granted": true}, true},
		{"consent_decided", map[string]any{"granted": false}, false},
		{"panel_opened", nil, false},
		{"app_launched", nil, false},
	}
	for _, tc := range cases {
		if got := IsKeyEvent(tc.name, tc.props); got != tc.want {
			t.Errorf("IsKeyEvent(%q)=%v want %v", tc.name, got, tc.want)
		}
	}
}
