// Commit context: the material an AI model needs to write a commit
// message. Kept in gitops (rather than the AI layer) so the "what am I
// about to commit?" question has one answer, and so the size limits that
// keep a diff inside a local model's context window live next to the
// command that produces the diff.
package gitops

import (
	"fmt"
	"strings"
)

// maxPatchChars caps how much unified diff is handed to the model. Local
// models run small context windows, and a commit subject is derived from
// the shape of a change rather than every line of it, so an oversized
// patch is truncated instead of being sent whole (or refused).
const maxPatchChars = 12000

// CommitContext is a snapshot of the pending change, ready to be
// summarised into a commit message.
type CommitContext struct {
	Branch    string     `json:"branch"`
	Staged    bool       `json:"staged"`    // true when summarising the index rather than the whole worktree
	Files     []DiffFile `json:"files"`     // per-file add/delete counts
	Patch     string     `json:"patch"`     // unified diff, possibly truncated
	Truncated bool       `json:"truncated"` // Patch was cut at maxPatchChars
	Empty     bool       `json:"empty"`     // nothing to commit
}

// CommitDiff returns the change that a commit would capture: staged
// changes when anything is staged, otherwise all uncommitted changes.
//
// The fallback matches GitCommit, which stages everything before
// committing. A user who has staged a subset clearly means that subset;
// a user who has staged nothing means "commit what I've been doing".
func CommitDiff(dir string) (*CommitContext, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errEmpty("project root")
	}

	ctx := &CommitContext{Branch: currentBranch(dir), Staged: true}

	// Try the index first. An empty result is not an error, it just means
	// nothing was staged, so fall through to the whole worktree.
	if err := fillDiff(ctx, dir, "diff", "--cached"); err != nil {
		return nil, err
	}
	if ctx.Empty {
		ctx.Staged = false
		if err := fillDiff(ctx, dir, "diff", "HEAD"); err != nil {
			// On a repo with no commits yet, `diff HEAD` has no HEAD to
			// resolve. Everything is new, so compare against the empty tree.
			if !isNoDiffErr(err) {
				return nil, err
			}
			ctx.Empty = true
		}
	}
	return ctx, nil
}

// fillDiff runs one diff invocation and populates the file summary,
// patch and truncation flag on ctx.
func fillDiff(ctx *CommitContext, dir string, args ...string) error {
	statArgs := append(append([]string{}, args...), "--numstat")
	if stat, err := gitCmd(dir, statArgs...); err == nil {
		ctx.Files = parseNumstat(stat)
	}

	patch, err := gitCmd(dir, args...)
	if err != nil {
		return err
	}
	patch = strings.TrimSpace(patch)
	if len(patch) > maxPatchChars {
		patch = patch[:maxPatchChars]
		ctx.Truncated = true
	}
	ctx.Patch = patch
	ctx.Empty = patch == ""
	return nil
}

// currentBranch is best-effort: a detached HEAD or a fresh repo yields an
// empty string, which the prompt simply omits.
func currentBranch(dir string) string {
	out, err := gitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	out = strings.TrimSpace(out)
	if out == "HEAD" {
		return ""
	}
	return out
}

// Prompt renders the context as the plain-text block sent to the model.
func (c *CommitContext) Prompt() string {
	var b strings.Builder

	scope := "all uncommitted changes"
	if c.Staged {
		scope = "staged changes"
	}
	fmt.Fprintf(&b, "Scope: %s\n", scope)
	if c.Branch != "" {
		fmt.Fprintf(&b, "Branch: %s\n", c.Branch)
	}

	if len(c.Files) > 0 {
		b.WriteString("\nFiles changed:\n")
		for _, f := range c.Files {
			fmt.Fprintf(&b, "  %s (+%d/-%d)\n", f.Path, f.Additions, f.Deletions)
		}
	}

	b.WriteString("\nDiff:\n")
	b.WriteString(c.Patch)
	if c.Truncated {
		b.WriteString("\n\n[diff truncated: summarise from the files listed above]")
	}
	return b.String()
}
