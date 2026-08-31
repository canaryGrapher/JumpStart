package analytics

import (
	"errors"
	"os"
	"path"
	"strings"
)

// maxValueLen caps how long a string property may be. Every legitimate
// value in this app's taxonomy is a bounded enum or a 12-character ref;
// anything longer is a leak (an error string, a command, a token, a path)
// that slipped past a call site.
const maxValueLen = 32

// maxSliceLen caps a comma-joined []string after each element is sanitized.
// GA4 event params are scalars; joining keeps cardinality bounded.
const maxSliceLen = 128

// secretPrefixes are credential formats that are short enough and dense
// enough to pass the other checks, so they get named explicitly.
var secretPrefixes = []string{
	"ghp_", "gho_", "ghu_", "ghs_", "ghr_", "github_pat_",
	"glpat-", "gltoken-", "sk-", "sk_live", "sk_test", "xox", "bearer",
}

// Sanitize returns a copy of props with anything that could identify the
// user or their code removed. It is the last line of defence: call sites
// are supposed to pass derived shapes already, and this makes a mistake at
// a call site a dropped property rather than a privacy incident.
//
// A string value is rejected when it contains whitespace, a path
// separator, a URL scheme, an "@", an "=", a known credential prefix, or
// exceeds maxValueLen. That covers the shapes this app could plausibly
// leak: paths, remotes, commit messages, prompts, commands, env
// assignments and tokens. Rejected values become "redacted" so the event
// still counts toward its metric.
func Sanitize(props map[string]any) map[string]any {
	if len(props) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(props))
	for k, v := range props {
		switch val := v.(type) {
		case string:
			out[k] = sanitizeString(val)
		case []string:
			list := make([]string, 0, len(val))
			for _, s := range val {
				list = append(list, sanitizeString(s))
			}
			out[k] = joinSlice(list)
		case error:
			out[k] = FailureReason(val)
		default:
			// Numbers, bools and nil carry no content by construction.
			out[k] = v
		}
	}
	return out
}

// joinSlice turns a sanitized string slice into one GA4-safe scalar.
func joinSlice(list []string) string {
	if len(list) == 0 {
		return ""
	}
	joined := strings.Join(list, ",")
	if len(joined) > maxSliceLen {
		joined = joined[:maxSliceLen]
		if i := strings.LastIndex(joined, ","); i >= 0 {
			joined = joined[:i]
		}
	}
	return joined
}

func sanitizeString(s string) string {
	if s == "" {
		return s
	}
	if len(s) > maxValueLen {
		return "redacted"
	}
	// Whitespace is the giveaway for prose: commit messages, prompts,
	// commands and error strings all have it, and no enum in the taxonomy
	// does (internal/deps' "go modules" is normalised by PackageManager
	// before it ever reaches here).
	if strings.ContainsAny(s, " \t\n\r/\\@=") {
		return "redacted"
	}
	if strings.Contains(s, "://") || strings.Contains(s, "..") {
		return "redacted"
	}
	lower := strings.ToLower(s)
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return "redacted"
		}
	}
	return s
}

// PathDepth is how many segments deep a path is. It is the only thing this
// app ever reports about a filesystem location: "users nest projects four
// deep" is useful for judging the sidebar and the folder picker, the path
// itself is not.
//
// Separators are normalised by hand rather than through filepath, because
// this has to give the same answer for a Windows path whichever platform
// the analysis runs on — including in tests on Linux.
func PathDepth(p string) int {
	p = strings.TrimSpace(p)
	if p == "" {
		return 0
	}
	clean := strings.ReplaceAll(p, "\\", "/")
	// A drive letter ("C:") is not a directory level.
	if len(clean) >= 2 && clean[1] == ':' {
		clean = clean[2:]
	}
	clean = path.Clean(clean)

	n := 0
	for _, seg := range strings.Split(clean, "/") {
		if seg != "" && seg != "." {
			n++
		}
	}
	return n
}

// Bounded failure reasons. A raw error string would leak paths and commands
// and would explode GA4's property cardinality, so every failure is
// mapped onto this fixed set.
const (
	ReasonNone            = ""
	ReasonCommandNotFound = "command_not_found"
	ReasonPortInUse       = "port_in_use"
	ReasonPermission      = "permission_denied"
	ReasonMissingPath     = "cwd_missing"
	ReasonNonZeroExit     = "nonzero_exit"
	ReasonNotFound        = "not_found"
	ReasonAuth            = "auth_failed"
	ReasonNetwork         = "network"
	ReasonTimeout         = "timeout"
	ReasonConflict        = "conflict"
	ReasonNotConfigured   = "not_configured"
	ReasonParse           = "parse_failed"
	ReasonOther           = "other"
)

// reasonMatchers is ordered: the first substring that matches wins, so the
// specific cases sit above the generic ones.
var reasonMatchers = []struct {
	needle string
	reason string
}{
	{"executable file not found", ReasonCommandNotFound},
	{"command not found", ReasonCommandNotFound},
	{"not recognized as an internal", ReasonCommandNotFound},
	{"address already in use", ReasonPortInUse},
	{"port is already allocated", ReasonPortInUse},
	{"permission denied", ReasonPermission},
	{"access is denied", ReasonPermission},
	{"operation not permitted", ReasonPermission},
	{"no such file or directory", ReasonMissingPath},
	{"cannot find the path", ReasonMissingPath},
	{"cannot find the file", ReasonMissingPath},
	{"exit status", ReasonNonZeroExit},
	{"401", ReasonAuth},
	{"403", ReasonAuth},
	{"unauthorized", ReasonAuth},
	{"authentication", ReasonAuth},
	{"bad credentials", ReasonAuth},
	{"no access token", ReasonNotConfigured},
	{"not configured", ReasonNotConfigured},
	{"no package manager", ReasonNotConfigured},
	{"no test command", ReasonNotConfigured},
	{"is required", ReasonNotConfigured},
	{"404", ReasonNotFound},
	{"not found", ReasonNotFound},
	{"does not exist", ReasonNotFound},
	{"timeout", ReasonTimeout},
	{"timed out", ReasonTimeout},
	{"deadline exceeded", ReasonTimeout},
	{"connection refused", ReasonNetwork},
	{"no such host", ReasonNetwork},
	{"network is unreachable", ReasonNetwork},
	{"dial tcp", ReasonNetwork},
	{"eof", ReasonNetwork},
	{"already exists", ReasonConflict},
	{"non-fast-forward", ReasonConflict},
	{"conflict", ReasonConflict},
	{"merge", ReasonConflict},
	{"unmarshal", ReasonParse},
	{"invalid character", ReasonParse},
	{"could not parse", ReasonParse},
	{"parse", ReasonParse},
}

// FailureReason maps an error onto one of the bounded reasons above.
// A nil error maps to "", so callers can pass err through unconditionally.
func FailureReason(err error) string {
	if err == nil {
		return ReasonNone
	}
	if errors.Is(err, os.ErrNotExist) {
		return ReasonMissingPath
	}
	if errors.Is(err, os.ErrPermission) {
		return ReasonPermission
	}
	msg := strings.ToLower(err.Error())
	for _, m := range reasonMatchers {
		if strings.Contains(msg, m.needle) {
			return m.reason
		}
	}
	return ReasonOther
}

// Bucket coarsens a 0..1 score into a low-cardinality band. Exact scores
// are noise in aggregate and would make every search a distinct value.
func Bucket(score float64) string {
	switch {
	case score <= 0:
		return "none"
	case score < 0.35:
		return "low"
	case score < 0.7:
		return "medium"
	default:
		return "high"
	}
}
