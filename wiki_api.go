package main

import "devdeck/internal/wiki"

// WikiInfo reports whether the project has a local GitHub-style wiki
// (typically a ".wiki" directory) and lists its pages.
func (a *App) WikiInfo(projectRoot string) (wiki.Info, error) {
	return wiki.Detect(projectRoot)
}

// WikiPage loads one wiki page by GitHub-style name (e.g. "Home" or
// "Code-Structure"). HasPage is false when the slug does not exist.
func (a *App) WikiPage(projectRoot, pageName string) (wiki.Content, error) {
	return wiki.ReadPage(projectRoot, pageName)
}
