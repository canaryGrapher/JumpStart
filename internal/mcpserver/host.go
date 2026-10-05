package mcpserver

import (
	"devdeck/internal/gitops"
	"devdeck/internal/model"
)

// Host is the JumpStart surface the MCP tools call into. App implements it.
type Host interface {
	ListProjects() ([]model.Project, error)
	GetProject(id string) (*model.Project, error)
	SaveProject(p model.Project) error

	StartProcess(projectID, procID string) error
	StopProcess(procID string) error
	StartAll(projectID string) []string
	StopAll(projectID string)
	ProcessStatus(procID string) model.Status
	ProcessLogs(procID string) []string

	UpdateTasks(projectID string, tasks []model.Task) error

	GitStatus(projectRoot string) (*gitops.Status, error)
	GitDiff(projectRoot, mode string) (*gitops.DiffResult, error)
	GitWorkingChanges(projectRoot string) ([]gitops.FileChange, error)
}
