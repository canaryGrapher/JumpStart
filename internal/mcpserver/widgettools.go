package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/dashboard"
	"devdeck/internal/model"
	"devdeck/internal/query"
)

func widgetSources(host Host, dataDir string) (dashboard.Sources, error) {
	projects, err := host.ListProjects()
	if err != nil {
		return dashboard.Sources{}, err
	}
	filters, _ := query.AppFilters(dataDir)
	return dashboard.Sources{Projects: projects, GlobalQuarters: host.GlobalQuarters(), AppFilters: filters, Today: time.Now()}, nil
}

func registerWidgetTools(server *mcp.Server, host Host, dataDir string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "widget_schema",
		Description: "Explain the dashboard widget format: types, sizes, due ranges, displays and groupings, " +
			"plus examples. Read this before upsert_widget.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, any, error) {
		return textResult(map[string]any{
			"types": map[string]string{
				"due":      "Tasks due in a range. Needs range.",
				"filter":   "Tasks matching a saved filter. Needs filterId (list_saved_filters).",
				"custom":   "Declarative widget from a task query. Needs query; display picks list/count/bar/donut/table; groupBy buckets bar/donut/table.",
				"html":     "Your own HTML/JS in a sandbox with no network, storage or app access. It receives {type:'jumpstart:data', data, theme} by postMessage (data comes from query) and may post {type:'jumpstart:open', projectId, taskId} or {type:'jumpstart:height', height}.",
				"stats":    "Built-in overview tiles.",
				"flow":     "Built-in task pipeline.",
				"donut":    "Built-in completion donut.",
				"recent":   "Built-in recent projects.",
				"activity": "Built-in activity by project.",
				"ports":    "Built-in live ports.",
				"import":   "Built-in config import panel.",
			},
			"sizes":     []string{"s", "m", "l"},
			"dueRanges": dashboard.DueRanges,
			"displays":  dashboard.Displays,
			"groupBys":  dashboard.GroupBys,
			"query":     "Same shape as query_tasks: duePreset, dueFrom, dueTo, noDueDate, overdue, statuses, priorities, types, sprints, assignees, labels, acceptance (has|none), subtasks (has|none), text.",
			"notes":     "Task widgets hide done tasks unless includeDone is true. projectId limits a widget to one project.",
			"examples": []dashboard.Widget{
				{Type: dashboard.TypeDue, Range: "next_quarter", Size: "s"},
				{Type: dashboard.TypeCustom, Title: "Open bugs by priority", Size: "m", Display: "bar", GroupBy: "priority", Query: &model.TaskQuery{Types: []string{"bug"}}},
				{Type: dashboard.TypeHTML, Title: "Count", Size: "s", Query: &model.TaskQuery{Overdue: true},
					HTML: "<h1 id=n>…</h1><script>addEventListener('message',e=>{if(e.data.type==='jumpstart:data')n.textContent=e.data.data.total})</script>"},
			},
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_widgets",
		Description: "List the dashboard's widgets in display order.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, any, error) {
		return textResult(dashboard.Load(dataDir))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_widget_data",
		Description: "Return the tasks (and groups) a task widget shows: due, filter, custom or html widgets.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		WidgetID string `json:"widgetId" jsonschema:"Widget id from list_widgets"`
	}) (*mcp.CallToolResult, any, error) {
		for _, w := range dashboard.Load(dataDir).Widgets {
			if w.ID == in.WidgetID {
				s, err := widgetSources(host, dataDir)
				if err != nil {
					return toolError(err)
				}
				return textResult(dashboard.WidgetData(w, s))
			}
		}
		return toolError(fmt.Errorf("no widget with id %q", in.WidgetID))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "upsert_widget",
		Description: "Add a dashboard widget, or replace the one with the same id. Call widget_schema first. " +
			"New widgets go at the end. Ask the user before adding html widgets.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in dashboard.Widget) (*mcp.CallToolResult, any, error) {
		w, err := dashboard.Upsert(dataDir, in)
		if err != nil {
			return toolError(err)
		}
		return textResult(w)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_widget",
		Description: "Remove a widget from the dashboard by id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		WidgetID string `json:"widgetId" jsonschema:"Widget id from list_widgets"`
	}) (*mcp.CallToolResult, any, error) {
		if err := dashboard.Remove(dataDir, in.WidgetID); err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{"deleted": in.WidgetID})
	})
}
