package model

// Process is a runnable subprocess inside a project,
// e.g. "frontend" (Next.js) or "backend" (Go Fiber).
type Process struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Dir         string            `json:"dir"`     // working directory
	Command     string            `json:"command"` // e.g. "npm run dev"
	Env         map[string]string `json:"env"`     // extra env vars
	TestCommand string            `json:"testCommand,omitempty"` // per-process override for RunTests
	Scripts     []Script          `json:"scripts,omitempty"`     // one-off custom commands, e.g. "migrate"
}

// Script is a one-off command attached to a Process, e.g. "migrate"
// running "go run . --migrate". Unlike the process command it is not
// long-lived: it runs, prints output, and exits.
type Script struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`            // button label, e.g. "Migrate"
	Command string            `json:"command"`         // e.g. "go run . --migrate"
	Dir     string            `json:"dir,omitempty"`   // defaults to the process dir
	Env     map[string]string `json:"env,omitempty"`   // merged over the process env
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

// Task is one work item in a project's board. It doubles as a user
// story: a story is a Task with Type == "story" whose child tasks
// point back to it via ParentID (a two-level story→task hierarchy).
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Done        bool      `json:"done"`             // kept for backward compat
	Status      string    `json:"status,omitempty"` // kanban column: backlog | todo | inprogress | done
	Type        string    `json:"type,omitempty"`   // story | task | bug (default task)
	ParentID    string    `json:"parentId,omitempty"`
	SprintID    string    `json:"sprintId,omitempty"` // empty = backlog
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority,omitempty"` // low | medium | high
	Labels      []string  `json:"labels,omitempty"`
	Subtasks    []Subtask `json:"subtasks,omitempty"`
	Acceptance  []Subtask `json:"acceptance,omitempty"` // acceptance criteria (stories)
	StoryPoints int       `json:"storyPoints,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	CreatedAt   int64     `json:"createdAt"` // unix ms
	UpdatedAt   int64     `json:"updatedAt,omitempty"`
}

// Project groups processes, e.g. "Project Alpha".
type Project struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Root         string    `json:"root"`
	Processes    []Process `json:"processes"`
	Tasks        []Task    `json:"tasks,omitempty"`
	Sprints      []Sprint  `json:"sprints,omitempty"`
	TasksEnabled bool      `json:"tasksEnabled"` // project management feature toggle
	LastUsedAt   int64     `json:"lastUsedAt,omitempty"`
	UseCount     int       `json:"useCount,omitempty"`
	Description  string    `json:"description,omitempty"`
	TestCommand  string    `json:"testCommand,omitempty"` // per-project override for RunTests
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
