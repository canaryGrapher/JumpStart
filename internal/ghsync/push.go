package ghsync

import (
	"context"
	"strings"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

// pushTask writes one local task up to the board: content first, then
// assignees/labels (promoting a draft when needed), then the Status
// column, then any custom field values JumpStart owns.
func (e *Engine) pushTask(ctx context.Context, task *model.Task, board *github.Project, cfg *model.GitHubSync) error {
	link := task.GitHub
	body := ComposeBody(task.Description, task.Acceptance, task.Subtasks)
	if link == nil || link.ItemID == "" {
		if err := e.createRemote(ctx, task, cfg, body); err != nil {
			return err
		}
		return e.pushFields(ctx, task, board, cfg)
	}

	if err := e.client.UpdateContent(ctx, link.ContentType, link.ContentID, task.Title, body); err != nil {
		return err
	}
	if err := e.ensureIssue(ctx, task, cfg); err != nil {
		return err
	}
	if err := e.pushAssignees(ctx, task, cfg); err != nil {
		return err
	}
	if err := e.pushLabels(ctx, task, cfg); err != nil {
		return err
	}
	if err := e.pushStatus(ctx, task, cfg); err != nil {
		return err
	}
	return e.pushFields(ctx, task, board, cfg)
}

// createRemote adds a task that has never synced to the board, as a
// draft by default or a real issue when the project is configured for
// one, a repository is known, or the card already carries assignees /
// labels that drafts cannot hold.
func (e *Engine) createRemote(ctx context.Context, task *model.Task, cfg *model.GitHubSync, body string) error {
	var (
		itemID string
		err    error
		link   = &model.GitHubLink{}
	)

	wantIssue := cfg.CreateAsIssue ||
		len(github.ParseAssignees(task.Assignee)) > 0 ||
		len(github.NormalizeLabels(task.Labels)) > 0

	if wantIssue && cfg.RepoID != "" {
		issue, ierr := e.client.CreateIssue(ctx, cfg.RepoID, task.Title, body)
		if ierr != nil {
			return ierr
		}
		itemID, err = e.client.AddContentItem(ctx, cfg.ProjectID, issue.ID)
		link.ContentID = issue.ID
		link.ContentType = "Issue"
		link.Number = issue.Number
		link.URL = issue.URL
		link.Repo = cfg.Repo
		link.State = "OPEN"
	} else {
		draft, derr := e.client.AddDraftItem(ctx, cfg.ProjectID, task.Title, body)
		if derr != nil {
			return derr
		}
		itemID = draft.ItemID
		link.ContentID = draft.ContentID
		link.ContentType = "DraftIssue"
	}
	if err != nil {
		return err
	}
	link.ItemID = itemID
	task.GitHub = link

	if err := e.ensureIssue(ctx, task, cfg); err != nil {
		return err
	}
	if err := e.pushAssignees(ctx, task, cfg); err != nil {
		return err
	}
	if err := e.pushLabels(ctx, task, cfg); err != nil {
		return err
	}
	return e.pushStatus(ctx, task, cfg)
}

// ensureIssue promotes a draft to a real issue when the card has
// assignees or labels that only issues can carry, and a repository is
// configured on the sync.
func (e *Engine) ensureIssue(ctx context.Context, task *model.Task, cfg *model.GitHubSync) error {
	link := task.GitHub
	if link == nil || link.ItemID == "" {
		return nil
	}
	if strings.EqualFold(link.ContentType, "Issue") {
		return nil
	}
	needsIssue := len(github.ParseAssignees(task.Assignee)) > 0 ||
		len(github.NormalizeLabels(task.Labels)) > 0
	if !needsIssue || cfg.RepoID == "" {
		return nil
	}
	issue, err := e.client.ConvertDraftToIssue(ctx, link.ItemID, cfg.RepoID)
	if err != nil {
		return err
	}
	link.ContentID = issue.ID
	link.ContentType = "Issue"
	link.Number = issue.Number
	link.URL = issue.URL
	link.State = "OPEN"
	if link.Repo == "" {
		link.Repo = cfg.Repo
	}
	return nil
}

// pushAssignees writes Task.Assignee onto the backing issue. Drafts have
// no assignees API, so they are skipped until promoted to a real issue.
func (e *Engine) pushAssignees(ctx context.Context, task *model.Task, cfg *model.GitHubSync) error {
	link := task.GitHub
	if link == nil || link.ContentID == "" {
		return nil
	}
	if !strings.EqualFold(link.ContentType, "Issue") {
		return nil
	}
	logins := github.ParseAssignees(task.Assignee)
	repo := link.Repo
	if repo == "" {
		repo = cfg.Repo
	}
	ids, err := e.resolveAssigneeIDs(ctx, repo, logins)
	if err != nil {
		return err
	}
	return e.client.SetIssueAssignees(ctx, link.ContentID, ids)
}

// pushLabels writes Task.Labels onto the backing issue, creating any
// missing repo labels along the way. Drafts are skipped.
func (e *Engine) pushLabels(ctx context.Context, task *model.Task, cfg *model.GitHubSync) error {
	link := task.GitHub
	if link == nil || link.ContentID == "" {
		return nil
	}
	if !strings.EqualFold(link.ContentType, "Issue") {
		return nil
	}
	repo := link.Repo
	if repo == "" {
		repo = cfg.Repo
	}
	ids, err := e.resolveLabelIDs(ctx, repo, cfg.RepoID, task.Labels)
	if err != nil {
		return err
	}
	return e.client.SetIssueLabels(ctx, link.ContentID, ids)
}

func (e *Engine) resolveAssigneeIDs(ctx context.Context, repo string, logins []string) ([]string, error) {
	logins = github.ParseAssignees(strings.Join(logins, ","))
	if len(logins) == 0 {
		return []string{}, nil
	}
	if err := e.warmAssigneeCache(ctx, repo); err != nil && len(e.assigneeIDs) == 0 {
		// Still try per-login lookups below.
	}
	ids := make([]string, 0, len(logins))
	for _, login := range logins {
		key := strings.ToLower(login)
		if id := e.assigneeIDs[key]; id != "" {
			ids = append(ids, id)
			continue
		}
		u, err := e.client.LookupUser(ctx, login)
		if err != nil {
			continue
		}
		if e.assigneeIDs == nil {
			e.assigneeIDs = map[string]string{}
		}
		e.assigneeIDs[strings.ToLower(u.Login)] = u.ID
		ids = append(ids, u.ID)
	}
	return ids, nil
}

func (e *Engine) resolveLabelIDs(ctx context.Context, repo, repoID string, names []string) ([]string, error) {
	names = github.NormalizeLabels(names)
	if len(names) == 0 {
		return []string{}, nil
	}
	if err := e.warmLabelCache(ctx, repo); err != nil && len(e.labelIDs) == 0 {
		// Fall through to create-on-miss below.
	}
	ids := make([]string, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(name)
		if id := e.labelIDs[key]; id != "" {
			ids = append(ids, id)
			continue
		}
		if repoID == "" {
			continue
		}
		created, err := e.client.CreateLabel(ctx, repoID, name, "")
		if err != nil {
			continue
		}
		if e.labelIDs == nil {
			e.labelIDs = map[string]string{}
		}
		e.labelIDs[strings.ToLower(created.Name)] = created.ID
		ids = append(ids, created.ID)
	}
	return ids, nil
}

func (e *Engine) warmAssigneeCache(ctx context.Context, repo string) error {
	if repo == "" {
		if e.assigneeIDs == nil {
			e.assigneeIDs = map[string]string{}
		}
		return nil
	}
	if e.assigneeIDs != nil && e.assigneeRepo == repo {
		return e.assigneeErr
	}
	e.assigneeRepo = repo
	e.assigneeIDs = map[string]string{}
	users, err := e.client.ListAssignableUsers(ctx, repo)
	e.assigneeErr = err
	if err != nil {
		return err
	}
	for _, u := range users {
		e.assigneeIDs[strings.ToLower(u.Login)] = u.ID
	}
	return nil
}

func (e *Engine) warmLabelCache(ctx context.Context, repo string) error {
	if repo == "" {
		if e.labelIDs == nil {
			e.labelIDs = map[string]string{}
		}
		return nil
	}
	if e.labelIDs != nil && e.labelRepo == repo {
		return e.labelErr
	}
	e.labelRepo = repo
	e.labelIDs = map[string]string{}
	labels, err := e.client.ListRepoLabels(ctx, repo)
	e.labelErr = err
	if err != nil {
		return err
	}
	for _, l := range labels {
		e.labelIDs[strings.ToLower(l.Name)] = l.ID
	}
	return nil
}

// pushStatus moves the card into the board column matching the task's
// local Kanban column.
func (e *Engine) pushStatus(ctx context.Context, task *model.Task, cfg *model.GitHubSync) error {
	if cfg.StatusFieldID == "" || task.GitHub == nil || task.GitHub.ItemID == "" {
		return nil
	}
	optionID := cfg.StatusMap[task.Status]
	if optionID == "" {
		return nil
	}
	return e.client.SetSingleSelect(ctx, cfg.ProjectID, task.GitHub.ItemID, cfg.StatusFieldID, optionID)
}

// pushFields writes the writable custom field values held on the task.
// Read-only rollups are skipped: GitHub computes those and rejects a
// write against them.
func (e *Engine) pushFields(ctx context.Context, task *model.Task, board *github.Project, cfg *model.GitHubSync) error {
	if task.GitHub == nil || task.GitHub.ItemID == "" {
		return nil
	}
	byID := map[string]github.Field{}
	for _, f := range board.Fields {
		byID[f.ID] = f
	}

	pointsField, hasPoints := storyPointsField(board.Fields)

	for id, v := range task.Fields {
		f, ok := byID[id]
		if !ok || !f.Writable || id == cfg.StatusFieldID {
			continue
		}
		// Native StoryPoints is authoritative for the points column.
		if hasPoints && id == pointsField.ID {
			continue
		}
		val := github.ItemFieldValue{
			FieldID:     id,
			DataType:    f.DataType,
			Text:        v.Text,
			Number:      v.Number,
			Date:        v.Date,
			OptionID:    v.OptionID,
			IterationID: v.Iteration,
		}
		if err := e.client.SetValue(ctx, cfg.ProjectID, task.GitHub.ItemID, f, val); err != nil {
			return err
		}
	}

	if hasPoints {
		if err := e.client.SetNumber(ctx, cfg.ProjectID, task.GitHub.ItemID, pointsField.ID, float64(task.StoryPoints)); err != nil {
			return err
		}
	}
	return nil
}

// taskFromItem builds a fresh local task for a board row JumpStart has
// never seen. Type is inferred from the labels GitHub carries.
func taskFromItem(item github.Item, cfg *model.GitHubSync, now int64) model.Task {
	t := model.Task{
		ID:        "gh-" + item.ID,
		Title:     item.Title,
		Status:    "todo",
		Type:      inferType(item),
		CreatedAt: now,
		UpdatedAt: now,
	}
	applyRemote(&t, item, cfg)
	if t.Status == "" {
		t.Status = "todo"
	}
	t.Done = t.Status == "done"
	clearConflict(&t, parseTime(item.UpdatedAt), now)
	return t
}

func inferType(item github.Item) string {
	if strings.EqualFold(item.IssueType, "bug") {
		return "bug"
	}
	for _, l := range item.Labels {
		switch strings.ToLower(l) {
		case "bug", "defect":
			return "bug"
		case "story", "user story", "epic":
			return "story"
		}
	}
	return "task"
}
