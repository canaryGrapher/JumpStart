package mcpserver

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/taskjson"
)

func registerProjectJSONTools(server *mcp.Server, host Host) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "export_project",
		Description: "Export a whole project as JSON: project fields, sprints, columns, quarters and every task with all fields and status. " +
			"Process environment values are redacted unless includeEnv is true. Attachment files are listed, not included.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID  string `json:"projectId" jsonschema:"JumpStart project id"`
		IncludeEnv bool   `json:"includeEnv,omitempty" jsonschema:"Include process environment values (may contain secrets)"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		data, err := taskjson.Export(*p, in.IncludeEnv, time.Now())
		if err != nil {
			return toolError(err)
		}
		return textResult(string(data))
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "import_project",
		Description: "Merge project JSON (an export_project document, a project object, or {\"tasks\":[...]}) into a project. " +
			"mode merge (default) updates tasks by id and adds new ones; replace also removes tasks absent from the JSON. " +
			"Name, description, quarters and new sprints are applied; root, processes, columns, icon and GitHub settings are not. " +
			"Use dryRun to preview the added/updated/removed tasks first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		ProjectID string `json:"projectId" jsonschema:"JumpStart project id"`
		JSON      string `json:"json" jsonschema:"Project JSON to import"`
		Mode      string `json:"mode,omitempty" jsonschema:"merge or replace"`
		DryRun    bool   `json:"dryRun,omitempty" jsonschema:"Only report what would change"`
	}) (*mcp.CallToolResult, any, error) {
		p, err := host.GetProject(in.ProjectID)
		if err != nil {
			return toolError(err)
		}
		out, pv, err := taskjson.Plan(*p, []byte(in.JSON), taskjson.Mode(in.Mode), time.Now())
		if err != nil {
			return toolError(err)
		}
		if in.DryRun {
			return textResult(map[string]any{"dryRun": true, "preview": pv})
		}
		saved := *p
		saved.Name, saved.Description, saved.Quarters, saved.Sprints = out.Name, out.Description, out.Quarters, out.Sprints
		if err := host.SaveProject(saved); err != nil {
			return toolError(fmt.Errorf("saving project fields: %w", err))
		}
		if err := host.UpdateTasks(in.ProjectID, out.Tasks); err != nil {
			return toolError(err)
		}
		return textResult(map[string]any{"dryRun": false, "preview": pv})
	})
}
