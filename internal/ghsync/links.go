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
