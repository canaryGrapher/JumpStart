package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"devdeck/internal/ghsync"
	"devdeck/internal/github"
	"devdeck/internal/model"
)

// syncTimeout bounds one reconcile pass. A board with hundreds of rows
// pages through several requests, so this is generous.
const syncTimeout = 3 * time.Minute

// GitHubLinkProject binds a JumpStart project to a Projects v2 board and
// runs the first reconcile immediately, so the board is populated before
// the user looks away.
func (a *App) GitHubLinkProject(projectID, boardID, repo string, createAsIssue bool) (*model.GitHubSync, error) {
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return nil, fmt.Errorf("project not found")
	}

	cfg := projects[idx].GitHub
	if cfg == nil {
		cfg = &model.GitHubSync{}
	}
	cfg.Enabled = true
	cfg.ProjectID = boardID
	cfg.CreateAsIssue = createAsIssue

	ctx, cancel := context.WithTimeout(a.ctx, syncTimeout)
	defer cancel()

	if _, err := ghsync.NewEngine(client).EnsureStatusMapping(ctx, cfg); err != nil {
		return nil, err
	}
	// Set when the board itself is linked (JumpStart sync works either
	// way) but GitHub's own repository-side link failed; surfaced to
	// the caller as a soft error below, alongside the cfg that already
	// saved successfully, rather than as a hard failure.
	var repoLinkErr error
	if repo != "" {
		r, rerr := client.GetRepository(ctx, repo)
		if rerr != nil {
			return nil, rerr
		}
		cfg.Repo = r.FullName
		cfg.RepoID = r.ID
		// This is what makes the board show up under the repo's own
		// "Projects" tab on github.com. It's separate from the RepoID
		// link above, which only tells JumpStart's local poller what to
		// read/write; that sync works regardless of whether this
		// GitHub-side attach succeeds, so a failure here (e.g. the token
		// lacks admin on the repo) shouldn't block linking.
		if lerr := client.LinkRepositoryToProject(ctx, boardID, r.ID); lerr != nil {
			repoLinkErr = fmt.Errorf("linked for sync, but could not attach the board to %s on GitHub: %w", r.FullName, lerr)
		}
	}

	projects[idx].GitHub = cfg
	if err := a.store.Save(projects); err != nil {
		return nil, err
	}

	if _, err := a.runSync(projectID, true); err != nil {
		cfg.LastSyncError = err.Error()
	} else if repoLinkErr != nil {
		// The sync itself is healthy; only note the cosmetic GitHub-side
		// attach failure, and do it as data on the returned cfg (not a
		// rejected promise) so the frontend still applies the successful
		// link instead of discarding it.
		cfg.LastSyncError = repoLinkErr.Error()
	}
	return cfg, nil
}

// GitHubUnlinkProject stops syncing a project and clears the per-task
// links, leaving every card in place locally.
func (a *App) GitHubUnlinkProject(projectID string) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return fmt.Errorf("project not found")
	}
	projects[idx].GitHub = nil
	for i := range projects[idx].Tasks {
		projects[idx].Tasks[i].GitHub = nil
	}
	if a.gh().scheduler.Current() == projectID {
		a.gh().scheduler.Stop()
	}
	return a.store.Save(projects)
}

// GitHubUpdateSync saves changed sync settings, such as a corrected
// column mapping or a different direction.
func (a *App) GitHubUpdateSync(projectID string, cfg model.GitHubSync) error {
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return fmt.Errorf("project not found")
	}
	projects[idx].GitHub = &cfg
	return a.store.Save(projects)
}

// GitHubGetSync returns a project's sync settings, or nil when it is not
// linked to a board.
func (a *App) GitHubGetSync(projectID string) (*model.GitHubSync, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return nil, err
	}
	return proj.GitHub, nil
}

// GitHubListAssignableUsers returns collaborators who can be assigned on
// issues in the project's linked repository. Empty when unlinked or no
// repo is configured.
func (a *App) GitHubListAssignableUsers(projectID string) ([]github.User, error) {
	proj, err := a.ghProject(projectID)
	if err != nil {
		return nil, err
	}
	if proj.GitHub == nil || proj.GitHub.Repo == "" {
		return nil, nil
	}
	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return client.ListAssignableUsers(ctx, proj.GitHub.Repo)
}

// GitHubSyncNow runs one reconcile pass on demand, for the Sync button.
func (a *App) GitHubSyncNow(projectID string) (*ghsync.Result, error) {
	return a.runSync(projectID, true)
}

