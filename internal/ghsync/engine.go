package ghsync

import (
	"context"
	"fmt"
	"time"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

// Result reports what one reconcile pass did, so the UI can show a
// meaningful status line rather than a spinner that never explains
// itself.
type Result struct {
	Pulled    int      `json:"pulled"`
	Pushed    int      `json:"pushed"`
	Created   int      `json:"created"`   // new local tasks from the board
	Uploaded  int      `json:"uploaded"`  // new board rows from local tasks
	Conflicts int      `json:"conflicts"` // tasks where both sides changed
	Unlinked  int      `json:"unlinked"`  // tasks whose board row disappeared
	Errors    []string `json:"errors,omitempty"`
	At        int64    `json:"at"`
	Changed   bool     `json:"changed"` // whether tasks need persisting
}

// ProgressFunc reports how far a Sync pass has got through the local
// task list. done is 1-based (the task just finished); total is the
// number of local tasks in this pass. Nil is fine.
type ProgressFunc func(done, total int)

// Engine reconciles one project against one board. It is stateless
// between passes: everything it needs to decide a direction lives on the
// task's GitHubLink watermark.
type Engine struct {
	client   *github.Client
	progress ProgressFunc
}

// NewEngine returns an Engine bound to an authenticated client.
func NewEngine(c *github.Client) *Engine { return &Engine{client: c} }

// WithProgress returns a shallow copy that reports per-task progress
// during Sync. Useful for the initial link pass, where every local card
// is uploaded one GraphQL call at a time.
func (e *Engine) WithProgress(fn ProgressFunc) *Engine {
	cp := *e
	cp.progress = fn
	return &cp
}

// Sync runs one full reconcile pass and returns the tasks as they should
// now be persisted. The input slice is never mutated.
func (e *Engine) Sync(ctx context.Context, tasks []model.Task, cfg *model.GitHubSync) ([]model.Task, *Result, error) {
	if cfg == nil || !cfg.Enabled || cfg.ProjectID == "" {
		return tasks, &Result{At: time.Now().UnixMilli()}, nil
	}

	board, err := e.client.GetProject(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board: %w", err)
	}
	items, err := e.client.ListItems(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board items: %w", err)
	}

	remote := make(map[string]github.Item, len(items))
	for _, it := range items {
		remote[it.ID] = it
	}

	now := time.Now().UnixMilli()
	res := &Result{At: now}
	out := make([]model.Task, 0, len(tasks)+len(items))
	seen := map[string]bool{}

	pushAllowed := cfg.Direction != "pull"
	pullAllowed := cfg.Direction != "push"

	total := len(tasks)
	if e.progress != nil && total > 0 {
		e.progress(0, total)
	}

	for i := range tasks {
		t := tasks[i]
		link := t.GitHub

		// A task that was linked to a row that no longer exists gets its
		// link dropped rather than deleted: the work is still real
		// locally, and a re-sync can adopt it again.
		if link != nil && link.ItemID != "" {
			item, ok := remote[link.ItemID]
			if !ok {
				t.GitHub = nil
				res.Unlinked++
				res.Changed = true
				out = append(out, t)
				if e.progress != nil {
					e.progress(i+1, total)
				}
				continue
			}
			seen[link.ItemID] = true

			remoteUpdated := parseTime(item.UpdatedAt)
			dir := decide(t, remoteUpdated)
			if dir == sideBoth {
				res.Conflicts++
				winner := resolve(t, remoteUpdated)
				t.GitHub.Conflict = true
				dir = winner
			}

			switch {
			case dir == sideRemote && pullAllowed:
				if applyRemote(&t, item, cfg) {
					res.Pulled++
					res.Changed = true
				}
				conflicted := t.GitHub != nil && t.GitHub.Conflict
				clearConflict(&t, remoteUpdated, now)
				t.GitHub.Conflict = conflicted
			case dir == sideLocal && pushAllowed:
				if err := e.pushTask(ctx, &t, board, cfg); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", t.Title, err))
					t.GitHub.Pending = true
					res.Changed = true
					break
				}
				res.Pushed++
				res.Changed = true
				conflicted := t.GitHub != nil && t.GitHub.Conflict
				clearConflict(&t, remoteUpdated, now)
				t.GitHub.Conflict = conflicted
			}
			out = append(out, t)
			if e.progress != nil {
				e.progress(i+1, total)
			}
			continue
		}

		// Never-synced task: create it on the board.
		if pushAllowed {
			if err := e.pushTask(ctx, &t, board, cfg); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", t.Title, err))
				out = append(out, t)
				if e.progress != nil {
					e.progress(i+1, total)
				}
				continue
			}
			if t.GitHub != nil {
				seen[t.GitHub.ItemID] = true
			}
			clearConflict(&t, 0, now)
			res.Uploaded++
			res.Changed = true
		}
		out = append(out, t)
		if e.progress != nil {
			e.progress(i+1, total)
		}
	}

	// Board rows JumpStart has never seen become new local cards.
	if pullAllowed {
		for _, it := range items {
			if seen[it.ID] {
				continue
			}
			out = append(out, taskFromItem(it, cfg, now))
			res.Created++
			res.Changed = true
		}
	}

	return out, res, nil
}

// EnsureStatusMapping fills in the Status field and column mapping when a
// project is linked, so the first sync already knows which board column
// each Kanban column corresponds to.
func (e *Engine) EnsureStatusMapping(ctx context.Context, cfg *model.GitHubSync) (*github.Project, error) {
	board, err := e.client.GetProject(ctx, cfg.ProjectID)
	if err != nil {
		return nil, err
	}
	cfg.ProjectTitle = board.Title
	cfg.ProjectURL = board.URL
	cfg.ProjectNumber = board.Number
	cfg.Owner = board.Owner
	cfg.OwnerType = board.OwnerType

	if cfg.StatusFieldID == "" || len(cfg.StatusMap) == 0 {
		if f, ok := FindStatusField(board.Fields); ok {
			cfg.StatusFieldID = f.ID
			if len(cfg.StatusMap) == 0 {
				cfg.StatusMap = BuildStatusMap(f)
			}
		}
	}
	if cfg.Direction == "" {
		cfg.Direction = "both"
	}
	return board, nil
}
