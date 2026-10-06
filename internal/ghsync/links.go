package ghsync

import "devdeck/internal/model"

// PreserveGitHubLinks copies a previously synced GitHub link onto any
// incoming task that is missing one. The task modal can open before the
// first sync finishes and then save a draft that never saw the link; without
// this, the next pass treats the card as never-synced and creates a
// duplicate board row.
func PreserveGitHubLinks(before, next []model.Task) []model.Task {
	if len(before) == 0 || len(next) == 0 {
		return next
	}
	byID := make(map[string]*model.GitHubLink, len(before))
	for i := range before {
		t := &before[i]
		if t.GitHub != nil && t.GitHub.ItemID != "" {
			link := *t.GitHub
			byID[t.ID] = &link
		}
	}
	if len(byID) == 0 {
		return next
	}
	out := make([]model.Task, len(next))
	copy(out, next)
	for i := range out {
		if out[i].GitHub != nil && out[i].GitHub.ItemID != "" {
			continue
		}
		if link := byID[out[i].ID]; link != nil {
			cp := *link
			// A local edit that wiped the link still needs to push; keep
			// the remote identity and mark pending so the next pass updates
			// the existing row instead of creating another.
			cp.Pending = true
			out[i].GitHub = &cp
		}
	}
	return out
}

// MergeIncomingTasks applies a board save by task ID so a stale UI snapshot
// cannot clobber a fresher GitHub reconcile that finished while the modal
// was open. Content fields come from next; sync watermarks prefer the
// store copy when it reconciled more recently.
func MergeIncomingTasks(before, next []model.Task) []model.Task {
	if len(before) == 0 || len(next) == 0 {
		return next
	}
	byID := make(map[string]model.Task, len(before))
	for _, t := range before {
		byID[t.ID] = t
	}
	out := make([]model.Task, len(next))
	copy(out, next)
	for i := range out {
		prev, ok := byID[out[i].ID]
		if !ok {
			continue
		}
		out[i].GitHub = mergeGitHubLink(prev.GitHub, out[i].GitHub, prev.UpdatedAt, out[i].UpdatedAt)
	}
	return out
}

// mergeGitHubLink keeps the remote identity and the newer reconcile
// watermarks for one task ID. A content edit that lands after the store
// copy is marked pending so the next pass pushes it; it does not rewind
// SyncedAt / RemoteUpdatedAt to a stale modal draft.
func mergeGitHubLink(prev, next *model.GitHubLink, prevUpdated, nextUpdated int64) *model.GitHubLink {
	if next == nil || next.ItemID == "" {
		if prev == nil || prev.ItemID == "" {
			return next
		}
		cp := *prev
		if nextUpdated > prevUpdated {
			cp.Pending = true
		}
		return &cp
	}
	if prev == nil || prev.ItemID == "" {
		cp := *next
		return &cp
	}

	// Same card: prefer the store's watermarks when they are ahead of the
	// incoming draft (typical race: sync stamped, then Save sent old link).
	if prev.SyncedAt > next.SyncedAt ||
		(prev.SyncedAt == next.SyncedAt && prev.RemoteUpdatedAt > next.RemoteUpdatedAt) {
		cp := *prev
		// Honour an explicit conflict clear/set from the UI on this task.
		cp.Conflict = next.Conflict
		if next.Pending || nextUpdated > prevUpdated {
			cp.Pending = true
		}
		return &cp
	}
	cp := *next
	return &cp
}