// GitHubWatch starts the polling loop for a project's board, and stops
// whichever board was polling before. An empty id stops polling.
func (a *App) GitHubWatch(projectID string) error {
	if projectID == "" {
		a.gh().scheduler.Stop()
		return nil
	}
	proj, err := a.ghProject(projectID)
	if err != nil {
		return err
	}
	if proj.GitHub == nil || !proj.GitHub.Enabled {
		a.gh().scheduler.Stop()
		return nil
	}
	interval := time.Duration(proj.GitHub.PollSeconds) * time.Second
	a.gh().scheduler.Watch(projectID, interval)
	return nil
}

// GitHubSetFocused switches the poll cadence between the fast rate used
// while the board is on screen and the slow background rate.
func (a *App) GitHubSetFocused(focused bool) {
	a.gh().scheduler.SetFocused(focused)
}

// GitHubSetFieldValue writes one Projects v2 field on one task and
// persists the new value locally, so a field edit reaches GitHub without
// waiting for the next pass.
func (a *App) GitHubSetFieldValue(projectID, taskID string, value model.FieldValue) error {
	client, err := a.ghClient()
	if err != nil {
		return err
	}
	projects, err := a.store.Load()
	if err != nil {
		return err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return fmt.Errorf("project not found")
	}
	cfg := projects[idx].GitHub
	if cfg == nil || !cfg.Enabled {
		return errors.New("this project is not linked to a GitHub board")
	}

	tIdx := -1
	for i := range projects[idx].Tasks {
		if projects[idx].Tasks[i].ID == taskID {
			tIdx = i
			break
		}
	}
	if tIdx < 0 {
		return fmt.Errorf("task not found")
	}
	task := &projects[idx].Tasks[tIdx]
	if task.GitHub == nil || task.GitHub.ItemID == "" {
		return errors.New("this task has not synced to GitHub yet")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()

	board, err := client.GetProject(ctx, cfg.ProjectID)
	if err != nil {
		return err
	}
	var field github.Field
	for _, f := range board.Fields {
		if f.ID == value.FieldID {
			field = f
			break
		}
	}
	if field.ID == "" {
		return fmt.Errorf("field not found on the board")
	}

	err = client.SetValue(ctx, cfg.ProjectID, task.GitHub.ItemID, field, github.ItemFieldValue{
		FieldID:     value.FieldID,
		DataType:    field.DataType,
		Text:        value.Text,
		Number:      value.Number,
		Date:        value.Date,
		OptionID:    value.OptionID,
		IterationID: value.Iteration,
	})
	if err != nil {
		return err
	}

	if task.Fields == nil {
		task.Fields = map[string]model.FieldValue{}
	}
	value.Name = field.Name
	value.DataType = field.DataType
	task.Fields[value.FieldID] = value
	task.UpdatedAt = time.Now().UnixMilli()
	return a.store.Save(projects)
}

// GitHubResolveConflict clears the conflict badge on a task, and when
// keepLocal is true marks the task for a push so the local copy is the
// one that survives. Last-write-wins already picked a side during the
// pass; this is how the user overrides that choice.
//
// Returns the updated GitHub link so the open task modal can drop the
// conflict UI without waiting for another sync event.
func (a *App) GitHubResolveConflict(projectID, taskID string, keepLocal bool) (*model.GitHubLink, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return nil, fmt.Errorf("project not found")
	}
	for i := range projects[idx].Tasks {
		t := &projects[idx].Tasks[i]
		if t.ID != taskID || t.GitHub == nil {
			continue
		}
		now := time.Now().UnixMilli()
		t.GitHub.Conflict = false
		t.GitHub.SyncedAt = now
		if keepLocal {
			t.GitHub.Pending = true
			t.UpdatedAt = now
		} else {
			// Accept the copy already on the board; advance watermarks so
			// the next poll does not re-raise the same divergence.
			t.GitHub.Pending = false
			if t.GitHub.RemoteUpdatedAt < now {
				t.GitHub.RemoteUpdatedAt = now
			}
		}
		if err := a.store.Save(projects); err != nil {
			return nil, err
		}
		link := *t.GitHub
		// Surface the cleared badge immediately so the open modal does
		// not keep showing Conflict while a follow-up push sync runs.
		a.emit("github:task:link", map[string]any{
			"projectId": projectID,
			"taskId":    taskID,
			"github":    link,
		})
		if keepLocal {
			go func() { _, _ = a.runSync(projectID, false) }()
		}
		return &link, nil
	}
	return nil, fmt.Errorf("task not found")
}

