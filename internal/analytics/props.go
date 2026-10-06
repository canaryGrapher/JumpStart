package analytics

import (
	"os"
	"runtime"
	"strings"
	"sync"
)

// globalProps builds the property set attached to every event by the
// emitter. Call sites never set these: a per-call-site copy would drift,
// and half of them would be forgotten.
func globalProps(opts Options, first bool, sessionStartSec int64) map[string]any {
	return map[string]any{
		"app":              "desktop",
		"platform":         runtime.GOOS,
		"app_version":      version(opts.Version),
		"update_channel":   channel(opts.Channel),
		"os":               runtime.GOOS,
		"os_version":       osVersion(),
		"arch":             runtime.GOARCH,
		"locale":           locale(),
		"install_age_days": installAgeDays(opts.Dir),
		// GA4 MP expects session_id as digits (Unix seconds at session start),
		// not a UUID — otherwise Realtime and active-user tiles stay empty.
		"session_id":       sessionStartSec,
		"is_first_session": first,
	}
}

func version(v string) string {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if v == "" {
		return "dev"
	}
	return v
}

func channel(c string) string {
	if strings.EqualFold(strings.TrimSpace(c), "beta") {
		return "beta"
	}
	return "stable"
}

// locale reports the user's language for drop-support and localisation
// decisions. Only the language/region tag is kept, never the full env value.
func locale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			// "en_US.UTF-8" -> "en-US"
			v = strings.SplitN(v, ".", 2)[0]
			v = strings.ReplaceAll(v, "_", "-")
			if v != "" && v != "C" && v != "POSIX" {
				return v
			}
		}
	}
	return "unknown"
}

var (
	osVersionOnce sync.Once
	osVersionVal  string
)

// osVersion is cached because it can cost a subprocess, and it cannot
// change while the app is running.
func osVersion() string {
	osVersionOnce.Do(func() {
		osVersionVal = readOSVersion()
	})
	return osVersionVal
}
