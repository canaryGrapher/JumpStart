package analytics

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// installIDFile holds a random UUID generated once per installation. It is
// deliberately NOT a machine ID, MAC address, hostname, or account: those
// identify a person across products, and this must not.
const installIDFile = "install_id"

// installID returns the persistent anonymous ID for this installation,
// creating it on first call. first reports whether it was just created,
// which is what marks a session as the user's very first.
func installID(dir string) (id string, first bool) {
	path := filepath.Join(dir, installIDFile)
	if data, err := os.ReadFile(path); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v, false
		}
	}
	id = randomID()
	// A write failure just means a new ID next launch: annoying for
	// retention math, never a reason to fail a launch.
	_ = os.WriteFile(path, []byte(id), 0o644)
	return id, true
}

// installAgeDays is how long ago this installation first ran, taken from
// the install ID file's creation time. Splitting new users from veterans is
// the single most useful breakdown in every chart.
func installAgeDays(dir string) int {
	info, err := os.Stat(filepath.Join(dir, installIDFile))
	if err != nil {
		return 0
	}
	days := int(time.Since(info.ModTime()).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// randomID returns a RFC 4122 version 4 UUID. It uses crypto/rand so IDs
// cannot be predicted or correlated across installs.
func randomID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Should not happen; a time-seeded fallback still yields a usable
		// per-install identifier rather than an empty distinct_id.
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// Ref maps a local identifier (a project ID, a process ID) to a stable
// opaque token, keyed by the install ID so the same project on two machines
// hashes differently. This makes "how many distinct projects does a user
// start processes in?" answerable without learning what any of them are.
func Ref(secret, value string) string {
	if value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))[:12]
}
