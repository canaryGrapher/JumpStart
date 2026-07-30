package contribute

import (
	"fmt"
	"runtime"
	"strings"
)

// Kind distinguishes the two issue templates the app can file.
type Kind string

const (
	KindBug     Kind = "bug"
	KindFeature Kind = "feature"
)

// Environment is the app/host information appended to every issue so
// maintainers don't have to ask for it.
type Environment struct {
	AppVersion string `json:"appVersion"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	GoVersion  string `json:"goVersion"`
}

// CurrentEnvironment reports the running build's environment.
func CurrentEnvironment(appVersion string) Environment {
	return Environment{
		AppVersion: appVersion,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		GoVersion:  runtime.Version(),
	}
}

// Draft is everything the UI collects for one issue submission.
type Draft struct {
	Kind        Kind     `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	// Diagnostics is optional log/context text the user explicitly agreed
	// to attach. Empty when consent was not given.
	Diagnostics string `json:"diagnostics"`
}

// DefaultLabels are the labels applied when the user adds none of their own.
func DefaultLabels(kind Kind) []string {
	if kind == KindFeature {
		return []string{"enhancement"}
	}
	return []string{"bug"}
}

// BuildBody renders the markdown issue body: the user's description under a
// template heading, optional diagnostics in a collapsed block, and the
// environment table.
func BuildBody(d Draft, env Environment) string {
	var b strings.Builder

	desc := strings.TrimSpace(d.Description)
	if desc == "" {
		desc = "_No description provided._"
	}

	if d.Kind == KindFeature {
		b.WriteString("### What would you like to see?\n\n")
	} else {
		b.WriteString("### What happened?\n\n")
	}
	b.WriteString(desc)
	b.WriteString("\n\n")

	if diag := strings.TrimSpace(d.Diagnostics); diag != "" {
		b.WriteString("<details>\n<summary>Diagnostics</summary>\n\n```\n")
		b.WriteString(truncate(diag, 40000))
		b.WriteString("\n```\n\n</details>\n\n")
	}

	b.WriteString("### Environment\n\n")
	b.WriteString("| | |\n|---|---|\n")
	b.WriteString(fmt.Sprintf("| JumpStart | %s |\n", fallback(env.AppVersion, "unknown")))
	b.WriteString(fmt.Sprintf("| OS | %s |\n", fallback(env.OS, "unknown")))
	b.WriteString(fmt.Sprintf("| Arch | %s |\n", fallback(env.Arch, "unknown")))
	b.WriteString(fmt.Sprintf("| Go | %s |\n", fallback(env.GoVersion, "unknown")))
	b.WriteString("\n_Filed from JumpStart._\n")

	return b.String()
}

// NormalizeLabels trims, de-duplicates, and drops empty labels, falling back
// to the template defaults when nothing usable remains.
func NormalizeLabels(kind Kind, labels []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, l := range labels {
		l = strings.TrimSpace(l)
		if l == "" || seen[strings.ToLower(l)] {
			continue
		}
		seen[strings.ToLower(l)] = true
		out = append(out, l)
	}
	for _, d := range DefaultLabels(kind) {
		if !seen[strings.ToLower(d)] {
			out = append(out, d)
		}
	}
	return out
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n… (truncated)"
}
