package model

// Process is a runnable subprocess inside a project,
// e.g. "frontend" (Next.js) or "backend" (Go Fiber).
type Process struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Dir     string            `json:"dir"`     // working directory
	Command string            `json:"command"` // e.g. "npm run dev"
	Env     map[string]string `json:"env"`     // extra env vars
}

// Subtask is a small checklist item inside a Task.
type Subtask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Task is one feature/work item in a project's task tracker.
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Done        bool      `json:"done"`               // kept for backward compat
	Status      string    `json:"status,omitempty"`   // kanban column: todo | inprogress | done
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority,omitempty"` // low | medium | high
	Labels      []string  `json:"labels,omitempty"`
	Subtasks    []Subtask `json:"subtasks,omitempty"`
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
	TasksEnabled bool      `json:"tasksEnabled"` // project management feature toggle
	LastUsedAt   int64     `json:"lastUsedAt,omitempty"`
	UseCount     int       `json:"useCount,omitempty"`
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
