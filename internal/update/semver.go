package update

import (
	"strconv"
	"strings"
)

// version is a parsed semantic version: three numeric segments plus an
// optional pre-release tail (the part after "-", e.g. "beta.2").
type version struct {
	nums [3]int
	pre  string
}

// parseVersion accepts "v1.2.3", "1.2.3-beta.1", "1.2.3+build" and similar.
// Unparsable numeric segments compare as zero. Build metadata ("+…") is
// ignored, matching semver precedence rules.
func parseVersion(v string) version {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	var out version
	if i := strings.IndexByte(v, '-'); i >= 0 {
		out.pre = v[i+1:]
		v = v[:i]
	}
	for i, part := range strings.SplitN(v, ".", 3) {
		if n, err := strconv.Atoi(part); err == nil {
			out.nums[i] = n
		}
	}
	return out
}

// IsPrerelease reports whether a version tag carries a pre-release tail,
// e.g. "1.4.0-beta.2".
func IsPrerelease(v string) bool {
	return parseVersion(v).pre != ""
}

// IsBeta reports whether a tag belongs to the beta channel: any tag whose
// pre-release tail starts with "beta" (v1.4.0-beta, v1.4.0-beta.2).
func IsBeta(v string) bool {
	return strings.HasPrefix(strings.ToLower(parseVersion(v).pre), "beta")
}

// Compare returns -1, 0 or 1 as a sorts before, equal to, or after b using
// semver precedence. A version with a pre-release tail ranks *below* the
// same version without one, so 1.4.0-beta.1 < 1.4.0.
func Compare(a, b string) int {
	va, vb := parseVersion(a), parseVersion(b)
	for i := 0; i < 3; i++ {
		if va.nums[i] != vb.nums[i] {
			return sign(va.nums[i] - vb.nums[i])
		}
	}
	return comparePre(va.pre, vb.pre)
}

// IsNewer reports whether version a is strictly newer than version b.
func IsNewer(a, b string) bool { return Compare(a, b) > 0 }

// comparePre applies semver pre-release precedence: an absent tail wins,
// then identifiers are compared left to right (numeric < alphanumeric).
func comparePre(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if c := compareIdent(pa[i], pb[i]); c != 0 {
			return c
		}
	}
	return sign(len(pa) - len(pb))
}

func compareIdent(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return sign(na - nb)
	case errA == nil:
		return -1 // numeric identifiers rank below alphanumeric ones
	case errB == nil:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}
