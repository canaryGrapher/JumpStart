package main

import (
	"runtime"
	"strings"
)

// AboutInfo is the branding + build payload behind Settings → About. It is one
// binding rather than five so the pane renders in a single round trip.
type AboutInfo struct {
	AppName    string `json:"appName"`
	Version    string `json:"version"`
	BuildDate  string `json:"buildDate"`
	Vendor     string `json:"vendor"`
	VendorURL  string `json:"vendorUrl"`
	ProductURL string `json:"productUrl"`
	RepoURL    string `json:"repoUrl"`
	Platform   string `json:"platform"`
	GoVersion  string `json:"goVersion"`
}

// GetAboutInfo returns everything the About pane shows. BuildDate is empty on
// unstamped local builds; the frontend hides the row rather than inventing one.
func (a *App) GetAboutInfo() AboutInfo {
	return AboutInfo{
		AppName:    "JumpStart",
		Version:    strings.TrimPrefix(Version, "v"),
		BuildDate:  BuildDate,
		Vendor:     VendorName,
		VendorURL:  VendorURL,
		ProductURL: ProductURL,
		RepoURL:    "https://github.com/" + UpdateOwner + "/" + UpdateRepo,
		Platform:   runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion:  runtime.Version(),
	}
}
