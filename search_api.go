package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
	"devdeck/internal/appsettings"
	"devdeck/internal/daterange"
	"devdeck/internal/model"
	"devdeck/internal/ocr"
	"devdeck/internal/search"
)

// --- Global search, attachment text (OCR) and AI search ---

func ocrConfig(s appsettings.Settings) ocr.Config {
	return ocr.Config{Engine: s.OCREngine, OllamaHost: s.OCRHost, OllamaModel: s.OCRModel}
}

// textWorker returns the background attachment-text worker, starting it on
// first use. New text emits "search:updated" so an open palette refreshes.
func (a *App) textWorker() *ocr.Worker {
	a.ocrOnce.Do(func() {
		dir := analytics.DataDir()
		a.ocr = ocr.NewWorker(dir, func() ocr.Config { return ocrConfig(appsettings.Load(dir)) }, func() {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "search:updated")
			}
		})
		if projects, err := a.store.Load(); err == nil {
			go a.ocr.Prune(projects)
		}
	})
	return a.ocr
}

// SearchRequest is one palette search.
type SearchRequest struct {
	Query     string `json:"query"`
	ProjectID string `json:"projectId"` // empty searches every project
	HideDone  bool   `json:"hideDone"`
	Limit     int    `json:"limit"`
}

func (a *App) runSearch(req SearchRequest) (search.Response, error) {
	projects, err := a.store.Load()
	if err != nil {
		return search.Response{}, err
	}
	dir := analytics.DataDir()
	s := appsettings.Load(dir)
	return search.Search(projects, req.Query, search.Options{
		ProjectID: req.ProjectID, HideDone: req.HideDone, Limit: req.Limit,
		Today: time.Now(), DateOrder: s.DateOrder, GlobalQuarters: daterange.LoadGlobal(dir),
		Text: a.textWorker().Text,
	}), nil
}

// Search finds projects, tasks and attachment contents. See internal/search
// for the query language (dates, status:, @assignee, #label, has:file …).
func (a *App) Search(req SearchRequest) (search.Response, error) {
	return a.runSearch(req)
}

// AISearchResult is a plain-language search: the filter the model chose
// (shown so it can be checked) and the real tasks it matches.
type AISearchResult struct {
	Plan    search.AIPlan   `json:"plan"`
	Results []search.Result `json:"results"`
	Total   int             `json:"total"`
	Stopped bool            `json:"stopped"`
}

// AISearch asks the local model to turn a question into a task filter, then
// runs that filter. Results are always real tasks.
func (a *App) AISearch(host, model, question, projectID string, opts AIRequestOptions) (AISearchResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return AISearchResult{}, fmt.Errorf("ask a question first")
	}
	projects, err := a.store.Load()
	if err != nil {
		return AISearchResult{}, err
	}
	scope := projects
	if projectID != "" {
		scope = nil
		for _, p := range projects {
			if p.ID == projectID {
				scope = append(scope, p)
			}
		}
	}
	vocab := search.BuildVocab(scope)
	now := time.Now()
	system, user := search.AIPrompt(question, vocab, now)
	ctx, chatOpts, done := a.aiCall(host, model, opts)
	defer done()
	reply, err := ai.New(host).ChatWith(ctx, model, []ai.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, true, chatOpts)
	if err == ai.ErrCanceled {
		return AISearchResult{Stopped: true}, nil
	}
	if err != nil {
		return AISearchResult{}, err
	}
	plan, err := search.ParseAIPlan(reply, vocab)
	if err != nil {
		return AISearchResult{}, err
	}
	results, err := search.RunPlan(scope, plan, "", daterange.LoadGlobal(analytics.DataDir()), now)
	if err != nil {
		return AISearchResult{}, err
	}
	out := AISearchResult{Plan: plan, Total: len(results), Results: results}
	if len(out.Results) > 100 {
		out.Results = out.Results[:100]
	}
	return out, nil
}

func (a *App) findAttachment(projectID, taskID, attachmentID string) (model.Attachment, error) {
	p, err := a.projectByID(projectID)
	if err != nil {
		return model.Attachment{}, err
	}
	for _, t := range p.Tasks {
		if t.ID != taskID {
			continue
		}
		for _, att := range t.Attachments {
			if att.ID == attachmentID {
				return att, nil
			}
		}
	}
	return model.Attachment{}, fmt.Errorf("attachment not found")
}

// AttachmentTextStatus reports whether an attachment's text is searchable.
func (a *App) AttachmentTextStatus(projectID, taskID, attachmentID string) (ocr.Info, error) {
	att, err := a.findAttachment(projectID, taskID, attachmentID)
	if err != nil {
		return ocr.Info{}, err
	}
	w := a.textWorker()
	if info := w.Status(projectID, taskID, att); info.State != "none" || info.Error != "" || info.At != 0 {
		return info, nil
	}
	w.Text(projectID, taskID, att) // queue it if it can be read
	return w.Status(projectID, taskID, att), nil
}

// RerunAttachmentText drops an attachment's cached text and reads it again
// with the current OCR settings.
func (a *App) RerunAttachmentText(projectID, taskID, attachmentID string) (ocr.Info, error) {
	att, err := a.findAttachment(projectID, taskID, attachmentID)
	if err != nil {
		return ocr.Info{}, err
	}
	w := a.textWorker()
	w.Forget(projectID, taskID, att)
	w.Text(projectID, taskID, att)
	return w.Status(projectID, taskID, att), nil
}

// OCREngines reports which OCR engines (and PDF text) work on this Mac.
func (a *App) OCREngines() []ocr.Status {
	dir := analytics.DataDir()
	return ocr.Engines(dir, ocrConfig(appsettings.Load(dir)))
}

// RereadAllImages forgets the text read from every image (failedOnly: just
// the ones that failed) so the next search reads them again. It returns how
// many were reset.
func (a *App) RereadAllImages(failedOnly bool) int {
	n := a.textWorker().Reset(failedOnly)
	if projects, err := a.store.Load(); err == nil {
		go func() {
			// Queue everything now instead of waiting for the next search.
			w := a.textWorker()
			for _, p := range projects {
				for _, t := range p.Tasks {
					for _, att := range t.Attachments {
						w.Text(p.ID, t.ID, att)
					}
				}
			}
		}()
	}
	return n
}
