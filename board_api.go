package main

import (
	"fmt"

	"devdeck/internal/ghsync"
	"devdeck/internal/model"
	"devdeck/internal/taskmerge"
)

// BoardLayoutResult is the saved board after SaveBoardLayout.
type BoardLayoutResult struct {
	Columns []model.BoardColumn `json:"columns"`
	Tasks   []model.Task        `json:"tasks"`
	Moved   int                 `json:"moved"`
}

// SaveBoardLayout applies an edited Kanban layout in one write: column
// order, labels, additions, and deletions (moving each deleted column's
// tasks to the column named in moves). See ghsync.ApplyBoardLayout.
func (a *App) SaveBoardLayout(projectID string, columns []model.BoardColumn, moves map[string]string) (BoardLayoutResult, error) {
	projects, err := a.store.Load()
	if err != nil {
		return BoardLayoutResult{}, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return BoardLayoutResult{}, fmt.Errorf("project not found")
	}
	moved, err := ghsync.ApplyBoardLayout(&projects[idx], columns, moves)
	if err != nil {
		return BoardLayoutResult{}, err
	}
	if err := a.store.Save(projects); err != nil {
		return BoardLayoutResult{}, err
	}
	p := projects[idx]
	return BoardLayoutResult{Columns: p.Columns, Tasks: p.Tasks, Moved: moved}, nil
}

// PatchTasks saves board edits without overwriting changes made elsewhere
// since the board loaded (MCP, agents, sync, another window). Each change
// carries the task as the board last saw it; only fields the user changed
// are applied (see internal/taskmerge). Returns the saved task list so the
// board can adopt everything that changed meanwhile.
func (a *App) PatchTasks(projectID string, changes []taskmerge.Change, deletes []string) ([]model.Task, error) {
	p, err := a.projectByID(projectID)
	if err != nil {
		return nil, err
	}
	merged := taskmerge.Apply(p.Tasks, changes, deletes)
	if err := a.UpdateTasks(projectID, merged); err != nil {
		return nil, err
	}
	saved, err := a.projectByID(projectID)
	if err != nil {
		return nil, err
	}
	return saved.Tasks, nil
}

// GetProjectTasks returns a project's current tasks so an open board can
// pick up changes made elsewhere.
func (a *App) GetProjectTasks(projectID string) ([]model.Task, error) {
	p, err := a.projectByID(projectID)
	if err != nil {
		return nil, err
	}
	return p.Tasks, nil
}
