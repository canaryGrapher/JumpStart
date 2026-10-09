package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/analytics"
	"devdeck/internal/attachments"
	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

// --- Task attachments and links ---
//
// Files are copied into ~/.jumpstart/attachments/<project>/<task>/ and the
// returned Attachment records are saved with the task through UpdateTasks, so
// the board stays the single source of truth. These bindings never trust a
// path from the frontend: files are always addressed by the project and task
// IDs plus the generated stored file name.

// PickTaskAttachments opens a native multi-file picker and copies the chosen
// files into the task's attachment store. If any file fails (for example it
// exceeds the size limit) the whole batch is rolled back.
func (a *App) PickTaskAttachments(projectID, taskID string) ([]model.Attachment, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Attach files to task",
	})
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	dataDir := analytics.DataDir()
	saved := make([]model.Attachment, 0, len(paths))
	rollback := func() {
		for _, s := range saved {
			_ = attachments.Remove(dataDir, projectID, taskID, s)
		}
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			rollback()
			return nil, err
		}
		if info.IsDir() {
			rollback()
			return nil, fmt.Errorf("%s is a folder; attach individual files", info.Name())
		}
		f, err := os.Open(path)
		if err != nil {
			rollback()
			return nil, err
		}
		att, err := attachments.Save(dataDir, projectID, taskID, info.Name(), f)
		f.Close()
		if err != nil {
			rollback()
			return nil, err
		}
		saved = append(saved, att)
	}
	return saved, nil
}

// AddTaskAttachmentData stores a file sent as base64, used for pasted
// screenshots and drag-and-drop where the app has bytes but no path.
func (a *App) AddTaskAttachmentData(projectID, taskID, name, base64Data string) (model.Attachment, error) {
	return attachments.SaveBase64(analytics.DataDir(), projectID, taskID, name, base64Data)
}

// TaskAttachmentPreview returns an image attachment as a data: URL so the
// board can show it inline without exposing the file system.
func (a *App) TaskAttachmentPreview(projectID, taskID string, att model.Attachment) (string, error) {
	return attachments.DataURL(analytics.DataDir(), projectID, taskID, att)
}

// OpenTaskAttachment opens the file in the system's default application.
func (a *App) OpenTaskAttachment(projectID, taskID string, att model.Attachment) error {
	p, err := attachments.Path(analytics.DataDir(), projectID, taskID, att)
	if err != nil {
		return err
	}
	return attachments.Open(p)
}

// PreviewTaskAttachment shows the file in Preview (images, PDFs) or Quick
// Look on macOS, and in the default application elsewhere.
func (a *App) PreviewTaskAttachment(projectID, taskID string, att model.Attachment) error {
	p, err := attachments.Path(analytics.DataDir(), projectID, taskID, att)
	if err != nil {
		return err
	}
	return attachments.Preview(p, att.Mime)
}

// RevealTaskAttachment shows the file in Finder / Explorer.
func (a *App) RevealTaskAttachment(projectID, taskID string, att model.Attachment) error {
	p, err := attachments.Path(analytics.DataDir(), projectID, taskID, att)
	if err != nil {
		return err
	}
	return attachments.Reveal(p)
}

// DiscardTaskAttachment deletes a file that was uploaded but never saved with
// its task, e.g. when the task modal is cancelled. It refuses to delete a
// file the saved board still references.
func (a *App) DiscardTaskAttachment(projectID, taskID string, att model.Attachment) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	for _, p := range projects {
		if p.ID != projectID {
			continue
		}
		for _, t := range p.Tasks {
			if t.ID != taskID {
				continue
			}
			for _, saved := range t.Attachments {
				if saved.ID == att.ID {
					return fmt.Errorf("attachment %s is saved on the task; remove it from the task instead", att.Name)
				}
			}
		}
	}
	return attachments.Remove(analytics.DataDir(), projectID, taskID, att)
}

// OpenTaskLink opens a task link in the user's default web browser. Only
// http, https, and mailto URLs are allowed.
func (a *App) OpenTaskLink(rawURL string) error {
	u, err := attachments.NormalizeURL(rawURL)
	if err != nil {
		return err
	}
	runtime.BrowserOpenURL(a.ctx, u)
	return nil
}

// cleanupRemovedAttachments moves stored files for attachments or whole tasks
// that disappeared between two versions of a project's task list into the
// recoverable trash (purged after a week).
func cleanupRemovedAttachments(projectID string, before, after []model.Task) {
	dataDir := analytics.DataDir()
	files, deleted := attachments.Removed(before, after)
	for taskID, list := range files {
		for _, att := range list {
			_ = attachments.Trash(dataDir, projectID, taskID, att)
		}
	}
	for _, taskID := range deleted {
		_ = attachments.TrashTask(dataDir, projectID, taskID)
	}
}

// --- Due-date ranges and quarter settings ---

// QuarterSettings is what Preferences → Calendar renders.
type QuarterSettings struct {
	Quarters []model.QuarterRange `json:"quarters"`
	// Custom is false when no app-wide override is saved and Quarters is the
	// calendar-year default.
	Custom bool `json:"custom"`
}

// GetQuarterSettings returns the app-wide quarter dates.
func (a *App) GetQuarterSettings() QuarterSettings {
	if qs := daterange.LoadGlobal(analytics.DataDir()); qs != nil {
		return QuarterSettings{Quarters: qs, Custom: true}
	}
	return QuarterSettings{Quarters: daterange.DefaultQuarters()}
}

// SetQuarterSettings saves the app-wide quarter dates. An empty list restores
// the calendar-year default.
func (a *App) SetQuarterSettings(quarters []model.QuarterRange) error {
	return daterange.SaveGlobal(analytics.DataDir(), quarters)
}

// ResolveDueRange turns a preset (this_week, q2, ...) into inclusive dates.
// projectQuarters is the project's own override, if any; it wins over the
// app-wide setting.
func (a *App) ResolveDueRange(preset string, projectQuarters []model.QuarterRange) (daterange.Range, error) {
	qs := daterange.Effective(projectQuarters, daterange.LoadGlobal(analytics.DataDir()))
	return daterange.Resolve(preset, time.Now(), qs)
}
