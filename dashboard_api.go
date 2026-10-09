package main

import (
	"time"

	"devdeck/internal/analytics"
	"devdeck/internal/dashboard"
	"devdeck/internal/daterange"
	"devdeck/internal/query"
)

// --- Customizable dashboard ---

// GetDashboard returns the widget layout (the default on first run).
func (a *App) GetDashboard() dashboard.Layout {
	return dashboard.Load(analytics.DataDir())
}

// SaveDashboard validates and stores the whole layout.
func (a *App) SaveDashboard(l dashboard.Layout) (dashboard.Layout, error) {
	return dashboard.Save(analytics.DataDir(), l)
}

// ResetDashboard restores the default layout.
func (a *App) ResetDashboard() (dashboard.Layout, error) {
	return dashboard.Save(analytics.DataDir(), dashboard.Default())
}

func (a *App) dashboardSources() (dashboard.Sources, error) {
	dir := analytics.DataDir()
	projects, err := a.store.Load()
	if err != nil {
		return dashboard.Sources{}, err
	}
	filters, _ := query.AppFilters(dir)
	return dashboard.Sources{Projects: projects, GlobalQuarters: daterange.LoadGlobal(dir), AppFilters: filters, Today: time.Now()}, nil
}

// DashboardWidgetData computes a task widget's data. The widget is passed
// in (not looked up) so unsaved edits preview live.
func (a *App) DashboardWidgetData(w dashboard.Widget) (dashboard.Data, error) {
	if err := dashboard.Validate(&w); err != nil {
		return dashboard.Data{WidgetID: w.ID, Error: err.Error()}, nil
	}
	s, err := a.dashboardSources()
	if err != nil {
		return dashboard.Data{}, err
	}
	return dashboard.WidgetData(w, s), nil
}

// ExportWidgets returns a .jumpstart-widget.json document for the widgets.
func (a *App) ExportWidgets(widgets []dashboard.Widget) (string, error) {
	data, err := dashboard.Export(widgets)
	return string(data), err
}

// WidgetImport is a parsed widget file awaiting confirmation.
type WidgetImport struct {
	Widgets  []dashboard.Widget `json:"widgets"`
	Warnings []string           `json:"warnings"`
}

// ParseWidgetImport validates a widget file without adding anything.
func (a *App) ParseWidgetImport(text string) (WidgetImport, error) {
	known := map[string]bool{}
	if projects, err := a.store.Load(); err == nil {
		for _, p := range projects {
			known[p.ID] = true
		}
	}
	ws, warnings, err := dashboard.ParseImport([]byte(text), known)
	if warnings == nil {
		warnings = []string{}
	}
	return WidgetImport{Widgets: ws, Warnings: warnings}, err
}
