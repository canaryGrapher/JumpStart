package main

import (
	"fmt"
	"time"

	"devdeck/internal/model"
	"devdeck/internal/taskjson"
)

// ExportProjectJSON returns a project's full JSON (see internal/taskjson).
// Process environment values are redacted unless includeEnv is true.
func (a *App) ExportProjectJSON(projectID string, includeEnv bool) (string, error) {
	p, err := a.projectByID(projectID)
	if err != nil {
		return "", err
	}
	data, err := taskjson.Export(p, includeEnv, time.Now())
	return string(data), err
}

// PreviewProjectJSON reports what importing text would change, without
// saving anything. mode is "merge" (default) or "replace".
func (a *App) PreviewProjectJSON(projectID, text, mode string) (taskjson.Preview, error) {
	p, err := a.projectByID(projectID)
	if err != nil {
		return taskjson.Preview{}, err
	}
	_, pv, err := taskjson.Plan(p, []byte(text), taskjson.Mode(mode), time.Now())
	return pv, err
}

// ApplyProjectJSON imports text into the project and returns what changed.
// Project-level fields (name, description, quarters, new sprints) are saved
// first; tasks then go through UpdateTasks so GitHub links, attachment
// cleanup and sync behave exactly as for edits made on the board.
func (a *App) ApplyProjectJSON(projectID, text, mode string) (taskjson.Preview, error) {
	projects, err := a.store.Load()
	if err != nil {
		return taskjson.Preview{}, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return taskjson.Preview{}, fmt.Errorf("project not found")
	}
	out, pv, err := taskjson.Plan(projects[idx], []byte(text), taskjson.Mode(mode), time.Now())
	if err != nil {
		return pv, err
	}
	projects[idx].Name = out.Name
	projects[idx].Description = out.Description
	projects[idx].Quarters = out.Quarters
	projects[idx].Sprints = out.Sprints
	if err := a.store.Save(projects); err != nil {
		return pv, err
	}
	return pv, a.UpdateTasks(projectID, out.Tasks)
}

func (a *App) projectByID(projectID string) (model.Project, error) {
	projects, err := a.store.Load()
	if err != nil {
		return model.Project{}, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return model.Project{}, fmt.Errorf("project not found")
	}
	return projects[idx], nil
}
