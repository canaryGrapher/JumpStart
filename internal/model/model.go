package model

// Process is a runnable subprocess inside a project,
// e.g. "frontend" (Next.js) or "backend" (Go Fiber).
type Process struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Dir         string            `json:"dir"`                   // working directory
	Command     string            `json:"command"`               // e.g. "npm run dev"
	Env         map[string]string `json:"env"`                   // extra env vars
	TestCommand string            `json:"testCommand,omitempty"` // per-process override for RunTests
	Scripts     []Script          `json:"scripts,omitempty"`     // one-off custom commands, e.g. "migrate"
}

// Script is a one-off command attached to a Process, e.g. "migrate"
// running "go run . --migrate". Unlike the process command it is not
// long-lived: it runs, prints output, and exits.
type Script struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`             // button label, e.g. "Migrate"
	Command string            `json:"command"`          // e.g. "go run . --migrate"
	Dir     string            `json:"dir,omitempty"`    // defaults to the process dir
	Env     map[string]string `json:"env,omitempty"`    // merged over the process env
	Source  string            `json:"source,omitempty"` // where auto-detect found it, e.g. "package.json"
}

// Subtask is a small checklist item inside a Task.
type Subtask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Sprint is a time-boxed bucket of tasks. Sprints are ordered by
// Order to form the project roadmap, and tasks join one via SprintID.
type Sprint struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Goal      string `json:"goal,omitempty"`
	Status    string `json:"status,omitempty"` // planned | active | completed
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
	Order     int    `json:"order"` // position in the roadmap sequence
	CreatedAt int64  `json:"createdAt"`
}

// BoardColumn is one Kanban column on a project's board. The built-in
// five (backlog → done) are used when Project.Columns is empty; custom
// columns are stored here and mapped to a GitHub Status option via
// GitHubSync.StatusMap.
type BoardColumn struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// Task is one work item in a project's board. It doubles as a user
// story: a story is a Task with Type == "story" whose child tasks
// point back to it via ParentID (a two-level story→task hierarchy).
type Task struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Done        bool         `json:"done"`             // kept for backward compat
	Status      string       `json:"status,omitempty"` // kanban column: backlog | todo | inprogress | testing | done
	Type        string       `json:"type,omitempty"`   // story | task | bug (default task)
	ParentID    string       `json:"parentId,omitempty"`
	SprintID    string       `json:"sprintId,omitempty"` // empty = backlog
	Description string       `json:"description,omitempty"`
	Priority    string       `json:"priority,omitempty"` // low | medium | high
	Labels      []string     `json:"labels,omitempty"`
	Subtasks    []Subtask    `json:"subtasks,omitempty"`
	Acceptance  []Subtask    `json:"acceptance,omitempty"` // acceptance criteria (any type)
	StoryPoints int          `json:"storyPoints,omitempty"`
	Assignee    string       `json:"assignee,omitempty"`
	DueDate     string       `json:"dueDate,omitempty"` // YYYY-MM-DD, local calendar date
	Links       []TaskLink   `json:"links,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	CreatedAt   int64        `json:"createdAt"` // unix ms
	UpdatedAt   int64        `json:"updatedAt,omitempty"`

	// GitHub is the Projects v2 linkage, set once the task has synced.
	GitHub *GitHubLink `json:"github,omitempty"`
	// Fields holds every Projects v2 field value for this task, keyed by
	// field id. Built-in columns (title, status, assignee, labels) stay in
	// the native members above; everything else on the board lands here.
	Fields map[string]FieldValue `json:"fields,omitempty"`
	// Milestone, Repository and the read-only rollups GitHub computes.
	Milestone string   `json:"milestone,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
	LinkedPRs []string `json:"linkedPrs,omitempty"`
	IssueType string   `json:"issueType,omitempty"`
	ParentKey string   `json:"parentKey,omitempty"` // GitHub parent issue ref
}

// TaskLink is a titled hyperlink attached to a task. Only http, https, and
// mailto URLs are accepted; see internal/attachments.NormalizeURL.
type TaskLink struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url"`
}

