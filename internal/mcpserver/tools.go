package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/model"
)

func registerTools(server *mcp.Server, host Host) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "List JumpStart projects (id, name, root, process/task counts).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		projects, err := host.ListProjects()
		if err != nil {
			return toolError(err)
		}
		type row struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Root          string `json:"root"`
			ProcessCount  int    `json:"processCount"`
			TaskCount     int    `json:"taskCount"`
			TasksEnabled  bool   `json:"tasksEnabled"`
			Favorite      bool   `json:"favorite"`
			Description   string `json:"description,omitempty"`
		}
		out := make([]row, 0, len(projects))
		for _, p := range projects {
			out = append(out, row{
				ID: p.ID, Name: p.Name, Root: p.Root,
				ProcessCount: len(p.Processes), TaskCount: len(p.Tasks),
				TasksEnabled: p.TasksEnabled, Favorite: p.Favorite,
				Description: p.Description,
			})
		}
		return textResult(out)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_project",
		Description: "Get one JumpStart project by id, including processes and tasks.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		return textResult(p)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_processes",
		Description: "List subprocesses for a project with live status (running, pid, ports).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		type row struct {
			ID      string            `json:"id"`
			Name    string            `json:"name"`
			Dir     string            `json:"dir"`
			Command string            `json:"command"`
			Env     map[string]string `json:"env,omitempty"`
			Status  model.Status      `json:"status"`
		}
		out := make([]row, 0, len(p.Processes))
		for _, proc := range p.Processes {
			out = append(out, row{
				ID: proc.ID, Name: proc.Name, Dir: proc.Dir,
				Command: proc.Command, Env: proc.Env,
				Status: host.ProcessStatus(proc.ID),
			})
		}
		return textResult(out)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_process",
		Description: "Start one subprocess by project and process id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		ProcessID string `json:"processId" jsonschema:"Process id"`
	}) (*mcp.CallToolResult, any, error) {
		return toolError(host.StartProcess(in.ProjectID, in.ProcessID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_process",
		Description: "Stop one running subprocess by process id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProcessID string `json:"processId" jsonschema:"Process id"`
	}) (*mcp.CallToolResult, any, error) {
		return toolError(host.StopProcess(in.ProcessID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "start_all_processes",
		Description: "Start every subprocess in a project. Returns per-process error strings.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		errs := host.StartAll(in.ProjectID)
		if len(errs) == 0 {
			return textResult(map[string]any{"ok": true, "errors": []string{}})
		}
		return textResult(map[string]any{"ok": false, "errors": errs})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop_all_processes",
		Description: "Stop every subprocess in a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		host.StopAll(in.ProjectID)
		return textResult("ok")
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_process_status",
		Description: "Get live status for one process (running, pid, ports, exit code).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProcessID string `json:"processId" jsonschema:"Process id"`
	}) (*mcp.CallToolResult, any, error) {
		return textResult(host.ProcessStatus(in.ProcessID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_process_logs",
		Description: "Get recent log lines for one process.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProcessID string `json:"processId" jsonschema:"Process id"`
		Tail      int    `json:"tail,omitempty" jsonschema:"Optional max number of trailing lines (0 = all buffered)"`
	}) (*mcp.CallToolResult, any, error) {
		logs := host.ProcessLogs(in.ProcessID)
		if in.Tail > 0 && in.Tail < len(logs) {
			logs = logs[len(logs)-in.Tail:]
		}
		return textResult(strings.Join(logs, "\n"))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_process",
		Description: "Edit a subprocess definition (name, dir, command, env). Pass only fields to change.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string            `json:"projectId" jsonschema:"JumpStart project id"`
		ProcessID string            `json:"processId" jsonschema:"Process id"`
		Name      string            `json:"name,omitempty" jsonschema:"New display name"`
		Dir       string            `json:"dir,omitempty" jsonschema:"New working directory"`
		Command   string            `json:"command,omitempty" jsonschema:"New start command"`
		Env       map[string]string `json:"env,omitempty" jsonschema:"Replace env map when provided"`
		ClearEnv  bool              `json:"clearEnv,omitempty" jsonschema:"If true, clear env even when env is empty"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		found := false
		for i := range p.Processes {
			if p.Processes[i].ID != in.ProcessID {
				continue
			}
			found = true
			if in.Name != "" {
				p.Processes[i].Name = in.Name
			}
			if in.Dir != "" {
				p.Processes[i].Dir = in.Dir
			}
			if in.Command != "" {
				p.Processes[i].Command = in.Command
			}
			if in.ClearEnv {
				p.Processes[i].Env = map[string]string{}
			} else if in.Env != nil {
				p.Processes[i].Env = in.Env
			}
			break
		}
		if !found {
			return toolError(fmt.Errorf("process %s not found", in.ProcessID))
		}
		if err := host.SaveProject(*p); err != nil {
			return toolError(err)
		}
		return textResult(p.Processes)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_process",
		Description: "Add a new subprocess to a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string            `json:"projectId" jsonschema:"JumpStart project id"`
		Name      string            `json:"name" jsonschema:"Display name"`
		Dir       string            `json:"dir" jsonschema:"Working directory"`
		Command   string            `json:"command" jsonschema:"Start command"`
		Env       map[string]string `json:"env,omitempty" jsonschema:"Optional environment variables"`
	}) (*mcp.CallToolResult, any, error) {
		if err := require(in.Name != "" && in.Command != "", "name and command are required"); err != nil {
			return toolError(err)
		}
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		proc := model.Process{
			ID:      uuid.NewString(),
			Name:    in.Name,
			Dir:     in.Dir,
			Command: in.Command,
			Env:     in.Env,
		}
		if proc.Dir == "" {
			proc.Dir = p.Root
		}
		p.Processes = append(p.Processes, proc)
		if err := host.SaveProject(*p); err != nil {
			return toolError(err)
		}
		return textResult(proc)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List kanban tasks for a project. Optional status filter: backlog|todo|inprogress|testing|done.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		Status    string `json:"status,omitempty" jsonschema:"Optional kanban column filter"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		status := strings.ToLower(strings.TrimSpace(in.Status))
		out := make([]model.Task, 0, len(p.Tasks))
		for _, t := range p.Tasks {
			if status != "" && strings.ToLower(t.Status) != status {
				continue
			}
			out = append(out, t)
		}
		return textResult(out)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Description: "Get one task by id within a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID    string `json:"taskId" jsonschema:"Task id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		for _, t := range p.Tasks {
			if t.ID == in.TaskID {
				return textResult(t)
			}
		}
		return toolError(fmt.Errorf("task %s not found", in.TaskID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "upsert_task",
		Description: "Create or update a kanban task. Omit taskId to create. Status: backlog|todo|inprogress|testing|done. Type: story|task|bug.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID   string   `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID      string   `json:"taskId,omitempty" jsonschema:"Existing task id to update; omit to create"`
		Title       string   `json:"title,omitempty" jsonschema:"Task title"`
		Description string   `json:"description,omitempty" jsonschema:"Task description"`
		Status      string   `json:"status,omitempty" jsonschema:"Kanban column"`
		Type        string   `json:"type,omitempty" jsonschema:"story|task|bug"`
		Priority    string   `json:"priority,omitempty" jsonschema:"low|medium|high"`
		Labels      []string `json:"labels,omitempty" jsonschema:"Labels"`
		Assignee    string   `json:"assignee,omitempty" jsonschema:"Assignee"`
		ParentID    string   `json:"parentId,omitempty" jsonschema:"Parent story id"`
		SprintID    string   `json:"sprintId,omitempty" jsonschema:"Sprint id; empty for backlog"`
		StoryPoints int      `json:"storyPoints,omitempty" jsonschema:"Story points"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		now := time.Now().UnixMilli()
		if in.TaskID == "" {
			if err := require(in.Title != "", "title is required when creating a task"); err != nil {
				return toolError(err)
			}
			t := model.Task{
				ID:          uuid.NewString(),
				Title:       in.Title,
				Description: in.Description,
				Status:      defaultStatus(in.Status),
				Type:        defaultType(in.Type),
				Priority:    in.Priority,
				Labels:      in.Labels,
				Assignee:    in.Assignee,
				ParentID:    in.ParentID,
				SprintID:    in.SprintID,
				StoryPoints: in.StoryPoints,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			p.Tasks = append(p.Tasks, t)
			if err := host.UpdateTasks(in.ProjectID, p.Tasks); err != nil {
				return toolError(err)
			}
			return textResult(t)
		}
		found := false
		var updated model.Task
		for i := range p.Tasks {
			if p.Tasks[i].ID != in.TaskID {
				continue
			}
			found = true
			if in.Title != "" {
				p.Tasks[i].Title = in.Title
			}
			if in.Description != "" {
				p.Tasks[i].Description = in.Description
			}
			if in.Status != "" {
				p.Tasks[i].Status = in.Status
			}
			if in.Type != "" {
				p.Tasks[i].Type = in.Type
			}
			if in.Priority != "" {
				p.Tasks[i].Priority = in.Priority
			}
			if in.Labels != nil {
				p.Tasks[i].Labels = in.Labels
			}
			if in.Assignee != "" {
				p.Tasks[i].Assignee = in.Assignee
			}
			if in.ParentID != "" {
				p.Tasks[i].ParentID = in.ParentID
			}
			if in.SprintID != "" {
				p.Tasks[i].SprintID = in.SprintID
			}
			if in.StoryPoints != 0 {
				p.Tasks[i].StoryPoints = in.StoryPoints
			}
			p.Tasks[i].UpdatedAt = now
			updated = p.Tasks[i]
			break
		}
		if !found {
			return toolError(fmt.Errorf("task %s not found", in.TaskID))
		}
		if err := host.UpdateTasks(in.ProjectID, p.Tasks); err != nil {
			return toolError(err)
		}
		return textResult(updated)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_task",
		Description: "Delete a task from a project's board.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		TaskID    string `json:"taskId" jsonschema:"Task id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		out := p.Tasks[:0]
		found := false
		for _, t := range p.Tasks {
			if t.ID == in.TaskID {
				found = true
				continue
			}
			out = append(out, t)
		}
		if !found {
			return toolError(fmt.Errorf("task %s not found", in.TaskID))
		}
		if err := host.UpdateTasks(in.ProjectID, out); err != nil {
			return toolError(err)
		}
		return textResult("ok")
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "List files and folders under a project root path (relative paths only).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		Path      string `json:"path,omitempty" jsonschema:"Relative path inside the project root (default .)"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		entries, err := listProjectDir(p.Root, in.Path)
		if err != nil {
			return toolError(err)
		}
		return textResult(entries)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_file",
		Description: "Read a UTF-8 text file inside a project root (max 1 MiB). Path is relative.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		Path      string `json:"path" jsonschema:"Relative file path inside the project root"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		text, err := readProjectFile(p.Root, in.Path)
		if err != nil {
			return toolError(err)
		}
		return textResult(text)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Create or overwrite a text file inside a project root. Path is relative. Creates parent dirs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		Path      string `json:"path" jsonschema:"Relative file path inside the project root"`
		Content   string `json:"content" jsonschema:"Full file contents to write"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		if err := writeProjectFile(p.Root, in.Path, in.Content); err != nil {
			return toolError(err)
		}
		return textResult("ok")
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "git_status",
		Description: "Git status for a project's repository (branch, dirty, ahead/behind).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		st, err := host.GitStatus(p.Root)
		if err != nil {
			return toolError(err)
		}
		return textResult(st)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "git_diff",
		Description: "Git diff for a project. Mode: working (default), staged, or all.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		Mode      string `json:"mode,omitempty" jsonschema:"working|staged|all"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		mode := in.Mode
		if mode == "" {
			mode = "working"
		}
		diff, err := host.GitDiff(p.Root, mode)
		if err != nil {
			return toolError(err)
		}
		return textResult(diff)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "git_working_changes",
		Description: "List working-tree file changes for a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		changes, err := host.GitWorkingChanges(p.Root)
		if err != nil {
			return toolError(err)
		}
		return textResult(changes)
	})
}

func defaultStatus(s string) string {
	if s == "" {
		return "backlog"
	}
	return s
}

func defaultType(t string) string {
	if t == "" {
		return "task"
	}
	return t
}
