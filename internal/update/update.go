// Package update checks GitHub Releases for a newer version of the app.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Info describes the result of an update check for the UI.
type Info struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseName    string `json:"releaseName"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseUrl"`
	PublishedAt    string `json:"publishedAt"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Asset is a downloadable release file for the current platform.
type Asset struct {
	Name string
	URL  string
	Size int64
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Check fetches the latest release for owner/repo and compares it against
// currentVersion. Draft and prerelease releases are ignored by the
// /releases/latest endpoint.
func Check(owner, repo, currentVersion string) (Info, error) {
	info := Info{CurrentVersion: currentVersion}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return info, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		// No releases published yet; treat as up to date.
		info.LatestVersion = currentVersion
		return info, nil
	}
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("update check failed: GitHub returned %s", resp.Status)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return info, err
	}
	info.LatestVersion = strings.TrimPrefix(rel.TagName, "v")
	info.ReleaseName = rel.Name
	info.ReleaseNotes = rel.Body
	info.ReleaseURL = rel.HTMLURL
	if !rel.PublishedAt.IsZero() {
		info.PublishedAt = rel.PublishedAt.Format(time.RFC3339)
	}
	info.Available = IsNewer(info.LatestVersion, currentVersion)
	return info, nil
}

// LatestAsset fetches the latest release and returns the downloadable asset
// matching the running platform (GOOS). It returns an error if no matching
// asset is published.
func LatestAsset(owner, repo string) (Asset, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return Asset{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return Asset{}, fmt.Errorf("update fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Asset{}, fmt.Errorf("update fetch failed: GitHub returned %s", resp.Status)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Asset{}, err
	}
	keyword := platformKeyword()
	for _, a := range rel.Assets {
		if strings.Contains(strings.ToLower(a.Name), keyword) {
			return Asset{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size}, nil
		}
	}
	return Asset{}, fmt.Errorf("no release asset found for this platform (%s)", keyword)
}

// platformKeyword maps the running OS to the token used in asset filenames.
func platformKeyword() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// IsNewer reports whether version a is strictly newer than version b.
// Versions are dotted numerics with an optional leading "v"; unparsable
// segments compare as zero.
func IsNewer(a, b string) bool {
	pa, pb := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Drop pre-release/build suffixes: "1.2.3-beta.1" -> "1.2.3".
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		n, err := strconv.Atoi(part)
		if err == nil {
			out[i] = n
		}
	}
	return out
}
