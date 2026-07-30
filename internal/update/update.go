// Package update checks GitHub Releases for a newer version of the app.
package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
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
	Prerelease     bool   `json:"prerelease"` // the offered release is a pre-release
	Channel        string `json:"channel"`    // "stable" | "beta" (channel that was checked)
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

// apiBase is the GitHub REST root. It is a variable so tests can point the
// package at a local server.
var apiBase = "https://api.github.com"

// ChannelName maps the beta flag to the label surfaced in Info.Channel.
func ChannelName(beta bool) string {
	if beta {
		return "beta"
	}
	return "stable"
}

// Check finds the newest release for owner/repo on the requested channel and
// compares it against currentVersion. When beta is false, pre-release tags
// (anything like "v1.4.0-beta") are ignored entirely; when true they are
// eligible alongside stable releases and the highest version wins.
func Check(owner, repo, currentVersion string, beta bool) (Info, error) {
	info := Info{
		CurrentVersion: strings.TrimPrefix(currentVersion, "v"),
		Channel:        ChannelName(beta),
	}
	rel, err := latestRelease(owner, repo, beta)
	if err != nil {
		return info, err
	}
	if rel == nil {
		// No releases published on this channel; treat as up to date.
		info.LatestVersion = info.CurrentVersion
		return info, nil
	}
	info.LatestVersion = strings.TrimPrefix(rel.TagName, "v")
	info.ReleaseName = rel.Name
	info.ReleaseNotes = rel.Body
	info.ReleaseURL = rel.HTMLURL
	info.Prerelease = rel.Prerelease || IsPrerelease(rel.TagName)
	if !rel.PublishedAt.IsZero() {
		info.PublishedAt = rel.PublishedAt.Format(time.RFC3339)
	}
	info.Available = IsNewer(info.LatestVersion, currentVersion)
	return info, nil
}

// LatestAsset returns the downloadable asset matching the running platform
// (GOOS) from the newest release on the requested channel.
func LatestAsset(owner, repo string, beta bool) (Asset, error) {
	rel, err := latestRelease(owner, repo, beta)
	if err != nil {
		return Asset{}, err
	}
	if rel == nil {
		return Asset{}, fmt.Errorf("no releases published")
	}
	keyword := platformKeyword()
	for _, a := range rel.Assets {
		if strings.Contains(strings.ToLower(a.Name), keyword) {
			return Asset{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size}, nil
		}
	}
	return Asset{}, fmt.Errorf("no release asset found for this platform (%s)", keyword)
}

// latestRelease lists recent releases and returns the highest-versioned one
// eligible for the channel, or nil when none qualify. Drafts are always
// skipped. GitHub's /releases/latest endpoint can't serve the beta channel
// (it excludes pre-releases), so both channels go through the list endpoint
// to keep selection consistent.
func latestRelease(owner, repo string, beta bool) (*githubRelease, error) {
	releases, err := listReleases(owner, repo)
	if err != nil {
		return nil, err
	}
	var best *githubRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.TagName == "" {
			continue
		}
		if !beta && (r.Prerelease || IsPrerelease(r.TagName)) {
			continue
		}
		if best == nil || IsNewer(r.TagName, best.TagName) {
			best = r
		}
	}
	return best, nil
}

func listReleases(owner, repo string) ([]githubRelease, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", apiBase, owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update check failed: GitHub returned %s", resp.Status)
	}
	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
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
