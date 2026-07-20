package main

// Version is the current application version. Bump this on each release;
// the in-app updater compares it against the latest GitHub release tag.
const Version = "1.2.2"

// UpdateOwner and UpdateRepo identify the GitHub repository whose
// Releases feed the in-app update check.
const (
	UpdateOwner = "canaryGrapher"
	UpdateRepo  = "JumpStart"
)
