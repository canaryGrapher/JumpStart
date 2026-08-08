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

// Vendor identifies who ships JumpStart. Shown in Settings → About.
const (
	VendorName = "Workvar"
	VendorURL  = "https://workvar.com"
	ProductURL = "https://jumpstart.workvar.com"
)
