package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/attachments"
	"devdeck/internal/daterange"
	"devdeck/internal/model"
)

// resolveRealFileUnderRoot resolves rel inside root and follows symlinks on
// both sides, so a link inside the project cannot point the tool at a file
// outside it. The target must be an existing regular file.
func resolveRealFileUnderRoot(root, rel string) (string, error) {
	lexical, err := resolveUnderRoot(root, rel)
	if err != nil {
		return "", err
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(lexical)
	if err != nil {
		return "", err
	}
	realRoot = filepath.Clean(realRoot)
	if realPath != realRoot && !strings.HasPrefix(realPath, realRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes project root")
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", rel)
	}
	return realPath, nil
}

func findTask(p *model.Project, taskID string) *model.Task {
	for i := range p.Tasks {
		if p.Tasks[i].ID == taskID {
			return &p.Tasks[i]
		}
	}
	return nil
}

func findAttachment(t *model.Task, attID string) (int, bool) {
	for i, a := range t.Attachments {
		if a.ID == attID {
			return i, true
		}
	}
	return -1, false
}

// attachProjectFile copies a file from inside the project root into the
// task's attachment store and returns the new record. The caller saves the
// task; on failure it should call attachments.Remove with the record.
func attachProjectFile(dataDir string, p *model.Project, taskID, rel, name string) (model.Attachment, error) {
	t := findTask(p, taskID)
	if t == nil {
		return model.Attachment{}, fmt.Errorf("task %s not found", taskID)
	}
	src, err := resolveRealFileUnderRoot(p.Root, rel)
	if err != nil {
		return model.Attachment{}, err
	}
	f, err := os.Open(src)
	if err != nil {
		return model.Attachment{}, err
	}
	defer f.Close()
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(src)
	}
	return attachments.Save(dataDir, p.ID, taskID, name, f)
}

// attachmentContent loads an attachment for an agent: text files as text
// (capped), images as image content, anything else as metadata only.
func attachmentContent(dataDir, projectID, taskID string, att model.Attachment) ([]mcp.Content, error) {
	path, err := attachments.Path(dataDir, projectID, taskID, att)
	if err != nil {
		return nil, err
	}
	header := fmt.Sprintf("Attachment %q (%s, %d bytes), id %s", att.Name, att.Mime, att.Size, att.ID)
	switch {
	case attachments.IsImage(att.Mime) && att.Size <= attachments.MaxPreviewBytes:
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return []mcp.Content{
			&mcp.TextContent{Text: header},
			&mcp.ImageContent{Data: raw, MIMEType: att.Mime},
		}, nil
	case isTextLike(att):
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		note := ""
		if len(raw) > maxReadBytes {
			raw = raw[:maxReadBytes]
			note = fmt.Sprintf("\n[truncated to %d bytes]", maxReadBytes)
		}
		return []mcp.Content{&mcp.TextContent{Text: header + "\n\n" + string(raw) + note}}, nil
	}
	return []mcp.Content{&mcp.TextContent{
		Text: header + "\n\nThis file type cannot be read as text; only its metadata is available.",
	}}, nil
}

var textLikeExts = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".csv": true, ".tsv": true, ".json": true,
	".yaml": true, ".yml": true, ".xml": true, ".html": true, ".htm": true, ".log": true,
	".toml": true, ".ini": true, ".sql": true, ".js": true, ".ts": true, ".jsx": true,
	".tsx": true, ".go": true, ".py": true, ".java": true, ".rb": true, ".sh": true,
	".css": true, ".scss": true, ".cls": true, ".trigger": true,
}

func isTextLike(a model.Attachment) bool {
	if strings.HasPrefix(strings.ToLower(a.Mime), "text/") {
		return true
	}
	return textLikeExts[strings.ToLower(filepath.Ext(a.Name))]
}

// QuarterInput is one quarter in a set_quarters call.
type QuarterInput struct {
	Name  string `json:"name,omitempty" jsonschema:"Optional label such as Q1"`
	Start string `json:"start" jsonschema:"First day of the quarter as MM-DD"`
	End   string `json:"end" jsonschema:"Last day of the quarter as MM-DD; may wrap into the next year"`
}

func toQuarterRanges(in []QuarterInput) []model.QuarterRange {
	out := make([]model.QuarterRange, len(in))
	for i, q := range in {
		name := q.Name
		if name == "" {
			name = fmt.Sprintf("Q%d", i+1)
		}
		out[i] = model.QuarterRange{Name: name, Start: q.Start, End: q.End}
	}
	return out
}

