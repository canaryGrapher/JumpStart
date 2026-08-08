package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/codectx"
)

// --- Code context (project knowledge base) ---

// CodeContextStatus tells the UI whether a project has been indexed and
// how big the index is.
type CodeContextStatus struct {
	Indexed bool          `json:"indexed"`
	Stats   codectx.Stats `json:"stats"`
	Preview []string      `json:"preview"` // a few headline facts for the UI
	Root    string        `json:"root"`
	Error   string        `json:"error,omitempty"`
}

// BuildCodeContext indexes a project's source tree and saves the result.
// Progress is emitted on the "codectx:progress" event so the UI can show a
// live counter during the scan.
func (a *App) BuildCodeContext(projectID string) (st CodeContextStatus, err error) {
	start := time.Now()
	defer func() {
		a.track("code_context_built", outcome(start, err, map[string]any{
			"project_ref": a.ref(projectID),
			"file_count":  st.Stats.Files,
			"chunk_count": st.Stats.Chunks,
		}))
	}()

	project, err := a.findProject(projectID)
	if err != nil {
		return st, err
	}

	ix, err := codectx.Build(project.ID, project.Name, project.Root, func(done, total int, path string) {
		runtime.EventsEmit(a.ctx, "codectx:progress", map[string]any{
			"projectId": project.ID,
			"done":      done,
			"total":     total,
			"path":      path,
		})
	})
	if err != nil {
		return st, err
	}
	if err := codectx.Save(ix); err != nil {
		return st, err
	}
	codectx.Put(ix)

	runtime.EventsEmit(a.ctx, "codectx:done", project.ID)
	return statusFor(ix, project.Root), nil
}

// GetCodeContextStatus reports the saved index for a project, if any.
func (a *App) GetCodeContextStatus(projectID string) (CodeContextStatus, error) {
	var st CodeContextStatus

	ix, err := codectx.Get(projectID)
	if err != nil {
		st.Error = err.Error()
		return st, nil
	}
	if ix == nil {
		if p, perr := a.findProject(projectID); perr == nil {
			st.Root = p.Root
		}
		return st, nil
	}
	return statusFor(ix, ix.Root), nil
}

// ClearCodeContext deletes a project's index.
func (a *App) ClearCodeContext(projectID string) error {
	codectx.Evict(projectID)
	return codectx.Delete(projectID)
}

// SearchCodeContext returns the code chunks matching a query. Useful for
// a "what does the model see?" debug view in the UI.
func (a *App) SearchCodeContext(projectID, query string, limit int) ([]codectx.Hit, error) {
	ix, err := codectx.Get(projectID)
	if err != nil {
		return nil, err
	}
	if ix == nil {
		return nil, fmt.Errorf("project has not been indexed yet")
	}
	if limit <= 0 {
		limit = 10
	}
	hits := ix.Search(query, limit)
	// The query itself is the user's code question and is never sent. Hit
	// count and a coarse top-score band are enough to tell a retriever that
	// works from one that returns noise.
	a.track("code_context_searched", map[string]any{
		"project_ref":      a.ref(projectID),
		"hit_count":        len(hits),
		"top_score_bucket": topScoreBucket(hits),
		"query_length":     len(query),
	})
	return hits, nil
}

// topScoreBucket coarsens the best BM25 score into a band. BM25 is
// unbounded, so analytics.Bucket's 0..1 thresholds do not apply; these
// cutoffs come from what a "clearly relevant" match scores on this index.
// The point is only to separate a retriever that found something from one
// that returned noise, which is what makes chat answers useless.
func topScoreBucket(hits []codectx.Hit) string {
	if len(hits) == 0 {
		return "none"
	}
	switch score := hits[0].Score; {
	case score < 2:
		return "low"
	case score < 6:
		return "medium"
	default:
		return "high"
	}
}

// contextFor builds the prompt context block for a project + query.
// Returns "" when the project has no index, so AI features keep working
// (just without code awareness) before the first scan.
func (a *App) contextFor(projectID, query string, topK int) string {
	if strings.TrimSpace(projectID) == "" {
		return ""
	}
	ix, err := codectx.Get(projectID)
	if err != nil || ix == nil {
		return ""
	}
	return ix.ContextFor(query, topK)
}

// sourcesFor lists the files the retriever surfaced for a query, so the
// UI can show which parts of the codebase informed an answer.
func (a *App) sourcesFor(projectID, query string, topK int) []string {
	ix, err := codectx.Get(projectID)
	if err != nil || ix == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, h := range ix.Search(query, topK) {
		if !seen[h.Chunk.Path] {
			seen[h.Chunk.Path] = true
			out = append(out, h.Chunk.Path)
		}
	}
	return out
}

// findProject looks up a saved project by ID.
func (a *App) findProject(projectID string) (*modelProject, error) {
	projects, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].ID == projectID {
			p := projects[i]
			return &modelProject{ID: p.ID, Name: p.Name, Root: p.Root}, nil
		}
	}
	return nil, fmt.Errorf("project %s not found", projectID)
}

// modelProject is the slim view of a project the context layer needs.
type modelProject struct {
	ID   string
	Name string
	Root string
}

func statusFor(ix *codectx.Index, root string) CodeContextStatus {
	stats := ix.Stats()
	st := CodeContextStatus{Indexed: true, Stats: stats, Root: root}

	ov := ix.Overview
	if len(ov.Frameworks) > 0 {
		st.Preview = append(st.Preview, strings.Join(ov.Frameworks, ", "))
	}
	for _, m := range ov.Manifests {
		st.Preview = append(st.Preview,
			fmt.Sprintf("%s · %d packages", m.Manager, len(m.Packages)))
	}
	if n := len(ov.Scripts); n > 0 {
		st.Preview = append(st.Preview, fmt.Sprintf("%d run scripts", n))
	}
	return st
}
