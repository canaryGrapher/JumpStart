package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/model"
	"devdeck/internal/taskcsv"
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
	Updated int          `json:"updated"`
	Created int          `json:"created"`
	Skipped int          `json:"skipped"`
	Total   int          `json:"total"`
	Path    string       `json:"path,omitempty"`
	Tasks   []model.Task `json:"tasks"`
}

// ExportTasksCSV writes matching tasks as CSV into a folder chosen via
// OpenDirectoryDialog. An empty filter exports everything. Progress is
// emitted on "tasks:csv:export:progress". An empty result with a nil
// error means the user cancelled the folder picker.
func (a *App) ExportTasksCSV(projectID string, filter taskcsv.Filter) (*CSVExportResult, error) {
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

	name := sanitizeFilename(projects[idx].Name)
	if name == "" {
		name = "tasks"
	}
	filename := name + "-tasks.csv"

	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Choose a folder for the tasks CSV",
		CanCreateDirectories: true,
	})
	if err != nil || dir == "" {
		return nil, err
	}

	path := uniqueCSVPath(filepath.Join(dir, filename))

	var buf bytes.Buffer
	if err := taskcsv.Encode(&buf, selected, func(done, total int) {
		a.emit("tasks:csv:export:progress", map[string]any{
			"projectId": projectID,
			"done":      done,
			"total":     total,
		})
	}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	a.emit("tasks:csv:export:done", map[string]any{
		"projectId": projectID,
		"count":     len(selected),
		"path":      path,
	})
	return &CSVExportResult{Path: path, Filename: filepath.Base(path), Count: len(selected)}, nil
}

// ImportTasksCSV opens a native file picker and merges the CSV into the
// project's tasks. Prefer ImportTasksCSVText when the UI already has the
// file bytes (drag-and-drop / <input type="file">).
func (a *App) ImportTasksCSV(projectID string) (*CSVImportResult, error) {
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
	return a.importTasksCSVBytes(projectID, data, path)
}

// ImportTasksCSVText merges a CSV document already held by the frontend
// (from drag-and-drop or a file input) into the project's tasks.
func (a *App) ImportTasksCSVText(projectID, csvText string) (*CSVImportResult, error) {
	if strings.TrimSpace(csvText) == "" {
		return nil, fmt.Errorf("csv is empty")
	}
	return a.importTasksCSVBytes(projectID, []byte(csvText), "")
}

func (a *App) importTasksCSVBytes(projectID string, data []byte, path string) (*CSVImportResult, error) {
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
	merged, res, err := taskcsv.Apply(before, records, func() string {
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
	if err := a.store.Save(projects); err != nil {
		return nil, err
	}
	a.trackTaskChanges(projectID, before, merged)

	if projects[idx].GitHub != nil && projects[idx].GitHub.Enabled {
		go func() { _, _ = a.runSync(projectID, false) }()
	}

	a.emit("tasks:csv:done", map[string]any{
		"projectId": projectID,
		"updated":   res.Updated,
		"created":   res.Created,
		"skipped":   res.Skipped,
		"total":     res.Total,
	})

	return &CSVImportResult{
		Updated: res.Updated,
		Created: res.Created,
		Skipped: res.Skipped,
		Total:   res.Total,
		Path:    path,
		Tasks:   merged,
	}, nil
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

// uniqueCSVPath returns path if free, otherwise inserts a timestamp before
// the extension so a second export does not silently overwrite the first.
func uniqueCSVPath(path string) string {
	if _, err := os.Stat(path); err != nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s-%s%s", base, time.Now().Format("20060102-150405"), ext)
}
