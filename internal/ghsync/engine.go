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
			// Only raise the conflict badge when the remote copy wins
			// last-write-wins. A local edit that is simply newer than
			// GitHub (the common "I edited this story" case) should push
			// quietly — the old behavior flagged every such edit because
			// the post-push GitHub timestamp looked like a remote change.
			// Count only badge-raising cases so the sync bar does not keep
			// showing "N conflicts" after a local-won (or Keep mine) pass.
			if dir == sideBoth {
				winner := resolve(t, remoteUpdated)
				if winner == sideRemote {
					t.GitHub.Conflict = true
					res.Conflicts++
				}
				dir = winner
			}

			switch {
			case dir == sideRemote && pullAllowed:
				if applyRemote(&t, item, cfg) {
					res.Pulled++
					res.Changed = true
				}
				// Keep a remote-won conflict badge so the modal can offer
				// "Keep mine, push it"; clear pending/watermarks otherwise.
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
				// Local won (or was the only editor). Stamp with the time
				// the push finished — pass-start `now` can be minutes old
				// on a large board, which lets GitHub's echo fall outside
				// the grace window and raise a false conflict.
				stampAfterPush(&t, time.Now().UnixMilli())
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
