package ghsync

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"devdeck/internal/github"
	"devdeck/internal/model"
)

// side names which copy of a task a reconcile decided to keep.
type side int

const (
	sideNone     side = iota // neither copy changed since the last sync
	sideLocal                // only JumpStart changed, push it
	sideRemote               // only GitHub changed, pull it
	sideBoth                 // both changed — content-diff decides next
	sideConflict             // real field divergence; hold until user picks
)

// decide compares a task and its remote item against the watermark left
// by the last clean sync, and reports which way the change flows.
//
// GitHub is the source of truth for quiet pulls. Local edits are pending
// pushes. A never-synced task counts as local so linking mid-project
// uploads existing work. ForcePush (Overwrite GitHub) always pushes.
func decide(task model.Task, remoteUpdated int64) side {
	link := task.GitHub
	if link == nil || link.SyncedAt == 0 {
		return sideLocal
	}
	if link.ForcePush {
		return sideLocal
	}
	localChanged := task.UpdatedAt > link.SyncedAt || link.Pending
	remoteChanged := remoteUpdated > link.RemoteUpdatedAt

	switch {
	case localChanged && remoteChanged:
		return sideBoth
	case localChanged:
		return sideLocal
	case remoteChanged:
		return sideRemote
	}
	return sideNone
}

// DiffTask compares a local task to its GitHub item and returns one
// ConflictField per diverged value. An empty slice means the copies
// match (including a post-push echo with a newer remote timestamp).
func DiffTask(task model.Task, item github.Item, cfg *model.GitHubSync) []model.ConflictField {
	var out []model.ConflictField

	if strings.TrimSpace(task.Title) != strings.TrimSpace(item.Title) {
		out = append(out, model.ConflictField{
			Field: "title", Label: "Title",
			Local: task.Title, Remote: item.Title,
		})
	}

	desc, acceptance, subtasks := ParseBody(item.Body)
	if task.Description != desc {
		out = append(out, model.ConflictField{
			Field: "body", Label: "Description",
			Local: truncateForConflict(task.Description),
			Remote: truncateForConflict(desc),
		})
	} else if bodyHasChecklistMarkers(item.Body) {
		if !sameChecklist(task.Acceptance, acceptance) || !sameChecklist(task.Subtasks, subtasks) {
			out = append(out, model.ConflictField{
				Field: "body", Label: "Checklists",
				Local:  checklistSummary(task.Acceptance, task.Subtasks),
				Remote: checklistSummary(acceptance, subtasks),
			})
		}
	}

	if !strings.EqualFold(item.ContentType, "DraftIssue") {
		remoteAssignee := github.FormatAssignees(item.Assignees)
		if github.FormatAssignees(github.ParseAssignees(task.Assignee)) != remoteAssignee {
			out = append(out, model.ConflictField{
				Field: "assignee", Label: "Assignees",
				Local: task.Assignee, Remote: remoteAssignee,
			})
		}
		localLabels := github.NormalizeLabels(task.Labels)
		remoteLabels := github.NormalizeLabels(item.Labels)
		if !sameLabelSet(localLabels, remoteLabels) {
			out = append(out, model.ConflictField{
				Field: "labels", Label: "Labels",
				Local:  strings.Join(localLabels, ", "),
				Remote: strings.Join(remoteLabels, ", "),
			})
		}
	}

	remoteStatus := remoteColumn(item, cfg)
	if remoteStatus != "" && task.Status != remoteStatus {
		out = append(out, model.ConflictField{
			Field: "status", Label: "Status",
			Local: task.Status, Remote: remoteStatus,
		})
	}

	if sp, ok := storyPointsFromValues(item.Values); ok && task.StoryPoints != sp {
		out = append(out, model.ConflictField{
			Field: "storyPoints", Label: "Story points",
			Local:  strconv.Itoa(task.StoryPoints),
			Remote: strconv.Itoa(sp),
		})
	}

	out = append(out, diffCustomFields(task.Fields, item.Values, cfg)...)
	return out
}

func remoteColumn(item github.Item, cfg *model.GitHubSync) string {
	if cfg != nil && cfg.StatusFieldID != "" {
		if v, ok := item.Values[cfg.StatusFieldID]; ok {
			if col := columnFor(cfg.StatusMap, v.OptionID); col != "" {
				return col
			}
		}
	}
	if strings.EqualFold(item.State, "CLOSED") {
		return "done"
	}
	return ""
}

