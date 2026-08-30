package main

// Version is the running build's version. Release builds override this via
// ldflags (-X main.Version=<tag>) in the CI workflow, so it always matches the
// git tag. Local/dev builds show "dev". The in-app updater compares it against
// the latest GitHub release tag.
var Version = "dev"

// BuildDate is the UTC date the running build was produced, as YYYY-MM-DD.
// Release builds override it via ldflags (-X main.BuildDate=<date>) alongside
// Version. Local/dev builds leave it empty and the About pane hides the row.
var BuildDate = ""

// UpdateOwner and UpdateRepo identify the GitHub repository whose
// Releases feed the in-app update check.
const (
	UpdateOwner = "canaryGrapher"
	UpdateRepo  = "JumpStart"
)

// GitHubClientID is the OAuth app JumpStart authenticates as when the
// user connects GitHub for Projects sync.
//
// This is deliberately a plain hardcoded value, not a build secret. The
// OAuth device flow does not use a client secret at all: the client id
// is only an identifier for which app is asking, and the user's own
// browser session is what actually grants access. GitHub's own CLI ships
// its client id in public source for the same reason. So there is
// nothing to inject in CI and nothing to leak from the binary.
//
// It stays a var rather than a const purely so a fork or an internal
// build can point at its own OAuth app without editing this file:
//
//	wails build -ldflags "-X main.GitHubClientID=Iv1_yourapp"
//
// A build that leaves this empty still works; Settings falls back to
// pasting a personal access token.
var GitHubClientID = "Ov23lipHMyjEOEInTdBR"

// Vendor identifies who ships JumpStart. Shown in Settings → About.
const (
	VendorName = "Workvar"
	VendorURL  = "https://workvar.com"
	ProductURL = "https://jumpstart.workvar.com"
)
