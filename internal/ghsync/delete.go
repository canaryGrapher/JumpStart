package ghsync

import "devdeck/internal/model"

// RemovedLinkedItemIDs returns GitHub Projects item IDs that were linked
// on tasks in before but whose tasks are gone from after. Those board
// rows must be deleted on the next sync pass so a local delete sticks.
func RemovedLinkedItemIDs(before, after []model.Task) []string {
	still := make(map[string]bool, len(after))
	for _, t := range after {
		still[t.ID] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, t := range before {
		if still[t.ID] || t.GitHub == nil || t.GitHub.ItemID == "" {
			continue
		}
		id := t.GitHub.ItemID
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// MergePendingDeletes appends more item IDs onto existing without
// duplicates, preserving order of first appearance.
func MergePendingDeletes(existing, more []string) []string {
	seen := make(map[string]bool, len(existing)+len(more))
	out := make([]string, 0, len(existing)+len(more))
	for _, id := range existing {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range more {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
