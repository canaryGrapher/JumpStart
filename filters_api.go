package main

import (
	"fmt"
	"time"

	"devdeck/internal/analytics"
	"devdeck/internal/model"
	"devdeck/internal/query"
)

// SavedFilterLists groups the filters a board can offer.
type SavedFilterLists struct {
	AppWide []model.SavedFilter `json:"appWide"`
	Project []model.SavedFilter `json:"project"`
}

// ListSavedFilters returns app-wide filters (seeding defaults once) and the
// project's own filters.
func (a *App) ListSavedFilters(projectID string) (SavedFilterLists, error) {
	app, err := query.AppFilters(analytics.DataDir())
	if err != nil {
		return SavedFilterLists{}, err
	}
	out := SavedFilterLists{AppWide: app, Project: []model.SavedFilter{}}
	if projectID != "" {
		if p, err := a.projectByID(projectID); err == nil && p.SavedFilters != nil {
			out.Project = p.SavedFilters
		}
	}
	return out, nil
}

// SaveFilter creates or updates a saved filter. scope is "app" or "project".
func (a *App) SaveFilter(scope, projectID string, f model.SavedFilter) (model.SavedFilter, error) {
	switch scope {
	case "app":
		return query.SaveAppFilter(analytics.DataDir(), f)
	case "project":
		var saved model.SavedFilter
		err := a.withProject(projectID, func(p *model.Project) error {
			list, s, err := query.Upsert(p.SavedFilters, f, time.Now())
			if err != nil {
				return err
			}
			p.SavedFilters, saved = list, s
			return nil
		})
		return saved, err
	}
	return f, fmt.Errorf("scope must be app or project")
}

// DeleteSavedFilter removes a saved filter (defaults included).
func (a *App) DeleteSavedFilter(scope, projectID, id string) error {
	switch scope {
	case "app":
		return query.DeleteAppFilter(analytics.DataDir(), id)
	case "project":
		return a.withProject(projectID, func(p *model.Project) error {
			list, found := query.Remove(p.SavedFilters, id)
			if !found {
				return fmt.Errorf("filter %s not found", id)
			}
			p.SavedFilters = list
			return nil
		})
	}
	return fmt.Errorf("scope must be app or project")
}

// RestoreDefaultFilters re-adds any default app-wide filter that was deleted.
func (a *App) RestoreDefaultFilters() ([]model.SavedFilter, error) {
	return query.RestoreDefaults(analytics.DataDir())
}

// withProject loads, edits and saves one project.
func (a *App) withProject(projectID string, edit func(p *model.Project) error) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return fmt.Errorf("project not found")
	}
	if err := edit(&projects[idx]); err != nil {
		return err
	}
	return a.store.Save(projects)
}
