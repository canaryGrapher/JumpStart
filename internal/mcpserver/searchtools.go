package mcpserver

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/appsettings"
	"devdeck/internal/search"
)

// TextProvider is implemented by hosts that can read attachment contents
// (text files, PDFs, OCR of images) for search. Without it, search covers
// task fields and file names only.
type TextProvider interface {
	AttachmentText() search.TextSource
}

func registerSearchTools(server *mcp.Server, host Host, dataDir string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "search",
		Description: "Search every project: project names, all task fields, link titles, file names and attachment text (including text read from screenshots). " +
			"Every word must match. Dates in the query (today, this week, next friday, Oct 9, 10/9/2026) filter by due date. " +
			"Qualifiers: status:todo, priority:high, type:bug, @assignee, #label, project:name, due:next week, created:this month, has:file|link|criteria|subtasks, no:due, is:overdue|open|done.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct {
		Query     string `json:"query" jsonschema:"Words, dates and qualifiers"`
		ProjectID string `json:"projectId,omitempty" jsonschema:"Limit to one project"`
		HideDone  bool   `json:"hideDone,omitempty" jsonschema:"Leave out done tasks"`
		Limit     int    `json:"limit,omitempty" jsonschema:"Maximum results (default 50)"`
	}) (*mcp.CallToolResult, any, error) {
		projects, err := host.ListProjects()
		if err != nil {
			return toolError(err)
		}
		var text search.TextSource
		if tp, ok := host.(TextProvider); ok {
			text = tp.AttachmentText()
		}
		resp := search.Search(projects, in.Query, search.Options{
			ProjectID: in.ProjectID, HideDone: in.HideDone, Limit: in.Limit, Today: time.Now(),
			DateOrder: appsettings.Load(dataDir).DateOrder, GlobalQuarters: host.GlobalQuarters(), Text: text,
		})
		return textResult(resp)
	})
}
