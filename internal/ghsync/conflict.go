package ghsync

import (
	"time"

	"devdeck/internal/model"
)

// side names which copy of a task a reconcile decided to keep.
type side int

const (
	sideNone   side = iota // neither copy changed since the last sync
	sideLocal              // only JumpStart changed, push it
	sideRemote             // only GitHub changed, pull it
	sideBoth               // both changed, a conflict to resolve
)

// remoteEchoGrace is added to RemoteUpdatedAt after a successful push.
// GitHub's updatedAt is often slightly ahead of the client's clock at
// write time; without this cushion the next local edit looks like
// sideBoth against our own echo and raises a false conflict badge.
const remoteEchoGrace = 2 * time.Minute

// decide compares a task and its remote item against the watermark left
// by the last clean sync, and reports which way the change flows.
//
// A task that has never synced counts as a local change, so a board that
// is linked mid-project pushes everything up rather than silently
// dropping work.
func decide(task model.Task, remoteUpdated int64) side {
	link := task.GitHub
	if link == nil || link.SyncedAt == 0 {
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

// resolve settles a two-sided conflict by last write wins, and flags the
// task so the board can show a badge. The user keeps the losing copy in
// the modal's conflict panel rather than losing it outright.
func resolve(task model.Task, remoteUpdated int64) side {
	if remoteUpdated > task.UpdatedAt {
		return sideRemote
	}
	return sideLocal
}

// clearConflict marks a task reconciled at now, which is what every
// successful push or pull ends with.
func clearConflict(task *model.Task, remoteUpdated, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.Pending = false
	task.GitHub.SyncedAt = now
	if remoteUpdated > 0 {
		task.GitHub.RemoteUpdatedAt = remoteUpdated
	}
}

// stampAfterPush clears pending/conflict flags and advances the remote
// watermark far enough that the post-push GitHub timestamp cannot look
// like an external edit on the next pass.
func stampAfterPush(task *model.Task, now int64) {
	clearConflict(task, now+remoteEchoGrace.Milliseconds(), now)
}

// StampAfterPush is the exported form used by direct GitHub writes
// (field editors) that already landed remotely and must not look like
// a two-sided conflict on the next poll.
func StampAfterPush(task *model.Task, now int64) {
	stampAfterPush(task, now)
}

// MarkKeepLocal records an explicit "Keep mine, push it" choice: clear
// the conflict badge, mark the task pending, and advance the remote
// watermark so the immediate follow-up sync classifies as sideLocal
// instead of sideBoth. Without that cushion, last-write-wins can put
// the Conflict badge right back when GitHub's timestamp is slightly
// ahead of local now.
func MarkKeepLocal(task *model.Task, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.Pending = true
	task.GitHub.SyncedAt = now
	task.UpdatedAt = now
	floor := now + remoteEchoGrace.Milliseconds()
	if task.GitHub.RemoteUpdatedAt < floor {
		task.GitHub.RemoteUpdatedAt = floor
	}
}

// MarkDismissConflict clears the badge and accepts the copy already on
// the board by advancing watermarks so the next poll does not re-raise
// the same divergence.
func MarkDismissConflict(task *model.Task, now int64) {
	if task.GitHub == nil {
		task.GitHub = &model.GitHubLink{}
	}
	task.GitHub.Conflict = false
	task.GitHub.Pending = false
	task.GitHub.SyncedAt = now
	if task.GitHub.RemoteUpdatedAt < now {
		task.GitHub.RemoteUpdatedAt = now
	}
}