func registerAttachmentTools(server *mcp.Server, host Host, dataDir string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "add_task_attachment",
		Description: "Attach a file from inside the project root to a task. The file is copied into JumpStart's attachment store (max 100 MB); " +
			"path is relative to the project root and symlinks that leave the root are rejected. Returns the attachment record.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID    string `json:"taskId" jsonschema:"Task id"`
		Path      string `json:"path" jsonschema:"File path relative to the project root"`
		Name      string `json:"name,omitempty" jsonschema:"Display name; defaults to the file name"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		att, err := attachProjectFile(dataDir, p, in.TaskID, in.Path, in.Name)
		if err != nil {
			return toolError(err)
		}
		t := findTask(p, in.TaskID)
		t.Attachments = append(t.Attachments, att)
		t.UpdatedAt = time.Now().UnixMilli()
		if err := host.UpdateTasks(in.ProjectID, p.Tasks); err != nil {
			_ = attachments.Remove(dataDir, in.ProjectID, in.TaskID, att)
			return toolError(err)
		}
		return textResult(att)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "read_task_attachment",
		Description: "Read a task attachment. Text files return their contents (up to 1 MiB), images return the image, " +
			"and other types (PDF, Office) return metadata only. Use get_task to list attachment ids.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID    string `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID       string `json:"taskId" jsonschema:"Task id"`
		AttachmentID string `json:"attachmentId" jsonschema:"Attachment id from the task"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		t := findTask(p, in.TaskID)
		if t == nil {
			return toolError(fmt.Errorf("task %s not found", in.TaskID))
		}
		i, ok := findAttachment(t, in.AttachmentID)
		if !ok {
			return toolError(fmt.Errorf("attachment %s not found on task", in.AttachmentID))
		}
		content, err := attachmentContent(dataDir, in.ProjectID, in.TaskID, t.Attachments[i])
		if err != nil {
			return toolError(err)
		}
		return &mcp.CallToolResult{Content: content}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_task_attachment",
		Description: "Remove an attachment from a task. JumpStart moves the stored file to a trash folder (purged after 7 days) when the change is saved.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID    string `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID       string `json:"taskId" jsonschema:"Task id"`
		AttachmentID string `json:"attachmentId" jsonschema:"Attachment id from the task"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		t := findTask(p, in.TaskID)
		if t == nil {
			return toolError(fmt.Errorf("task %s not found", in.TaskID))
		}
		i, ok := findAttachment(t, in.AttachmentID)
		if !ok {
			return toolError(fmt.Errorf("attachment %s not found on task", in.AttachmentID))
		}
		t.Attachments = append(t.Attachments[:i:i], t.Attachments[i+1:]...)
		t.UpdatedAt = time.Now().UnixMilli()
		if err := host.UpdateTasks(in.ProjectID, p.Tasks); err != nil {
			return toolError(err)
		}
		return textResult(t)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_quarters",
		Description: "Show the quarter dates used by due-date presets (q1..q4): the app-wide setting, a project's override, " +
			"and the effective set that applies. Dates are year-agnostic MM-DD.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId,omitempty" jsonschema:"Project id; omit for the app-wide setting only"`
	}) (*mcp.CallToolResult, any, error) {
		global := host.GlobalQuarters()
		var project []model.QuarterRange
		if in.ProjectID != "" {
			p, err := host.GetProject(in.ProjectID)
			if err != nil {
				return toolError(err)
			}
			project = p.Quarters
		}
		return textResult(map[string]any{
			"appWide":   global,
			"project":   project,
			"effective": daterange.Effective(project, global),
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "set_quarters",
		Description: "Set which dates make up Q1-Q4 for due-date presets. Give exactly four quarters (start/end as MM-DD, end may wrap into the next year). " +
			"With projectId it sets that project's override; without it, the app-wide default. reset=true removes the override instead.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string         `json:"projectId,omitempty" jsonschema:"Project id; omit for the app-wide default"`
		Quarters  []QuarterInput `json:"quarters,omitempty" jsonschema:"Exactly four quarters in order Q1..Q4"`
		Reset     bool           `json:"reset,omitempty" jsonschema:"Remove the override instead of setting one"`
	}) (*mcp.CallToolResult, any, error) {
		var qs []model.QuarterRange
		if !in.Reset {
			qs = toQuarterRanges(in.Quarters)
			if err := daterange.ValidateQuarters(qs); err != nil {
				return toolError(err)
			}
		}
		if in.ProjectID == "" {
			if err := daterange.SaveGlobal(dataDir, qs); err != nil {
				return toolError(err)
			}
			return textResult(map[string]any{"scope": "app-wide", "quarters": qs})
		}
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		p.Quarters = qs
		if err := host.SaveProject(*p); err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{"scope": "project", "quarters": qs})
	})
}
