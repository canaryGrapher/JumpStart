package main

import (
	"fmt"
	"strings"
	"time"

	"devdeck/internal/contribute"
)

// contributeRepo is the upstream repository community issues are filed against.
var contributeRepo = contribute.Repo{Owner: UpdateOwner, Name: UpdateRepo}

// ContributeInfo is everything the Contribute settings pane needs to render
// before the user does anything: which repo, whether GitHub is connected, and
// the environment string that will be attached to submissions.
type ContributeInfo struct {
	Repo        string                 `json:"repo"`
	RepoURL     string                 `json:"repoUrl"`
	NewIssueURL string                 `json:"newIssueUrl"`
	IssuesURL   string                 `json:"issuesUrl"`
	Connected   bool                   `json:"connected"`
	Environment contribute.Environment `json:"environment"`
}

// GetContributeInfo reports repository links and whether a GitHub token is
// stored, so the UI can decide between in-app submission and the browser
// fallback.
func (a *App) GetContributeInfo() (ContributeInfo, error) {
	info := ContributeInfo{
		Repo:        contributeRepo.FullName(),
		RepoURL:     contributeRepo.URL(),
		NewIssueURL: contributeRepo.NewIssueURL(),
		IssuesURL:   contributeRepo.URL() + "/issues",
		Environment: contribute.CurrentEnvironment(a.GetAppVersion()),
	}
	info.Connected = ghHasToken()
	return info, nil
}

// SubmitIssue files a bug report or feature request against the JumpStart
// repository using the stored GitHub token, and returns the new issue's URL.
func (a *App) SubmitIssue(draft contribute.Draft) (url string, err error) {
	start := time.Now()
	// The title and body are the user's words and are never sent; only
	// which kind of issue it was and whether the submission worked.
	defer func() {
		a.track("issue_submitted", outcome(start, err, map[string]any{
			"kind":         issueKind(draft.Kind),
			"label_count":  len(draft.Labels),
			"title_length": len(strings.TrimSpace(draft.Title)),
		}))
	}()

	token, err := a.ghAccessToken(a.ctx, false)
	if err != nil || token == "" {
		if err == nil {
			err = fmt.Errorf("connect GitHub in Settings → Git before submitting issues")
		}
		return "", err
	}

	kind := contribute.KindBug
	if draft.Kind == contribute.KindFeature {
		kind = contribute.KindFeature
	}
	draft.Kind = kind

	env := contribute.CurrentEnvironment(a.GetAppVersion())
	url, err = contribute.CreateIssue(contributeRepo, token, contribute.IssueRequest{
		Title:  strings.TrimSpace(draft.Title),
		Body:   contribute.BuildBody(draft, env),
		Labels: contribute.NormalizeLabels(kind, draft.Labels),
	})
	return url, err
}

// issueKind bounds the submission type.
func issueKind(kind contribute.Kind) string {
	if kind == contribute.KindFeature {
		return "feature"
	}
	return "bug"
}

// ListRepoIssues returns open issues from the JumpStart repository. Pass
// "good first issue" as label to show newcomer-friendly work; pass an empty
// label for the most recently updated open issues.
func (a *App) ListRepoIssues(label string, limit int) ([]contribute.Issue, error) {
	token, _ := a.ghAccessToken(a.ctx, false)
	return contribute.ListIssues(contributeRepo, token, label, limit)
}

// CollectDiagnostics builds the optional log attachment: environment, the
// list of running processes, and their recent log lines. It is only called
// when the user has ticked the consent box in the issue form.
func (a *App) CollectDiagnostics() string {
	env := contribute.CurrentEnvironment(a.GetAppVersion())

	var b strings.Builder
	fmt.Fprintf(&b, "JumpStart %s (%s/%s, %s)\n", env.AppVersion, env.OS, env.Arch, env.GoVersion)

	running := a.manager.RunningPIDs()
	if len(running) == 0 {
		b.WriteString("\nNo processes are currently running.\n")
		return b.String()
	}

	names := a.processNames()
	fmt.Fprintf(&b, "\n%d running process(es):\n", len(running))
	for id := range running {
		name := names[id]
		if name == "" {
			name = id
		}
		fmt.Fprintf(&b, "\n--- %s ---\n", name)
		for _, line := range lastLines(a.manager.Logs(id), 100) {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// processNames maps process IDs to "Project / Process" labels for readable
// diagnostics output.
func (a *App) processNames() map[string]string {
	out := map[string]string{}
	projects, err := a.store.Load()
	if err != nil {
		return out
	}
	for _, p := range projects {
		for _, proc := range p.Processes {
			out[proc.ID] = p.Name + " / " + proc.Name
		}
	}
	return out
}

func lastLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
