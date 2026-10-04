package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/model"
	"devdeck/internal/taskcsv"
	"devdeck/internal/tasksheet"
)

// CSVExportResult is returned after a successful bulk download so the UI
// can confirm where the file landed.
type CSVExportResult struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Count    int    `json:"count"`
}

// CSVImportResult is returned to the frontend after a bulk upload so the
// board can adopt the merged task list without a full project reload.
type CSVImportResult struct {
	Updated        int            `json:"updated"`
	Created        int            `json:"created"`
	Removed        int            `json:"removed"`
	Skipped        int            `json:"skipped"`
	Total          int            `json:"total"`
	SprintsCreated int            `json:"sprintsCreated"`
	Mode           string         `json:"mode"`
	Path           string         `json:"path,omitempty"`
	Tasks          []model.Task   `json:"tasks"`
	Sprints        []model.Sprint `json:"sprints"`
}

// ExportTasksCSV writes matching tasks as CSV into a folder chosen via
// OpenDirectoryDialog. Prefer ExportTasksSheet when the UI offers more
// formats. An empty result with a nil error means the user cancelled.
func (a *App) ExportTasksCSV(projectID string, filter taskcsv.Filter) (*CSVExportResult, error) {
	return a.ExportTasksSheet(projectID, string(tasksheet.FormatCSV), filter)
}

// ExportTasksSheet writes matching tasks as csv, xlsx, pdf, or png into a
// folder chosen via OpenDirectoryDialog. format is one of those extensions
// (or "excel" / "image"). Progress is emitted on "tasks:csv:export:progress".
func (a *App) ExportTasksSheet(projectID, format string, filter taskcsv.Filter) (*CSVExportResult, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return nil, fmt.Errorf("project not found")
	}

	selected := taskcsv.Select(projects[idx].Tasks, &filter)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no tasks match the current export filters")
	}

	fmtKind := tasksheet.ParseFormat(format)
	name := sanitizeFilename(projects[idx].Name)
	if name == "" {
		name = "tasks"
	}
	filename := name + "-tasks" + tasksheet.Ext(fmtKind)
	sheetTitle := projects[idx].Name
	if sheetTitle == "" {
		sheetTitle = "Task sheet"
	}

	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Choose a folder for the task sheet",
		CanCreateDirectories: true,
	})
	if err != nil || dir == "" {
		return nil, err
	}

	path := uniqueExportPath(filepath.Join(dir, filename))

	var buf bytes.Buffer
	total := len(selected)
	a.emit("tasks:csv:export:progress", map[string]any{
		"projectId": projectID,
		"done":      0,
		"total":     total,
	})
	if err := tasksheet.Encode(&buf, fmtKind, selected, projects[idx].Sprints, sheetTitle); err != nil {
		return nil, err
	}
	a.emit("tasks:csv:export:progress", map[string]any{
		"projectId": projectID,
		"done":      total,
		"total":     total,
	})
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	a.emit("tasks:csv:export:done", map[string]any{
		"projectId": projectID,
		"count":     len(selected),
		"path":      path,
		"format":    string(fmtKind),
	})
	return &CSVExportResult{Path: path, Filename: filepath.Base(path), Count: len(selected)}, nil
}

// ImportTasksCSV opens a native file picker and merges the CSV into the
// project's tasks. Prefer ImportTasksCSVText when the UI already has the
// file bytes (drag-and-drop / <input type="file">). mode is "add" or
// "replace" (see taskcsv.ParseMode).
func (a *App) ImportTasksCSV(projectID, mode string) (*CSVImportResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import tasks from CSV",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return a.importTasksCSVBytes(projectID, data, path, taskcsv.ParseMode(mode))
}

// ImportTasksCSVText merges a CSV document already held by the frontend
// (from drag-and-drop or a file input) into the project's tasks. mode is
// "add" (incremental upsert) or "replace" (CSV becomes the full board).
func (a *App) ImportTasksCSVText(projectID, csvText, mode string) (*CSVImportResult, error) {
	if strings.TrimSpace(csvText) == "" {
		return nil, fmt.Errorf("csv is empty")
	}
	return a.importTasksCSVBytes(projectID, []byte(csvText), "", taskcsv.ParseMode(mode))
}

func (a *App) importTasksCSVBytes(projectID string, data []byte, path string, mode taskcsv.Mode) (*CSVImportResult, error) {
	records, err := taskcsv.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return nil, fmt.Errorf("project not found")
	}

	before := projects[idx].Tasks
	beforeSprints := projects[idx].Sprints
	merged, sprints, res, err := taskcsv.Apply(before, beforeSprints, records, mode, func() string {
		return uuid.NewString()
	}, func(done, total int) {
		a.emit("tasks:csv:progress", map[string]any{
			"projectId": projectID,
			"done":      done,
			"total":     total,
		})
	})
	if err != nil {
		return nil, err
	}

	projects[idx].Tasks = merged
	projects[idx].Sprints = sprints
	if err := a.store.Save(projects); err != nil {
		return nil, err
	}
	a.trackTaskChanges(projectID, before, merged)
	a.trackSprintChanges(projectID, beforeSprints, sprints)

	if projects[idx].GitHub != nil && projects[idx].GitHub.Enabled {
		ghCfg := projects[idx].GitHub
		names := sprintNamesCreated(beforeSprints, sprints)
		go func() {
			if len(names) > 0 {
				_ = a.ensureGitHubSprints(ghCfg.ProjectID, names)
			}
			_, _ = a.runSync(projectID, false)
		}()
	}

	a.emit("tasks:csv:done", map[string]any{
		"projectId":      projectID,
		"updated":        res.Updated,
		"created":        res.Created,
		"removed":        res.Removed,
		"skipped":        res.Skipped,
		"total":          res.Total,
		"sprintsCreated": res.SprintsCreated,
		"mode":           string(mode),
	})

	return &CSVImportResult{
		Updated:        res.Updated,
		Created:        res.Created,
		Removed:        res.Removed,
		Skipped:        res.Skipped,
		Total:          res.Total,
		SprintsCreated: res.SprintsCreated,
		Mode:           string(mode),
		Path:           path,
		Tasks:          merged,
		Sprints:        sprints,
	}, nil
}

// sprintNamesCreated returns display names of sprints that appear in
// after but not before — the ones CSV import just minted.
func sprintNamesCreated(before, after []model.Sprint) []string {
	had := make(map[string]bool, len(before))
	for _, s := range before {
		had[s.ID] = true
	}
	var names []string
	for _, s := range after {
		if !had[s.ID] && strings.TrimSpace(s.Name) != "" {
			names = append(names, s.Name)
		}
	}
	return names
}

// ensureGitHubSprints creates missing Iteration cycles on the linked
// Projects v2 board for the given sprint titles.
func (a *App) ensureGitHubSprints(boardID string, titles []string) error {
	if boardID == "" || len(titles) == 0 {
		return nil
	}
	client, err := a.ghClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()
	return client.EnsureIterations(ctx, boardID, titles)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// uniqueExportPath returns path if free, otherwise inserts a timestamp before
// the extension so a second export does not silently overwrite the first.
func uniqueExportPath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s-%s%s", base, time.Now().Format("20060102-150405"), ext)
}
