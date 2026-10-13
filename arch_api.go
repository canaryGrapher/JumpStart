package main

import "devdeck/internal/arch"

// ArchitectureInfo reports whether .jumpstart/architecture.xml exists and
// returns section summaries for the Architecture tab sidebar.
func (a *App) ArchitectureInfo(projectRoot string) (arch.Info, error) {
	return arch.Detect(projectRoot)
}

// ArchitectureDoc loads the full parsed architecture document for the
// visual explorer. Returns an error when the file is missing or invalid.
func (a *App) ArchitectureDoc(projectRoot string) (arch.Document, error) {
	return arch.Read(projectRoot)
}