func diffCustomFields(local map[string]model.FieldValue, remote map[string]github.ItemFieldValue, cfg *model.GitHubSync) []model.ConflictField {
	statusID := ""
	if cfg != nil {
		statusID = cfg.StatusFieldID
	}
	ids := map[string]bool{}
	for id := range local {
		ids[id] = true
	}
	for id := range remote {
		ids[id] = true
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		if id == statusID {
			continue
		}
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)

	var out []model.ConflictField
	for _, id := range sorted {
		lv, lok := local[id]
		rv, rok := remote[id]
		localDisp := ""
		remoteDisp := ""
		label := id
		if lok {
			localDisp = fieldDisplay(lv)
			if lv.Name != "" {
				label = lv.Name
			}
		}
		if rok {
			remoteDisp = itemFieldDisplay(rv)
			if rv.FieldName != "" {
				label = rv.FieldName
			}
		}
		if localDisp == remoteDisp {
			continue
		}
		out = append(out, model.ConflictField{
			Field:  "field:" + id,
			Label:  label,
			Local:  localDisp,
			Remote: remoteDisp,
		})
	}
	return out
}

func fieldDisplay(v model.FieldValue) string {
	if v.Display != "" {
		return v.Display
	}
	if v.Text != "" {
		return v.Text
	}
	if v.Date != "" {
		return v.Date
	}
	if v.OptionKey != "" {
		return v.OptionKey
	}
	if v.OptionID != "" {
		return v.OptionID
	}
	if v.Iteration != "" {
		return v.Iteration
	}
	if v.Number != nil {
		return strconv.FormatFloat(*v.Number, 'f', -1, 64)
	}
	return ""
}

func itemFieldDisplay(v github.ItemFieldValue) string {
	if v.Display != "" {
		return v.Display
	}
	if v.Text != "" {
		return v.Text
	}
	if v.Date != "" {
		return v.Date
	}
	if v.OptionName != "" {
		return v.OptionName
	}
	if v.OptionID != "" {
		return v.OptionID
	}
	if v.IterationID != "" {
		return v.IterationID
	}
	if v.Number != nil {
		return strconv.FormatFloat(*v.Number, 'f', -1, 64)
	}
	return ""
}

func sameLabelSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := map[string]bool{}
	for _, s := range a {
		am[strings.ToLower(s)] = true
	}
	for _, s := range b {
		if !am[strings.ToLower(s)] {
			return false
		}
	}
	return true
}

func checklistSummary(acceptance, subtasks []model.Subtask) string {
	return fmt.Sprintf("%d acceptance, %d subtasks", len(acceptance), len(subtasks))
}

func truncateForConflict(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 80 {
		return s
	}
	return s[:77] + "…"
}

// clearConflict marks a task reconciled at now, which is what every
// successful push or pull ends with.
func clearConflict(task *model.Task, remoteUpdated, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.ConflictFields = nil
	task.GitHub.Pending = false
	task.GitHub.ForcePush = false
	task.GitHub.SyncedAt = now
	if remoteUpdated > 0 {
		task.GitHub.RemoteUpdatedAt = remoteUpdated
	}
}

// stampAfterPush clears pending/conflict flags after a successful write.
// Content-diff conflict detection means a slightly-ahead GitHub echo is
// reconciled quietly on the next poll, so no grace cushion is needed.
func stampAfterPush(task *model.Task, now int64) {
	clearConflict(task, now, now)
}

// StampAfterPush is the exported form used by direct GitHub writes
// (field editors) that already landed remotely.
func StampAfterPush(task *model.Task, now int64) {
	stampAfterPush(task, now)
}

// MarkKeepLocal records "Overwrite GitHub": clear the conflict badge,
// keep local content, and force the next pass to push.
func MarkKeepLocal(task *model.Task, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.ConflictFields = nil
	task.GitHub.Pending = true
	task.GitHub.ForcePush = true
	task.GitHub.SyncedAt = now
	task.UpdatedAt = now
}

// AcceptRemote pulls GitHub onto the local task (SSOT) and clears the
// conflict badge. Used by the Accept GitHub resolve action.
func AcceptRemote(task *model.Task, item github.Item, cfg *model.GitHubSync, now int64) {
	applyRemote(task, item, cfg)
	clearConflict(task, parseTime(item.UpdatedAt), now)
	if task.UpdatedAt < now {
		task.UpdatedAt = now
	}
}

// MarkDismissConflict clears the badge when Accept GitHub cannot pull
// (e.g. the remote row is already gone) or after content already matches.
func MarkDismissConflict(task *model.Task, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.ConflictFields = nil
	task.GitHub.Pending = false
	task.GitHub.ForcePush = false
	task.GitHub.SyncedAt = now
	if task.GitHub.RemoteUpdatedAt < now {
		task.GitHub.RemoteUpdatedAt = now
	}
}

// markConflict stores atomic reasons and holds the local copy until the
// user Accepts GitHub or Overwrites it.
func markConflict(task *model.Task, fields []model.ConflictField) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = true
	task.GitHub.ConflictFields = fields
	task.GitHub.Pending = true
}
