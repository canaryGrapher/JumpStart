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
	Unlinked  int      `json:"unlinked"`  // push-only: link cleared when row gone
	Deleted   int      `json:"deleted"`   // local tasks dropped or remote rows removed
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
	client   syncClient
	progress ProgressFunc

	// assigneeIDs caches login→node-id lookups for one Sync pass so a
	// board with many cards does not re-list assignableUsers each push.
	assigneeIDs  map[string]string
	assigneeErr  error
	assigneeRepo string

	// labelIDs caches label-name→node-id lookups the same way.
	labelIDs  map[string]string
	labelErr  error
	labelRepo string
}

// NewEngine returns an Engine bound to an authenticated client.
func NewEngine(c syncClient) *Engine { return &Engine{client: c} }

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

	now := time.Now().UnixMilli()
	res := &Result{At: now}

	pushAllowed := cfg.Direction != "pull"
	pullAllowed := cfg.Direction != "push"

	pendingSet := map[string]bool{}
	for _, id := range cfg.PendingDeletes {
		if id != "" {
			pendingSet[id] = true
		}
	}

	// Flush local deletes to the board before reading items, so a just-
	// deleted card cannot be pulled back in as a "new" task this pass.
	if pushAllowed && len(cfg.PendingDeletes) > 0 {
		remaining := make([]string, 0, len(cfg.PendingDeletes))
		for _, id := range cfg.PendingDeletes {
			if id == "" {
				continue
			}
			if err := e.client.DeleteItem(ctx, cfg.ProjectID, id); err != nil {
				remaining = append(remaining, id)
				res.Errors = append(res.Errors, fmt.Sprintf("delete %s: %v", id, err))
				continue
			}
			delete(pendingSet, id)
			res.Deleted++
			res.Changed = true
		}
		cfg.PendingDeletes = remaining
		for _, id := range remaining {
			pendingSet[id] = true
		}
	}

	board, err := e.client.GetProject(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board: %w", err)
	}
	// Pick up Status options for columns added after the board was linked
	// (Testing is the common case) without rewriting hand-tuned mappings.
	if f, ok := FindStatusField(board.Fields); ok {
		if cfg.StatusFieldID == "" {
			cfg.StatusFieldID = f.ID
			res.Changed = true
		}
		if cfg.StatusFieldID == f.ID {
			next := FillMissingStatusMap(f, cfg.StatusMap)
			if !sameStatusMap(cfg.StatusMap, next) {
				cfg.StatusMap = next
				res.Changed = true
			}
		}
	}
	items, err := e.client.ListItems(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board items: %w", err)
	}

	remote := make(map[string]github.Item, len(items))
	for _, it := range items {
		remote[it.ID] = it
	}

	out := make([]model.Task, 0, len(tasks)+len(items))
	seen := map[string]bool{}
	dropped := map[string]bool{}

	total := len(tasks)
	if e.progress != nil && total > 0 {
		e.progress(0, total)
	}

	for i := range tasks {
		t := tasks[i]
		link := t.GitHub

		// A task linked to a row that no longer exists: when we pull,
		// the remote delete is authoritative and the local card goes
		// away. Push-only keeps the work and clears the stale link so a
		// later pass can upload it again.
		if link != nil && link.ItemID != "" {
			item, ok := remote[link.ItemID]
			if !ok {
				if pullAllowed {
					dropped[t.ID] = true
					res.Deleted++
					res.Changed = true
				} else {
					t.GitHub = nil
					res.Unlinked++
					res.Changed = true
					out = append(out, t)
				}
				if e.progress != nil {
					e.progress(i+1, total)
				}
				continue
			}
			seen[link.ItemID] = true

			remoteUpdated := parseTime(item.UpdatedAt)
			dir := decide(t, remoteUpdated)
			// GitHub is SSOT. When both sides moved, only raise a conflict
			// when field values actually diverge — a post-push echo with
			// matching content stamps quietly. Real divergence holds the
			// local copy and records per-field reasons for the UI.
			if dir == sideBoth {
				fields := DiffTask(t, item, cfg)
				if len(fields) == 0 {
					clearConflict(&t, remoteUpdated, now)
					dir = sideNone
					res.Changed = true
				} else {
					markConflict(&t, fields)
					res.Conflicts++
					res.Changed = true
					dir = sideConflict
				}
			}

			switch {
			case dir == sideRemote && pullAllowed:
				if applyRemote(&t, item, cfg) {
					res.Pulled++
					res.Changed = true
				}
				clearConflict(&t, remoteUpdated, now)
			case dir == sideLocal && pushAllowed:
				if err := e.pushTask(ctx, &t, board, cfg); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", t.Title, err))
					t.GitHub.Pending = true
					res.Changed = true
					break
				}
				res.Pushed++
				res.Changed = true
				stampAfterPush(&t, time.Now().UnixMilli())
			case dir == sideConflict:
				// Hold local; user Accepts GitHub or Overwrites.
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

	// Drop children whose parent story was removed by a remote delete.
	if len(dropped) > 0 {
		kept := out[:0]
		for _, t := range out {
			if t.ParentID != "" && dropped[t.ParentID] {
				dropped[t.ID] = true
				res.Deleted++
				res.Changed = true
				continue
			}
			kept = append(kept, t)
		}
		out = kept
	}

	// Board rows JumpStart has never seen become new local cards — except
	// rows we are still trying to delete, which must not come back.
	if pullAllowed {
		for _, it := range items {
			if seen[it.ID] || pendingSet[it.ID] {
				continue
			}
			out = append(out, taskFromItem(it, cfg, now))
			res.Created++
			res.Changed = true
		}
	}

	return out, res, nil
}

// needsPendingFlush reports whether a task should be included in a
// pending-only batch push (local edit waiting for GitHub, force overwrite,
// or never synced).
func needsPendingFlush(t model.Task) bool {
	link := t.GitHub
	if link == nil || link.ItemID == "" || link.SyncedAt == 0 {
		return true
	}
	if link.Conflict && !link.ForcePush {
		// Waiting on the user; do not push until Overwrite GitHub.
		return false
	}
	return link.Pending || link.ForcePush || t.UpdatedAt > link.SyncedAt
}

// SyncPending pushes only local pending / never-synced tasks (and flushes
// PendingDeletes). It still loads the board once for field ids and runs
// content-diff when a pending task's remote row also moved. Quiet cards
// and remote-only pulls are left for the full Sync poll.
func (e *Engine) SyncPending(ctx context.Context, tasks []model.Task, cfg *model.GitHubSync) ([]model.Task, *Result, error) {
	if cfg == nil || !cfg.Enabled || cfg.ProjectID == "" {
		return tasks, &Result{At: time.Now().UnixMilli()}, nil
	}

	now := time.Now().UnixMilli()
	res := &Result{At: now}
	pushAllowed := cfg.Direction != "pull"
	if !pushAllowed {
		return tasks, res, nil
	}

	pendingSet := map[string]bool{}
	for _, id := range cfg.PendingDeletes {
		if id != "" {
			pendingSet[id] = true
		}
	}
	if len(cfg.PendingDeletes) > 0 {
		remaining := make([]string, 0, len(cfg.PendingDeletes))
		for _, id := range cfg.PendingDeletes {
			if id == "" {
				continue
			}
			if err := e.client.DeleteItem(ctx, cfg.ProjectID, id); err != nil {
				remaining = append(remaining, id)
				res.Errors = append(res.Errors, fmt.Sprintf("delete %s: %v", id, err))
				continue
			}
			delete(pendingSet, id)
			res.Deleted++
			res.Changed = true
		}
		cfg.PendingDeletes = remaining
	}

	var toFlush []int
	for i := range tasks {
		if needsPendingFlush(tasks[i]) {
			toFlush = append(toFlush, i)
		}
	}
	if len(toFlush) == 0 && !res.Changed {
		return tasks, res, nil
	}

	board, err := e.client.GetProject(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board: %w", err)
	}
	if f, ok := FindStatusField(board.Fields); ok {
		if cfg.StatusFieldID == "" {
			cfg.StatusFieldID = f.ID
			res.Changed = true
		}
		if cfg.StatusFieldID == f.ID {
			next := FillMissingStatusMap(f, cfg.StatusMap)
			if !sameStatusMap(cfg.StatusMap, next) {
				cfg.StatusMap = next
				res.Changed = true
			}
		}
	}

	items, err := e.client.ListItems(ctx, cfg.ProjectID)
	if err != nil {
		return tasks, nil, fmt.Errorf("reading board items: %w", err)
	}
	remote := make(map[string]github.Item, len(items))
	for _, it := range items {
		remote[it.ID] = it
	}

	out := make([]model.Task, len(tasks))
	copy(out, tasks)

	total := len(toFlush)
	if e.progress != nil && total > 0 {
		e.progress(0, total)
	}

	for n, i := range toFlush {
		t := out[i]
		link := t.GitHub

		if link != nil && link.ItemID != "" {
			item, ok := remote[link.ItemID]
			if !ok {
				// Row gone: clear stale link so a later full sync can re-upload.
				t.GitHub = nil
				res.Unlinked++
				res.Changed = true
				out[i] = t
				if e.progress != nil {
					e.progress(n+1, total)
				}
				continue
			}
			remoteUpdated := parseTime(item.UpdatedAt)
			dir := decide(t, remoteUpdated)
			if dir == sideBoth {
				fields := DiffTask(t, item, cfg)
				if len(fields) == 0 {
					clearConflict(&t, remoteUpdated, now)
					out[i] = t
					res.Changed = true
					if e.progress != nil {
						e.progress(n+1, total)
					}
					continue
				}
				if !link.ForcePush {
					markConflict(&t, fields)
					res.Conflicts++
					res.Changed = true
					out[i] = t
					if e.progress != nil {
						e.progress(n+1, total)
					}
					continue
				}
				// Overwrite GitHub: push through divergence.
			}
			if err := e.pushTask(ctx, &t, board, cfg); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", t.Title, err))
				if t.GitHub != nil {
					t.GitHub.Pending = true
				}
				res.Changed = true
				out[i] = t
				if e.progress != nil {
					e.progress(n+1, total)
				}
				continue
			}
			res.Pushed++
			res.Changed = true
			stampAfterPush(&t, time.Now().UnixMilli())
			out[i] = t
			if e.progress != nil {
				e.progress(n+1, total)
			}
			continue
		}

		// Never-synced: create on the board.
		if err := e.pushTask(ctx, &t, board, cfg); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", t.Title, err))
			out[i] = t
			if e.progress != nil {
				e.progress(n+1, total)
			}
			continue
		}
		clearConflict(&t, 0, time.Now().UnixMilli())
		res.Uploaded++
		res.Changed = true
		out[i] = t
		if e.progress != nil {
			e.progress(n+1, total)
		}
	}

	return out, res, nil
}

// EnsureStatusMapping fills in the Status field and column mapping when a
// project is linked, so the first sync already knows which board column
// each Kanban column corresponds to. It also fills any local columns that
// were added after the link (e.g. Testing) when a matching Status option
// already exists on the board.
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

	if f, ok := FindStatusField(board.Fields); ok {
		if cfg.StatusFieldID == "" {
			cfg.StatusFieldID = f.ID
		}
		if cfg.StatusFieldID == f.ID {
			cfg.StatusMap = FillMissingStatusMap(f, cfg.StatusMap)
		}
	}
	if cfg.Direction == "" {
		cfg.Direction = "both"
	}
	return board, nil
}
