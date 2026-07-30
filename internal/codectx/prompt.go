package codectx

import (
	"fmt"
	"strings"
)

// Prompt budgets. Local models handle long context poorly, so the
// retrieved code is capped well below the nominal window.
const (
	maxSnippetChars = 9000
	defaultTopK     = 8
)

// OverviewText renders the always-on project summary.
func (ix *Index) OverviewText() string {
	if ix == nil {
		return ""
	}
	ov := ix.Overview
	var b strings.Builder

	fmt.Fprintf(&b, "PROJECT: %s\nROOT: %s\n", ov.Name, ov.Root)

	if len(ov.Languages) > 0 {
		parts := make([]string, 0, len(ov.Languages))
		for i, l := range ov.Languages {
			if i >= 8 {
				break
			}
			parts = append(parts, fmt.Sprintf("%s (%d)", l.Lang, l.Files))
		}
		fmt.Fprintf(&b, "LANGUAGES: %s\n", strings.Join(parts, ", "))
	}
	if len(ov.Frameworks) > 0 {
		fmt.Fprintf(&b, "FRAMEWORKS: %s\n", strings.Join(ov.Frameworks, ", "))
	}

	for _, m := range ov.Manifests {
		fmt.Fprintf(&b, "\nDEPENDENCIES (%s, %s in %s/) — install with `%s`:\n",
			m.Manager, m.File, m.Dir, m.Install)
		b.WriteString(wrapList(m.Packages, 100))
		b.WriteString("\n")
	}

	if len(ov.Scripts) > 0 {
		b.WriteString("\nRUN SCRIPTS:\n")
		for i, s := range ov.Scripts {
			if i >= 40 {
				break
			}
			fmt.Fprintf(&b, "  [%s] %s → %s\n", s.Dir, s.Name, s.Command)
		}
	}

	if len(ov.EnvKeys) > 0 {
		fmt.Fprintf(&b, "\nENV VARS EXPECTED (names only): %s\n",
			strings.Join(capList(ov.EnvKeys, 60), ", "))
	}

	if len(ov.Tree) > 0 {
		b.WriteString("\nDIRECTORY LAYOUT:\n")
		for _, line := range ov.Tree {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	if ov.Readme != "" {
		b.WriteString("\nREADME:\n")
		b.WriteString(ov.Readme)
		b.WriteString("\n")
	}
	return b.String()
}

// SnippetsFor retrieves the code most relevant to a query and renders it
// as a prompt section. Returns an empty string when nothing matches.
func (ix *Index) SnippetsFor(query string, topK int) string {
	if ix == nil {
		return ""
	}
	if topK <= 0 {
		topK = defaultTopK
	}
	hits := ix.Search(query, topK)
	if len(hits) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("RELEVANT CODE:\n")
	for _, h := range hits {
		block := h.Chunk.Render()
		if b.Len()+len(block) > maxSnippetChars {
			break
		}
		b.WriteString(block)
		b.WriteString("\n")
	}
	return b.String()
}

// ContextFor is the full context block for one user turn: the overview
// plus code retrieved for that turn's question.
func (ix *Index) ContextFor(query string, topK int) string {
	if ix == nil {
		return ""
	}
	parts := []string{ix.OverviewText()}
	if s := ix.SnippetsFor(query, topK); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

func wrapList(items []string, perLine int) string {
	var b strings.Builder
	line := "  "
	for _, it := range items {
		if len(line)+len(it)+2 > perLine {
			b.WriteString(strings.TrimRight(line, " "))
			b.WriteString("\n")
			line = "  "
		}
		line += it + ", "
	}
	b.WriteString(strings.TrimRight(strings.TrimRight(line, " "), ","))
	return b.String()
}

func capList(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return append(in[:n:n], "…")
}
