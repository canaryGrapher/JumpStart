package github

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultIterationDays = 14

// EnsureIterations makes sure each title exists as a cycle on the board's
// Iteration field. Missing titles are appended; if the board has no
// Iteration field yet, one named "Sprint" is created.
//
// GitHub's updateProjectV2Field overwrites the whole iteration list and
// regenerates cycle ids, so this rewrites the active configuration with
// every existing cycle plus the new ones. Completed iterations are left
// alone (they are not part of the writable configuration payload).
func (c *Client) EnsureIterations(ctx context.Context, projectID string, titles []string) error {
	wanted := uniqueTitles(titles)
	if len(wanted) == 0 {
		return nil
	}

	board, err := c.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	field, ok := findIterationField(board.Fields)
	if !ok {
		return c.createIterationField(ctx, projectID, wanted)
	}

	have := map[string]bool{}
	for _, it := range field.Iterations {
		if it.Completed {
			continue
		}
		have[strings.ToLower(strings.TrimSpace(it.Title))] = true
	}

	var missing []string
	for _, title := range wanted {
		if !have[strings.ToLower(title)] {
			missing = append(missing, title)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	iters := make([]map[string]any, 0, len(field.Iterations)+len(missing))
	nextStart := time.Now().UTC().Truncate(24 * time.Hour)
	for _, it := range field.Iterations {
		if it.Completed {
			continue
		}
		start := it.StartDate
		dur := it.Duration
		if dur <= 0 {
			dur = defaultIterationDays
		}
		if start == "" {
			start = nextStart.Format("2006-01-02")
		}
		iters = append(iters, map[string]any{
			"title":     it.Title,
			"startDate": start,
			"duration":  dur,
		})
		if t, err := time.Parse("2006-01-02", start); err == nil {
			end := t.AddDate(0, 0, dur)
			if end.After(nextStart) {
				nextStart = end
			}
		}
	}
	for _, title := range missing {
		iters = append(iters, map[string]any{
			"title":     title,
			"startDate": nextStart.Format("2006-01-02"),
			"duration":  defaultIterationDays,
		})
		nextStart = nextStart.AddDate(0, 0, defaultIterationDays)
	}

	vars := map[string]any{
		"fieldId": field.ID,
		"configuration": map[string]any{
			"startDay":   2, // Monday
			"duration":   defaultIterationDays,
			"iterations": iters,
		},
	}
	return c.Query(ctx, mutationUpdateIterationField, vars, nil)
}

func (c *Client) createIterationField(ctx context.Context, projectID string, titles []string) error {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	iters := make([]map[string]any, 0, len(titles))
	for i, title := range titles {
		iters = append(iters, map[string]any{
			"title":     title,
			"startDate": start.AddDate(0, 0, i*defaultIterationDays).Format("2006-01-02"),
			"duration":  defaultIterationDays,
		})
	}
	vars := map[string]any{
		"projectId": projectID,
		"name":      "Sprint",
		"configuration": map[string]any{
			"startDay":   2,
			"duration":   defaultIterationDays,
			"iterations": iters,
		},
	}
	if err := c.Query(ctx, mutationCreateIterationField, vars, nil); err != nil {
		return fmt.Errorf("creating Sprint iteration field: %w", err)
	}
	return nil
}

func findIterationField(fields []Field) (Field, bool) {
	var fallback Field
	found := false
	for _, f := range fields {
		if f.DataType != FieldIteration {
			continue
		}
		if strings.EqualFold(f.Name, "Sprint") || strings.EqualFold(f.Name, "Iteration") || strings.EqualFold(f.Name, "Sprints") {
			return f, true
		}
		if !found {
			fallback, found = f, true
		}
	}
	return fallback, found
}

func uniqueTitles(titles []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(titles))
	for _, t := range titles {
		t = strings.TrimSpace(t)
		if t == "" || strings.EqualFold(t, "backlog") {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}
