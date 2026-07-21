package main

// Version is the running build's version. Release builds override this via
// ldflags (-X main.Version=<tag>) in the CI workflow, so it always matches the
// git tag. Local/dev builds show "dev". The in-app updater compares it against
// the latest GitHub release tag.
var Version = "dev"

// UpdateOwner and UpdateRepo identify the GitHub repository whose
// Releases feed the in-app update check.
const (
	UpdateOwner = "canaryGrapher"
	UpdateRepo  = "JumpStart"
)