// runSync reconciles one project and persists the result. Passes never
// overlap: a manual Sync during a poll tick is dropped rather than run
// twice against the same board.
func (a *App) runSync(projectID string, manual bool) (*ghsync.Result, error) {
	state := a.gh()
	if _, busy := state.syncing.LoadOrStore(projectID, true); busy {
		return nil, errors.New("a sync is already running for this project")
	}
	defer state.syncing.Delete(projectID)

	client, err := a.ghClient()
	if err != nil {
		return nil, err
	}

	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	idx := indexOfProject(projects, projectID)
	if idx < 0 {
		return nil, fmt.Errorf("project not found")
	}
	cfg := projects[idx].GitHub
	if cfg == nil || !cfg.Enabled {
		return nil, errors.New("this project is not linked to a GitHub board")
	}

	a.emit("github:sync:start", projectID)

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	baseline := make(map[string]bool, len(projects[idx].Tasks))
	for _, t := range projects[idx].Tasks {
		baseline[t.ID] = true
	}

	engine := ghsync.NewEngine(client).WithProgress(func(done, total int) {
		a.emit("github:sync:progress", map[string]any{
			"projectId": projectID,
			"done":      done,
			"total":     total,
		})
	})
	tasks, res, err := engine.Sync(ctx, projects[idx].Tasks, cfg)
	if err != nil {
		cfg.LastSyncError = err.Error()
		projects[idx].GitHub = cfg
		_ = a.store.Save(projects)
		a.emit("github:sync:error", map[string]any{"projectId": projectID, "error": err.Error()})
		return nil, err
	}

	// Reload before writing: a task edit may have landed while the pass
	// was in flight, and clobbering it would lose the user's typing.
	var flushDeletes bool
	fresh, lerr := a.store.Load()
	if lerr == nil {
		if fidx := indexOfProject(fresh, projectID); fidx >= 0 {
			merged, deletedItems := mergeConcurrent(fresh[fidx].Tasks, tasks, res.At, baseline)
			fresh[fidx].Tasks = merged
			cfg.LastSyncAt = res.At
			cfg.LastSyncError = ""
			if len(deletedItems) > 0 && cfg.Direction != "pull" {
				cfg.PendingDeletes = ghsync.MergePendingDeletes(cfg.PendingDeletes, deletedItems)
				flushDeletes = true
			}
			fresh[fidx].GitHub = cfg
			if serr := a.store.Save(fresh); serr != nil {
				return nil, serr
			}
			projects = fresh
			idx = fidx
		}
	}

	a.emit("github:sync:done", map[string]any{
		"projectId": projectID,
		"result":    res,
		"tasks":     projects[idx].Tasks,
		"manual":    manual,
	})
	if flushDeletes {
		// The pass that observed the in-flight delete has unlocked by the
		// time this goroutine runs; it flushes PendingDeletes to GitHub.
		go func() { _, _ = a.runSync(projectID, false) }()
	}
	return res, nil
}

// mergeConcurrent keeps a local edit made while a sync was in flight.
// A task the user touched after the pass started wins over the synced
// copy and is left pending, so the next pass pushes it.
//
// baseline lists task IDs that existed when the pass started. A baseline
// task missing from current was deleted by the user during the pass: it
// stays deleted, and any linked board row id is returned for PendingDeletes.
func mergeConcurrent(current, synced []model.Task, startedAt int64, baseline map[string]bool) ([]model.Task, []string) {
	byID := map[string]model.Task{}
	for _, t := range current {
		byID[t.ID] = t
	}
	out := make([]model.Task, 0, len(synced))
	var pendingDeletes []string
	for _, t := range synced {
		if _, ok := byID[t.ID]; !ok && baseline[t.ID] {
			if t.GitHub != nil && t.GitHub.ItemID != "" {
				pendingDeletes = append(pendingDeletes, t.GitHub.ItemID)
			}
			continue
		}
		if live, ok := byID[t.ID]; ok && live.UpdatedAt > startedAt {
			if t.GitHub != nil {
				link := *t.GitHub
				link.Pending = true
				live.GitHub = &link
			}
			out = append(out, live)
			continue
		}
		out = append(out, t)
	}
	return out, pendingDeletes
}

func indexOfProject(projects []model.Project, id string) int {
	for i := range projects {
		if projects[i].ID == id {
			return i
		}
	}
	return -1
}
