package main

import (
	"fmt"

	"devdeck/internal/analytics"
	"devdeck/internal/daterange"
	"devdeck/internal/gitops"
	"devdeck/internal/mcpserver"
	"devdeck/internal/model"
)

// MCPSettings is what Preferences → Agents renders.
type MCPSettings struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Token   string `json:"token"`
	URL     string `json:"url"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

// mcpHost adapts App to mcpserver.Host without exporting extra Wails bindings.
type mcpHost struct{ app *App }

func (h mcpHost) ListProjects() ([]model.Project, error) {
	return h.app.store.Load()
}

func (h mcpHost) GetProject(id string) (*model.Project, error) {
	projects, err := h.app.store.Load()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ID == id {
			p := projects[i]
			return &p, nil
		}
	}
	return nil, fmt.Errorf("project %s not found", id)
}

func (h mcpHost) SaveProject(p model.Project) error {
	return h.app.SaveProject(p)
}

func (h mcpHost) StartProcess(projectID, procID string) error {
	return h.app.StartProcess(projectID, procID)
}

func (h mcpHost) StopProcess(procID string) error {
	return h.app.StopProcess(procID)
}

func (h mcpHost) StartAll(projectID string) []string {
	return h.app.StartAll(projectID)
}

func (h mcpHost) StopAll(projectID string) {
	h.app.StopAll(projectID)
}

func (h mcpHost) ProcessStatus(procID string) model.Status {
	return h.app.manager.Status(procID)
}

func (h mcpHost) ProcessLogs(procID string) []string {
	return h.app.manager.Logs(procID)
}

func (h mcpHost) UpdateTasks(projectID string, tasks []model.Task) error {
	return h.app.UpdateTasks(projectID, tasks)
}

func (h mcpHost) GlobalQuarters() []model.QuarterRange {
	return daterange.LoadGlobal(analytics.DataDir())
}

func (h mcpHost) GitStatus(projectRoot string) (*gitops.Status, error) {
	return h.app.GitStatus(projectRoot)
}

func (h mcpHost) GitDiff(projectRoot, mode string) (*gitops.DiffResult, error) {
	return h.app.GitDiff(projectRoot, mode)
}

func (h mcpHost) GitWorkingChanges(projectRoot string) ([]gitops.FileChange, error) {
	return h.app.GitWorkingChanges(projectRoot)
}

var _ mcpserver.Host = mcpHost{}

func (a *App) initMCP() {
	a.mcp = mcpserver.New(mcpHost{app: a}, analytics.DataDir(), Version)
	if err := a.mcp.SyncFromDisk(); err != nil {
		// Non-fatal: Preferences still shows the error string.
		_ = err
	}
}

func (a *App) stopMCP() {
	if a.mcp != nil {
		_ = a.mcp.Stop()
	}
}

// GetMCPSettings returns the MCP preference + live status for Preferences.
func (a *App) GetMCPSettings() MCPSettings {
	if a.mcp == nil {
		cfg := mcpserver.LoadConfig(analytics.DataDir())
		return MCPSettings{
			Enabled: cfg.Enabled,
			Port:    cfg.Port,
			Token:   cfg.Token,
			URL:     fmt.Sprintf("http://127.0.0.1:%d/mcp", cfg.Port),
		}
	}
	st := a.mcp.Status()
	return MCPSettings{
		Enabled: st.Enabled,
		Port:    st.Port,
		Token:   st.Token,
		URL:     st.URL,
		Running: st.Running,
		Error:   st.Error,
	}
}

// SetMCPSettings saves MCP preferences and starts or stops the listener.
func (a *App) SetMCPSettings(enabled bool, port int) (MCPSettings, error) {
	if a.mcp == nil {
		a.initMCP()
	}
	cfg := a.mcp.Config()
	cfg.Enabled = enabled
	if port > 0 {
		cfg.Port = port
	}
	if err := a.mcp.Apply(cfg); err != nil {
		return a.GetMCPSettings(), err
	}
	a.track("mcp_settings_changed", map[string]any{
		"enabled": enabled,
		"port":    cfg.Port,
	})
	return a.GetMCPSettings(), nil
}

// RotateMCPToken issues a new bearer token and restarts the listener if enabled.
func (a *App) RotateMCPToken() (MCPSettings, error) {
	if a.mcp == nil {
		a.initMCP()
	}
	cfg, err := mcpserver.RotateToken(analytics.DataDir())
	if err != nil {
		return a.GetMCPSettings(), err
	}
	if err := a.mcp.Apply(cfg); err != nil {
		return a.GetMCPSettings(), err
	}
	a.track("mcp_token_rotated", map[string]any{"succeeded": true})
	return a.GetMCPSettings(), nil
}
