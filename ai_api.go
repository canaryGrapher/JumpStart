package main

import (
	"fmt"
	"strings"
	"time"

	"devdeck/internal/ai"
	"devdeck/internal/analytics"
)

// --- Shared AI result types ---

// EnrichResult is the AI-suggested fill for one task/story.
// Field names match the shared task catalog (CSV / import-config prompt).
type EnrichResult struct {
	Description string   `json:"description"`
	Acceptance  []string `json:"acceptance"`
	Subtasks    []string `json:"subtasks"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
	StoryPoints int      `json:"storyPoints"`
}

// GeneratedTask is a child task suggested for a generated story.
type GeneratedTask struct {
	Title string `json:"title"`
}

// GeneratedStory is one user story proposed by the chat assistant.
type GeneratedStory struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Acceptance  []string        `json:"acceptance"`
	Priority    string          `json:"priority"`
	Labels      []string        `json:"labels"`
	StoryPoints int             `json:"storyPoints"`
	Tasks       []GeneratedTask `json:"tasks"`
}

// ChatResult bundles the assistant's prose reply with any stories it
// generated in the same turn, plus the files it was shown.
type ChatResult struct {
	Reply   string           `json:"reply"`
	Stories []GeneratedStory `json:"stories"`
	Sources []string         `json:"sources"` // repo-relative paths from the code index
}

// OllamaListModels returns the models installed on the given Ollama host
// (empty host uses the default localhost:11434).
func (a *App) OllamaListModels(host string) ([]string, error) {
	models, err := ai.New(host).ListModels(a.ctx)
	// Once per session: the UI probes for Ollama whenever an AI surface
	// mounts, and this is a reach question ("can this user use AI at all?")
	// rather than a volume one.
	a.trackOnce("ollama_detected", "ollama_detected", map[string]any{
		"reachable":   err == nil,
		"model_count": len(models),
	})
	return models, err
}

// OllamaEnrichTask asks the model to flesh out a single item: a
// description, acceptance criteria, subtasks, a priority, and labels.
//
// body is the description the user already typed, if any. It is the
// stronger signal, so the model is told to expand on it rather than
// invent a new scope from the title alone.
func (a *App) OllamaEnrichTask(host, model, title, body, kind, projectID string) (res EnrichResult, err error) {
	start := time.Now()
	usedContext := false
	// No titles, bodies, prompts or model output: only the shape of what
	// went in and what came back. `accepted` is not reported here, because
	// generating is not the same as keeping — the frontend fires
	// ai_suggestion_accepted when the user actually applies the result.
	defer func() {
		a.track("ai_task_enriched", outcome(start, err, map[string]any{
			"kind":              taskKind(kind),
			"model_family":      analytics.AIModelFamily(model),
			"param_size":        analytics.AIParamSize(model),
			"used_code_context": usedContext,
			"had_body":          strings.TrimSpace(body) != "",
			"acceptance_count":  len(res.Acceptance),
			"subtask_count":     len(res.Subtasks),
		}))
	}()

	if strings.TrimSpace(title) == "" {
		return res, fmt.Errorf("title is required")
	}
	if kind == "" {
		kind = "task"
	}

	system := "You are a product management assistant. Given a work item, return ONLY a JSON object " +
		"with keys matching JumpStart task fields: " +
		"description (string, one short paragraph; for a story write it as \"As a <role>, I want <goal>, so that <benefit>\"), " +
		"acceptance (array of short testable acceptance-criteria strings — for any type: story, task, or bug), " +
		"subtasks (array of short actionable implementation checklist strings), " +
		"priority (one of \"low\", \"medium\", \"high\"), labels (array of 1-3 short lowercase tags), " +
		"storyPoints (integer 1-13). " +
		"If the user already wrote a description, keep its intent and expand on it instead of replacing it. " +
		"Ground subtasks in the project's actual stack and file layout when context is provided. " +
		"Do not include any prose outside the JSON."

	var user strings.Builder
	fmt.Fprintf(&user, "Item type: %s\nTitle: %s\n", kind, title)
	if strings.TrimSpace(body) != "" {
		fmt.Fprintf(&user, "Existing description:\n%s\n", body)
	}
	if ctx := a.contextFor(projectID, title+" "+body, 5); ctx != "" {
		usedContext = true
		fmt.Fprintf(&user, "\n=== PROJECT CONTEXT ===\n%s", ctx)
	}

	out, err := ai.New(host).Chat(a.ctx, model, []ai.ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user.String()},
	}, true)
	if err != nil {
		return res, err
	}
	parsed, perr := ai.ParseEnrich(out)
	if perr != nil {
		err = perr
		return res, err
	}
	res.Description = parsed.Description
	res.Acceptance = parsed.Acceptance
	res.Subtasks = parsed.Subtasks
	res.Priority = parsed.Priority
	res.Labels = parsed.Labels
	res.StoryPoints = parsed.StoryPoints
	return res, nil
}

// chatSystemPrompt instructs the model to serve both jobs from one thread:
// answering questions about the codebase, and drafting user stories. It
// decides which by whether the user asked for work to be planned.
const chatSystemPrompt = "You are a technical product partner embedded in a specific codebase. You do two things:\n" +
	"1. ANSWER QUESTIONS about the project (how to run it, what is installed, where something lives, how to add a component). " +
	"Answer strictly from the PROJECT CONTEXT below. If the context does not cover it, say so plainly instead of guessing. " +
	"Cite file paths you relied on.\n" +
	"2. DRAFT USER STORIES when the user asks you to create, plan, or break down work.\n\n" +
	"Return ONLY a JSON object with keys:\n" +
	"  reply   — string, your conversational answer in markdown-free plain text.\n" +
	"  stories — array, EMPTY unless the user asked you to plan or break down work.\n" +
	"Each story object uses JumpStart task field names: title (string), description (string, \"As a <role>, I want <goal>, so that <benefit>\"), " +
	"acceptance (array of testable criteria strings), priority (\"low\"|\"medium\"|\"high\"), labels (array of short tags), " +
	"storyPoints (integer 1-13), tasks (array of child task objects each with a title string; children become type \"task\" under the story).\n" +
	"When the user is asking a question rather than requesting work, return an empty stories array. " +
	"Never put text outside the JSON."

// OllamaChat drives the story-generating chat. It receives the full
// conversation, retrieves the code most relevant to the latest turn from
// the project's index, and returns a prose reply plus any stories.
func (a *App) OllamaChat(host, model string, history []ai.ChatMessage, projectID string) (ChatResult, error) {
	var res ChatResult

	system := chatSystemPrompt
	query := lastUserMessage(history)
	if ctx := a.contextFor(projectID, query, 8); ctx != "" {
		system += "\n\n=== PROJECT CONTEXT ===\n" + ctx
		res.Sources = a.sourcesFor(projectID, query, 8)
	}

	msgs := append([]ai.ChatMessage{{Role: "system", Content: system}}, history...)
	out, err := ai.New(host).Chat(a.ctx, model, msgs, true)
	if err != nil {
		return res, err
	}

	parsed := ai.ParseChat(out)
	res.Reply = parsed.Reply
	for _, s := range parsed.Stories {
		story := GeneratedStory{
			Title:       s.Title,
			Description: s.Description,
			Acceptance:  s.Acceptance,
			Priority:    s.Priority,
			Labels:      s.Labels,
			StoryPoints: s.StoryPoints,
		}
		for _, t := range s.Tasks {
			story.Tasks = append(story.Tasks, GeneratedTask{Title: t.Title})
		}
		res.Stories = append(res.Stories, story)
	}
	return res, nil
}

func lastUserMessage(history []ai.ChatMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			return history[i].Content
		}
	}
	return ""
}

// extractJSON pulls the first {...} block out of a model response.
// Kept as a thin wrapper so ai_commit.go and older call sites stay put.
func extractJSON(s string) string { return ai.ExtractJSON(s) }
