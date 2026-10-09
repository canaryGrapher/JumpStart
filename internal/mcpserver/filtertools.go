package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/model"
	"devdeck/internal/query"
)

func registerFilterTools(server *mcp.Server, host Host, dataDir string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_saved_filters",
		Description: "List saved task filters: app-wide ones (including the deletable defaults) and, with projectId, that project's own filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId,omitempty" jsonschema:"Project id for project-only filters"`
	}) (*mcp.CallToolResult, any, error) {
		app, err := query.AppFilters(dataDir)
		if err != nil {
			return toolError(err)
		}
		out := map[string]any{"appWide": app, "project": []model.SavedFilter{}}
		if in.ProjectID != "" {
			p, err := host.GetProject(in.ProjectID)
			if err != nil {
				return toolError(err)
			}
			if p.SavedFilters != nil {
				out["project"] = p.SavedFilters
			}
		}
		return textResult(out)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "run_saved_filter",
		Description: "Return a project's tasks matching a saved filter (app-wide or project). Find ids with list_saved_filters.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"Project to filter"`
		FilterID  string `json:"filterId" jsonschema:"Saved filter id"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		app, err := query.AppFilters(dataDir)
		if err != nil {
			return toolError(err)
		}
		var found *model.SavedFilter
		for _, list := range [][]model.SavedFilter{p.SavedFilters, app} {
			for i := range list {
				if list[i].ID == in.FilterID {
					found = &list[i]
				}
			}
		}
		if found == nil {
			return toolError(fmt.Errorf("filter %s not found", in.FilterID))
		}
		tasks, err := query.Run(found.Query, *p, host.GlobalQuarters(), time.Now())
		if err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{"filter": found.Name, "tasks": tasks})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "query_tasks",
		Description: "Return tasks matching a TaskQuery, in one project or (without projectId) across all projects. Fields: duePreset " +
			"(today, tomorrow, last_week, this_week, next_week, this_month, next_month, last_quarter, this_quarter, next_quarter, q1-q4, this_year), " +
			"dueFrom/dueTo (YYYY-MM-DD), noDueDate, overdue, statuses, priorities (\"none\" = unset), types, sprints (\"\" = backlog), " +
			"assignees (\"__none__\" = unassigned), labels, acceptance/subtasks (\"has\"|\"none\"), text.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string          `json:"projectId,omitempty" jsonschema:"Limit to one project"`
		Query     model.TaskQuery `json:"query" jsonschema:"Filter"`
	}) (*mcp.CallToolResult, any, error) {
		type hit struct {
			ProjectID   string     `json:"projectId"`
			ProjectName string     `json:"projectName"`
			Task        model.Task `json:"task"`
		}
		var projects []model.Project
		if in.ProjectID != "" {
			p, err := host.GetProject(in.ProjectID)
			if err != nil {
				return toolError(err)
			}
			projects = []model.Project{*p}
		} else {
			all, err := host.ListProjects()
			if err != nil {
				return toolError(err)
			}
			projects = all
		}
		out := []hit{}
		for _, p := range projects {
			tasks, err := query.Run(in.Query, p, host.GlobalQuarters(), time.Now())
			if err != nil {
				return toolError(err)
			}
			for _, t := range tasks {
				out = append(out, hit{ProjectID: p.ID, ProjectName: p.Name, Task: t})
			}
		}
		return textResult(out)
	})
}
