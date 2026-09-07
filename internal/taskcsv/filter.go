package taskcsv

import "devdeck/internal/model"

// Filter narrows which tasks Export includes. Empty slices mean "any".
// A task must match every non-empty dimension (AND across dimensions,
// OR within a dimension).
type Filter struct {
	Statuses   []string `json:"statuses"`
	SprintIDs  []string `json:"sprintIds"`
	Types      []string `json:"types"`
	Labels     []string `json:"labels"`
	Priorities []string `json:"priorities"`
}

// Select returns the tasks that match f. When f is nil or empty, every
// task is returned unchanged.
func Select(tasks []model.Task, f *Filter) []model.Task {
	if f == nil || f.empty() {
		out := make([]model.Task, len(tasks))
		copy(out, tasks)
		return out
	}
	statusOK := setOf(f.Statuses)
	sprintOK := setOf(f.SprintIDs)
	typeOK := setOf(f.Types)
	labelOK := setOf(f.Labels)
	prioOK := setOf(f.Priorities)

	out := make([]model.Task, 0, len(tasks))
	for _, t := range tasks {
		if len(statusOK) > 0 && !statusOK[t.Status] {
			continue
		}
		if len(sprintOK) > 0 && !sprintOK[t.SprintID] {
			continue
		}
		if len(typeOK) > 0 && !typeOK[normType(t.Type)] {
			continue
		}
		if len(prioOK) > 0 && !prioOK[t.Priority] {
			continue
		}
		if len(labelOK) > 0 && !hasAnyLabel(t.Labels, labelOK) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (f *Filter) empty() bool {
	return len(f.Statuses) == 0 &&
		len(f.SprintIDs) == 0 &&
		len(f.Types) == 0 &&
		len(f.Labels) == 0 &&
		len(f.Priorities) == 0
}

func setOf(vals []string) map[string]bool {
	if len(vals) == 0 {
		return nil
	}
	out := make(map[string]bool, len(vals))
	for _, v := range vals {
		out[v] = true
	}
	return out
}

func normType(t string) string {
	if t == "" {
		return "task"
	}
	return t
}

func hasAnyLabel(labels []string, want map[string]bool) bool {
	for _, l := range labels {
		if want[l] {
			return true
		}
	}
	return false
}
