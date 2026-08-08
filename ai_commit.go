// AI commit messages: hand the pending diff to the local model and get
// back a Conventional Commits subject plus an optional body. Split out of
// ai_api.go because it is the one AI surface that reads the working tree
// rather than the task board.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
	"devdeck/internal/gitops"
)

// CommitSuggestion is a drafted commit message. Subject and Body are kept
// apart so the UI can show the 50/72 split, and Message is the joined
// form that goes straight into the commit box.
type CommitSuggestion struct {
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Message   string `json:"message"`
	Staged    bool   `json:"staged"`    // summarised the index rather than the whole worktree
	FileCount int    `json:"fileCount"` // files the model was shown
	Truncated bool   `json:"truncated"` // the diff was too large to send whole
}

const commitSystemPrompt = "You write git commit messages in the Conventional Commits format. " +
	"Given a diff, return ONLY a JSON object with keys:\n" +
	"  subject — string, the full subject line: \"<type>(<scope>): <description>\". " +
	"type is one of feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert. " +
	"scope is optional and should be a short module or directory name, omitted entirely if the change spans many areas. " +
	"description is imperative mood, lowercase, no trailing period, and the whole subject must be at most 72 characters.\n" +
	"  body — string, may be empty. When the change is non-obvious, one short paragraph or 2-4 \"- \" bullets " +
	"explaining what changed and why. Wrap lines at 72 characters. Do not restate the subject and do not list every file.\n" +
	"Describe what the diff actually does. Never invent issue numbers, ticket IDs, or co-authors. " +
	"Never put text outside the JSON."

// OllamaGenerateCommitMessage drafts a commit message from the project's
// pending changes: staged changes if anything is staged, otherwise every
// uncommitted change.
func (a *App) OllamaGenerateCommitMessage(host, model, projectRoot string) (res CommitSuggestion, err error) {
	start := time.Now()
	// No diff content, paths, or message text: only the shape of the
	// change and what came back. Whether the user keeps the suggestion is
	// reported separately by the frontend, since generating a message is
	// not the same as committing with it.
	defer func() {
		a.track("ai_commit_message_generated", outcome(start, err, map[string]any{
			"model_family":   analytics.AIModelFamily(model),
			"param_size":     analytics.AIParamSize(model),
			"staged":         res.Staged,
			"file_count":     res.FileCount,
			"diff_truncated": res.Truncated,
			"had_body":       res.Body != "",
		}))
	}()

	if strings.TrimSpace(model) == "" {
		return res, fmt.Errorf("no AI model selected — choose one in Preferences → AI")
	}

	ctx, err := gitops.CommitDiff(projectRoot)
	if err != nil {
		return res, err
	}
	if ctx.Empty {
		err = fmt.Errorf("nothing to commit — the working tree is clean")
		return res, err
	}

	res.Staged = ctx.Staged
	res.FileCount = len(ctx.Files)
	res.Truncated = ctx.Truncated

	out, err := ai.New(host).Chat(a.ctx, model, []ai.ChatMessage{
		{Role: "system", Content: commitSystemPrompt},
		{Role: "user", Content: ctx.Prompt()},
	}, true)
	if err != nil {
		return res, err
	}

	var parsed struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if uerr := json.Unmarshal([]byte(extractJSON(out)), &parsed); uerr != nil {
		err = fmt.Errorf("could not parse model output: %w", uerr)
		return res, err
	}

	res.Subject = cleanSubject(parsed.Subject)
	res.Body = strings.TrimSpace(parsed.Body)
	if res.Subject == "" {
		err = fmt.Errorf("the model did not return a commit subject")
		return res, err
	}
	res.Message = res.Subject
	if res.Body != "" {
		res.Message += "\n\n" + res.Body
	}
	return res, nil
}

// cleanSubject strips the decorations models like to add around a subject
// line: surrounding quotes, a leading "Subject:" label, and any second
// line that leaked in from the body.
func cleanSubject(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "Subject:"))
	s = strings.Trim(s, "`\"'")
	return strings.TrimSpace(s)
}