// Attachment is a file copied into JumpStart's attachment store for a task.
// The bytes live at <data dir>/attachments/<projectId>/<taskId>/<File>; the
// path is always derived from IDs, never taken from a client-supplied string.
type Attachment struct {
	ID      string `json:"id"`
	Name    string `json:"name"` // original file name, for display
	File    string `json:"file"` // stored file name (id + extension)
	Mime    string `json:"mime,omitempty"`
	Size    int64  `json:"size"`
	AddedAt int64  `json:"addedAt"` // unix ms
}

// TaskQuery is a task filter shared by the board, sheet view, saved
// filters, dashboard widgets, MCP and Raycast. Empty fields mean "no
// constraint"; values in a list are ORed and fields are ANDed. Evaluation
// lives in internal/query.
type TaskQuery struct {
	DuePreset  string   `json:"duePreset,omitempty"`
	DueFrom    string   `json:"dueFrom,omitempty"`
	DueTo      string   `json:"dueTo,omitempty"`
	NoDueDate  bool     `json:"noDueDate,omitempty"`
	Overdue    bool     `json:"overdue,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Priorities []string `json:"priorities,omitempty"` // "none" = no priority
	Types      []string `json:"types,omitempty"`      // empty type counts as "task"
	Sprints    []string `json:"sprints,omitempty"`    // "" = backlog
	Assignees  []string `json:"assignees,omitempty"`  // "__none__" = unassigned
	Labels     []string `json:"labels,omitempty"`
	Acceptance string   `json:"acceptance,omitempty"` // "" | "has" | "none"
	Subtasks   string   `json:"subtasks,omitempty"`   // "" | "has" | "none"
	Text       string   `json:"text,omitempty"`       // title/description contains
}

// SavedFilter is a named TaskQuery, stored app-wide or on one project.
type SavedFilter struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     TaskQuery `json:"query"`
	Builtin   bool      `json:"builtin,omitempty"` // one of the seeded defaults
	CreatedAt int64     `json:"createdAt,omitempty"`
}

// QuarterRange is one quarter of the reporting year as a month/day span,
// stored year-agnostic as "MM-DD" so the same config applies every year.
// A quarter may wrap the new year (Start "11-01", End "01-31").
type QuarterRange struct {
	Name  string `json:"name,omitempty"` // "Q1".."Q4"
	Start string `json:"start"`          // MM-DD
	End   string `json:"end"`            // MM-DD
}

// Project groups processes, e.g. "Project Alpha".
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Root      string    `json:"root"`
	Processes []Process `json:"processes"`
	Tasks     []Task    `json:"tasks,omitempty"`
	Sprints   []Sprint  `json:"sprints,omitempty"`
	// Columns is the project's Kanban layout. Empty means the built-in
	// backlog / todo / inprogress / testing / done set.
	Columns      []BoardColumn `json:"columns,omitempty"`
	TasksEnabled bool          `json:"tasksEnabled"`       // project management feature toggle
	Favorite     bool          `json:"favorite,omitempty"` // pinned to the Favorites group in the sidebar
	LastUsedAt   int64         `json:"lastUsedAt,omitempty"`
	UseCount     int           `json:"useCount,omitempty"`
	Description  string        `json:"description,omitempty"`
	Icon         string        `json:"icon,omitempty"`        // data: URI (base64) for a square project icon, shown in the sidebar and header
	TestCommand  string        `json:"testCommand,omitempty"` // per-project override for RunTests
	// SavedFilters are filters saved for this project only (app-wide ones
	// live in <data dir>/filters.json).
	SavedFilters []SavedFilter `json:"savedFilters,omitempty"`
	// Quarters overrides the app-wide quarter dates for due-date filters.
	// Empty means use the global setting (or calendar quarters).
	Quarters []QuarterRange `json:"quarters,omitempty"`
	// GitHub links this project's board to a GitHub Projects v2 board.
	GitHub *GitHubSync `json:"github,omitempty"`
}

// Status is the live state of one process.
type Status struct {
	ProcID    string `json:"procId"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	Ports     []int  `json:"ports"`
	StartedAt int64  `json:"startedAt"` // unix ms, 0 if stopped
	ExitCode  int    `json:"exitCode"`  // -1 while running
}
