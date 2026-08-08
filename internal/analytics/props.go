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
func globalProps(opts Options, first bool, session string) map[string]any {
	return map[string]any{
		"app_version":      version(opts.Version),
		"update_channel":   channel(opts.Channel),
		"os":               runtime.GOOS,
		"os_version":       osVersion(),
		"arch":             runtime.GOARCH,
		"locale":           locale(),
		"install_age_days": installAgeDays(opts.Dir),
		"session_id":       session,
		"is_first_session": first,
		"$lib":             "jumpstart-go",
		"$lib_version":     version(opts.Version),
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
