package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"devdeck/internal/daterange"
	"devdeck/internal/gitops"
	"devdeck/internal/model"
	"devdeck/internal/search"
)

// memHost is an in-memory Host holding one project.
type memHost struct {
	stubHost
	project model.Project
	global  []model.QuarterRange
}

func (h *memHost) GetProject(id string) (*model.Project, error) {
	if id != h.project.ID {
		return nil, errNotFound
	}
	cp := h.project
	cp.Tasks = append([]model.Task(nil), h.project.Tasks...)
	return &cp, nil
}
func (h *memHost) SaveProject(p model.Project) error { h.project = p; return nil }
func (h *memHost) UpdateTasks(_ string, tasks []model.Task) error {
	h.project.Tasks = tasks
	return nil
}
func (h *memHost) GlobalQuarters() []model.QuarterRange     { return h.global }
func (h *memHost) GitStatus(string) (*gitops.Status, error) { return nil, nil }

// call invokes a tool over an in-memory MCP connection.
func connect(t *testing.T, host Host, dataDir string) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	registerTools(server, host, dataDir)
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func call(t *testing.T, s *mcp.ClientSession, tool string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	return res
}

func text(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func mustOK(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool error: %s", text(res))
	}
	return text(res)
}

func TestEndToEndTaskFieldsOverMCP(t *testing.T) {
	root := t.TempDir()
	host := &memHost{project: model.Project{ID: "p1", Name: "Demo", Root: root}}
	dataDir := t.TempDir()
	s := connect(t, host, dataDir)

	// Create with every new field.
	out := mustOK(t, call(t, s, "upsert_task", map[string]any{
		"projectId": "p1", "title": "Ship invoices", "status": "done", "dueDate": "2026-10-20",
		"milestone":  "v2",
		"subtasks":   []any{map[string]any{"title": "Draft"}, map[string]any{"title": "Review", "done": true}},
		"acceptance": []any{map[string]any{"title": "PDF renders"}},
		"links":      []any{map[string]any{"title": "Spec", "url": "example.com/spec"}},
	}))
	var task model.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	if task.DueDate != "2026-10-20" || task.Milestone != "v2" || !task.Done {
		t.Errorf("fields not stored: %+v", task)
	}
	if len(task.Subtasks) != 2 || !task.Subtasks[1].Done || task.Subtasks[0].ID == "" {
		t.Errorf("subtasks = %+v", task.Subtasks)
	}
	if len(task.Links) != 1 || task.Links[0].URL != "https://example.com/spec" {
		t.Errorf("links = %+v", task.Links)
	}

	// Update: re-sending a title keeps its id and checked state; clear removes fields.
	mustOK(t, call(t, s, "upsert_task", map[string]any{
		"projectId": "p1", "taskId": task.ID,
		"subtasks": []any{map[string]any{"title": "review"}},
		"clear":    []any{"dueDate", "links"},
		"status":   "todo",
	}))
	got := host.project.Tasks[0]
	if got.DueDate != "" || got.Links != nil {
		t.Errorf("clear failed: %+v", got)
	}
	if len(got.Subtasks) != 1 || got.Subtasks[0].ID != task.Subtasks[1].ID || !got.Subtasks[0].Done {
		t.Errorf("checklist id/state not preserved: %+v", got.Subtasks)
	}
	if got.Done || got.Status != "todo" {
		t.Errorf("reopening should clear done: %+v", got)
	}

	// Bad input is rejected with a tool error, not stored.
	for _, args := range []map[string]any{
		{"projectId": "p1", "taskId": task.ID, "dueDate": "someday"},
		{"projectId": "p1", "taskId": task.ID, "links": []any{map[string]any{"url": "javascript:alert(1)"}}},
		{"projectId": "p1", "taskId": task.ID, "clear": []any{"title"}},
	} {
		if res := call(t, s, "upsert_task", args); !res.IsError {
			t.Errorf("expected error for %v, got %s", args, text(res))
		}
	}

	// Notes append rather than replace.
	mustOK(t, call(t, s, "upsert_task", map[string]any{"projectId": "p1", "taskId": task.ID, "description": "Original"}))
	mustOK(t, call(t, s, "append_task_note", map[string]any{"projectId": "p1", "taskId": task.ID, "note": "Blocked on legal"}))
	if d := host.project.Tasks[0].Description; !strings.HasPrefix(d, "Original\n\nNote (") || !strings.HasSuffix(d, "Blocked on legal") {
		t.Errorf("description = %q", d)
	}
}

func TestEndToEndListFiltersAndAttachments(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("Deploy Friday"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pic.png"), []byte("\x89PNG\r\n\x1a\n0000"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "doc.pdf"), []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(t.TempDir(), "secret.txt")
	os.WriteFile(secret, []byte("outside"), 0o644)
	if err := os.Symlink(secret, filepath.Join(root, "link.txt")); err != nil {
		t.Skip("symlinks unavailable")
	}

	host := &memHost{project: model.Project{ID: "p1", Root: root, Tasks: []model.Task{
		{ID: "t1", Title: "One", Status: "todo"},
		{ID: "t2", Title: "Two", Status: "todo", DueDate: "2026-10-09"},
	}}}
	dataDir := t.TempDir()
	s := connect(t, host, dataDir)

	var att model.Attachment
	if err := json.Unmarshal([]byte(mustOK(t, call(t, s, "add_task_attachment", map[string]any{
		"projectId": "p1", "taskId": "t1", "path": "notes.md",
	}))), &att); err != nil {
		t.Fatal(err)
	}
	if att.Name != "notes.md" || len(host.project.Tasks[0].Attachments) != 1 {
		t.Fatalf("attachment not saved: %+v", host.project.Tasks[0].Attachments)
	}
	mustOK(t, call(t, s, "add_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "path": "pic.png"}))
	mustOK(t, call(t, s, "add_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "path": "doc.pdf"}))

	// Escapes are refused.
	for _, path := range []string{"../etc/passwd", "/etc/passwd", "link.txt", "missing.txt", "."} {
		if res := call(t, s, "add_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "path": path}); !res.IsError {
			t.Errorf("path %q should be rejected", path)
		}
	}
	if len(host.project.Tasks[0].Attachments) != 3 {
		t.Errorf("rejected paths must not add attachments: %d", len(host.project.Tasks[0].Attachments))
	}

	// Reads: text, image, and metadata-only.
	atts := host.project.Tasks[0].Attachments
	res := call(t, s, "read_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "attachmentId": atts[0].ID})
	if body := mustOK(t, res); !strings.Contains(body, "Deploy Friday") {
		t.Errorf("text read = %q", body)
	}
	res = call(t, s, "read_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "attachmentId": atts[1].ID})
	var gotImage bool
	for _, c := range res.Content {
		if _, ok := c.(*mcp.ImageContent); ok {
			gotImage = true
		}
	}
	if !gotImage {
		t.Errorf("image attachment should return image content: %+v", res.Content)
	}
	if body := mustOK(t, call(t, s, "read_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "attachmentId": atts[2].ID})); !strings.Contains(body, "metadata is available") {
		t.Errorf("pdf read = %q", body)
	}
	if res := call(t, s, "read_task_attachment", map[string]any{"projectId": "p1", "taskId": "t2", "attachmentId": atts[0].ID}); !res.IsError {
		t.Error("an attachment id from another task must not resolve")
	}

	// Filters over the wire.
	list := func(args map[string]any) []string {
		args["projectId"] = "p1"
		var tasks []model.Task
		if err := json.Unmarshal([]byte(mustOK(t, call(t, s, "list_tasks", args))), &tasks); err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, tk := range tasks {
			ids = append(ids, tk.ID)
		}
		return ids
	}
	if got := strings.Join(list(map[string]any{"hasAttachments": true}), ","); got != "t1" {
		t.Errorf("hasAttachments = %s", got)
	}
	if got := strings.Join(list(map[string]any{"hasAttachments": false}), ","); got != "t2" {
		t.Errorf("no attachments = %s", got)
	}
	if got := strings.Join(list(map[string]any{"dueFrom": "2026-10-01", "dueTo": "2026-10-31"}), ","); got != "t2" {
		t.Errorf("date range = %s", got)
	}
	if res := call(t, s, "list_tasks", map[string]any{"projectId": "p1", "duePreset": "someday"}); !res.IsError {
		t.Error("unknown preset should be a tool error")
	}

	// Removing drops the record (the app deletes the stored file on save).
	mustOK(t, call(t, s, "remove_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "attachmentId": atts[0].ID}))
	if n := len(host.project.Tasks[0].Attachments); n != 2 {
		t.Errorf("attachments after remove = %d", n)
	}
	if res := call(t, s, "remove_task_attachment", map[string]any{"projectId": "p1", "taskId": "t1", "attachmentId": "nope"}); !res.IsError {
		t.Error("unknown attachment id should error")
	}
}

func TestEndToEndQuarters(t *testing.T) {
	host := &memHost{project: model.Project{ID: "p1", Root: t.TempDir()}}
	dataDir := t.TempDir()
	s := connect(t, host, dataDir)

	fiscal := []any{
		map[string]any{"start": "07-01", "end": "09-30"}, map[string]any{"start": "10-01", "end": "12-31"},
		map[string]any{"start": "01-01", "end": "03-31"}, map[string]any{"start": "04-01", "end": "06-30"},
	}
	mustOK(t, call(t, s, "set_quarters", map[string]any{"quarters": fiscal}))
	if got := daterange.LoadGlobal(dataDir); len(got) != 4 || got[0].Start != "07-01" || got[0].Name != "Q1" {
		t.Errorf("app-wide quarters = %+v", got)
	}
	host.global = daterange.LoadGlobal(dataDir)

	mustOK(t, call(t, s, "set_quarters", map[string]any{"projectId": "p1", "quarters": []any{
		map[string]any{"start": "02-01", "end": "04-30"}, map[string]any{"start": "05-01", "end": "07-31"},
		map[string]any{"start": "08-01", "end": "10-31"}, map[string]any{"start": "11-01", "end": "01-31"},
	}}))
	if host.project.Quarters[0].Start != "02-01" {
		t.Errorf("project override = %+v", host.project.Quarters)
	}
	var q struct {
		Effective []model.QuarterRange `json:"effective"`
	}
	json.Unmarshal([]byte(mustOK(t, call(t, s, "get_quarters", map[string]any{"projectId": "p1"}))), &q)
	if q.Effective[0].Start != "02-01" {
		t.Errorf("project override should win: %+v", q.Effective)
	}

	if res := call(t, s, "set_quarters", map[string]any{"quarters": fiscal[:2]}); !res.IsError {
		t.Error("two quarters must be rejected")
	}
	mustOK(t, call(t, s, "set_quarters", map[string]any{"projectId": "p1", "reset": true}))
	if host.project.Quarters != nil {
		t.Errorf("reset should clear the override: %+v", host.project.Quarters)
	}
	mustOK(t, call(t, s, "set_quarters", map[string]any{"reset": true}))
	if daterange.LoadGlobal(dataDir) != nil {
		t.Error("app-wide reset should remove the file")
	}
}

func TestEndToEndProjectJSONOverMCP(t *testing.T) {
	host := &memHost{project: model.Project{ID: "p1", Name: "Demo", Root: t.TempDir(),
		Processes: []model.Process{{ID: "w", Name: "web", Env: map[string]string{"TOKEN": "s3cret"}}},
		Tasks:     []model.Task{{ID: "t1", Title: "One", Status: "todo"}, {ID: "t2", Title: "Two", Status: "backlog"}}}}
	s := connect(t, host, t.TempDir())

	exported := mustOK(t, call(t, s, "export_project", map[string]any{"projectId": "p1"}))
	if strings.Contains(exported, "s3cret") || !strings.Contains(exported, "<redacted>") {
		t.Fatalf("env not redacted:\n%s", exported)
	}
	var doc map[string]any
	json.Unmarshal([]byte(exported), &doc)
	proj := doc["project"].(map[string]any)
	tasks := proj["tasks"].([]any)
	tasks[0].(map[string]any)["status"] = "done"
	proj["tasks"] = append(tasks, map[string]any{"title": "Three", "status": "inprogress", "dueDate": "2026-12-01"})
	edited, _ := json.Marshal(doc)

	dry := mustOK(t, call(t, s, "import_project", map[string]any{"projectId": "p1", "json": string(edited), "dryRun": true}))
	if !strings.Contains(dry, `"Three"`) || len(host.project.Tasks) != 2 {
		t.Fatalf("dry run must preview without saving: %s", dry)
	}
	mustOK(t, call(t, s, "import_project", map[string]any{"projectId": "p1", "json": string(edited)}))
	if len(host.project.Tasks) != 3 || !host.project.Tasks[0].Done || host.project.Tasks[2].DueDate != "2026-12-01" {
		t.Fatalf("import not applied: %+v", host.project.Tasks)
	}
	if host.project.Processes[0].Env["TOKEN"] != "s3cret" {
		t.Error("redacted export must not overwrite the real environment")
	}
	// Replace mode drops tasks missing from the JSON.
	mustOK(t, call(t, s, "import_project", map[string]any{"projectId": "p1", "json": `{"tasks":[{"id":"t1","title":"One","status":"done"}]}`, "mode": "replace"}))
	if len(host.project.Tasks) != 1 {
		t.Errorf("replace should leave one task, got %d", len(host.project.Tasks))
	}
	if r := call(t, s, "import_project", map[string]any{"projectId": "p1", "json": `{"tasks":[{"title":"x","status":"nope"}]}`}); !r.IsError {
		t.Error("invalid status should be a tool error")
	}
}

func (h *memHost) ListProjects() ([]model.Project, error) { return []model.Project{h.project}, nil }

func TestEndToEndSavedFiltersAndQueryOverMCP(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	host := &memHost{project: model.Project{ID: "p1", Name: "Demo", Tasks: []model.Task{
		{ID: "a", Title: "Due today", Status: "todo", DueDate: today, Priority: "high"},
		{ID: "b", Title: "Blocked thing", Status: "todo", Labels: []string{"blocked"}},
		{ID: "c", Title: "Done", Status: "done", Done: true},
	}, SavedFilters: []model.SavedFilter{{ID: "pf", Name: "Mine", Query: model.TaskQuery{Statuses: []string{"done"}}}}}}
	s := connect(t, host, t.TempDir())

	var lists struct {
		AppWide []model.SavedFilter `json:"appWide"`
		Project []model.SavedFilter `json:"project"`
	}
	json.Unmarshal([]byte(mustOK(t, call(t, s, "list_saved_filters", map[string]any{"projectId": "p1"}))), &lists)
	if len(lists.AppWide) < 10 || len(lists.Project) != 1 {
		t.Fatalf("lists = %d app, %d project", len(lists.AppWide), len(lists.Project))
	}
	run := func(id string) string {
		var out struct {
			Tasks []model.Task `json:"tasks"`
		}
		json.Unmarshal([]byte(mustOK(t, call(t, s, "run_saved_filter", map[string]any{"projectId": "p1", "filterId": id}))), &out)
		var ids []string
		for _, x := range out.Tasks {
			ids = append(ids, x.ID)
		}
		return strings.Join(ids, ",")
	}
	if got := run("default-due-today"); got != "a" {
		t.Errorf("due today = %q", got)
	}
	if got := run("default-blocked"); got != "b" {
		t.Errorf("blocked = %q", got)
	}
	if got := run("pf"); got != "c" {
		t.Errorf("project filter = %q", got)
	}
	if r := call(t, s, "run_saved_filter", map[string]any{"projectId": "p1", "filterId": "nope"}); !r.IsError {
		t.Error("unknown filter should error")
	}
	var hits []map[string]any
	json.Unmarshal([]byte(mustOK(t, call(t, s, "query_tasks", map[string]any{"query": map[string]any{"priorities": []string{"high"}}}))), &hits)
	if len(hits) != 1 || hits[0]["projectName"] != "Demo" {
		t.Errorf("query_tasks = %+v", hits)
	}
}

type textHost struct{ *memHost }

func (textHost) AttachmentText() search.TextSource {
	return func(_, _ string, a model.Attachment) (string, string) {
		if a.ID == "img" {
			return "Invoice 4471 on the whiteboard", search.TextReady
		}
		return "", search.TextNone
	}
}

func TestEndToEndSearchOverMCP(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	host := textHost{&memHost{project: model.Project{ID: "p1", Name: "Demo", Tasks: []model.Task{
		{ID: "a", Title: "Fix login bug", Status: "todo", DueDate: today, Assignee: "Sam"},
		{ID: "b", Title: "Release notes", Status: "todo", Attachments: []model.Attachment{{ID: "img", Name: "board.png", Mime: "image/png"}}},
		{ID: "c", Title: "Old login", Status: "done", Done: true},
	}}}}
	s := connect(t, host, t.TempDir())
	search := func(args map[string]any) string {
		var out struct {
			Results []struct{ Kind, Title, TaskID string } `json:"results"`
			Notes   []string                               `json:"notes"`
		}
		json.Unmarshal([]byte(mustOK(t, call(t, s, "search", args))), &out)
		var got []string
		for _, r := range out.Results {
			got = append(got, r.Kind+":"+r.Title)
		}
		return strings.Join(got, "|")
	}
	if got := search(map[string]any{"query": "login"}); got != "task:Fix login bug|task:Old login" {
		t.Errorf("login = %s", got)
	}
	if got := search(map[string]any{"query": "login", "hideDone": true}); got != "task:Fix login bug" {
		t.Errorf("hideDone = %s", got)
	}
	if got := search(map[string]any{"query": "today @sam"}); got != "task:Fix login bug" {
		t.Errorf("date + assignee = %s", got)
	}
	if got := search(map[string]any{"query": "invoice 4471"}); got != "file:board.png" {
		t.Errorf("OCR text = %s", got)
	}
	if got := search(map[string]any{"query": "nothing-matches-this"}); got != "" {
		t.Errorf("no match = %s", got)
	}
}
