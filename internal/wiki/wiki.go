// Package wiki reads a GitHub-compatible project wiki kept in the repo
// (typically a ".wiki" directory). Free private GitHub repos cannot enable
// the hosted wiki, so JumpStart renders the same markdown layout locally.
package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxPageBytes = 1 << 20 // 1 MiB

// Special markdown files GitHub wiki reserves; they are not browsable pages.
var reserved = map[string]bool{
	"_Sidebar.md": true,
	"_Footer.md":  true,
	"_Header.md":  true,
}

// Info describes a detected wiki and its navigable pages.
type Info struct {
	Present    bool   `json:"present"`
	RelRoot    string `json:"relRoot"`    // path relative to project root, e.g. ".wiki"
	SidebarMD  string `json:"sidebarMd"`  // raw _Sidebar.md, empty if absent
	HasSidebar bool   `json:"hasSidebar"`
	Pages      []Page `json:"pages"`
}

// Page is one wiki page (metadata only; body is loaded separately).
type Page struct {
	Name  string `json:"name"`  // filename without .md (GitHub page slug)
	Title string `json:"title"` // display title (hyphens → spaces)
}

// Content is a fully loaded page plus optional footer.
type Content struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Footer  string `json:"footer,omitempty"`
	HasPage bool   `json:"hasPage"`
}

// Detect reports whether projectRoot contains a local wiki and lists pages.
func Detect(projectRoot string) (Info, error) {
	root, rel, err := findRoot(projectRoot)
	if err != nil {
		return Info{}, err
	}
	if root == "" {
		return Info{Present: false, Pages: []Page{}}, nil
	}

	pages, err := listPages(root)
	if err != nil {
		return Info{}, err
	}

	info := Info{
		Present: true,
		RelRoot: rel,
		Pages:   pages,
	}
	if md, err := readFile(filepath.Join(root, "_Sidebar.md")); err == nil {
		info.SidebarMD = md
		info.HasSidebar = true
	}
	return info, nil
}

// ReadPage loads a page by GitHub-style name (with or without .md).
func ReadPage(projectRoot, pageName string) (Content, error) {
	root, _, err := findRoot(projectRoot)
	if err != nil {
		return Content{}, err
	}
	if root == "" {
		return Content{}, fmt.Errorf("no wiki found")
	}

	raw := strings.TrimSpace(pageName)
	if raw == "" {
		raw = "Home"
	}
	if strings.Contains(raw, "..") || strings.ContainsAny(raw, `/\`) {
		return Content{}, fmt.Errorf("invalid page name")
	}

	name := normalizePageName(raw)
	if name == "" {
		return Content{}, fmt.Errorf("invalid page name")
	}
	file := name + ".md"
	if reserved[file] {
		return Content{}, fmt.Errorf("page not found")
	}

	path, err := resolveInWiki(root, file)
	if err != nil {
		return Content{}, err
	}

	body, err := readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Content{Name: name, Title: titleOf(name), HasPage: false}, nil
		}
		return Content{}, err
	}

	out := Content{
		Name:    name,
		Title:   titleOf(name),
		Body:    body,
		HasPage: true,
	}
	if footer, err := readFile(filepath.Join(root, "_Footer.md")); err == nil {
		out.Footer = footer
	}
	return out, nil
}

func findRoot(projectRoot string) (abs string, rel string, err error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", "", fmt.Errorf("project root is empty")
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", "", err
	}
	absRoot = filepath.Clean(absRoot)

	candidates := []struct {
		abs string
		rel string
	}{
		{filepath.Join(absRoot, ".wiki"), ".wiki"},
		// In-repo mirror used by some projects (including JumpStart itself).
		{filepath.Join(absRoot, "docs", "wiki"), "docs/wiki"},
	}
	// Sibling clone of a GitHub wiki repo: MyRepo.wiki next to MyRepo/.
	base := filepath.Base(absRoot)
	sibling := filepath.Join(filepath.Dir(absRoot), base+".wiki")
	candidates = append(candidates, struct {
		abs string
		rel string
	}{sibling, ".." + string(filepath.Separator) + base + ".wiki"})

	for _, c := range candidates {
		if isWikiDir(c.abs) {
			return c.abs, filepath.ToSlash(c.rel), nil
		}
	}
	return "", "", nil
}

func isWikiDir(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".md") && !reserved[name] {
			return true
		}
		if reserved[name] {
			return true
		}
	}
	return false
}

func listPages(root string) ([]Page, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var pages []Page
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if reserved[name] {
			continue
		}
		ext := filepath.Ext(name)
		if !strings.EqualFold(ext, ".md") {
			continue
		}
		// Keep filename casing; GitHub wiki slugs match the on-disk name.
		slug := name[:len(name)-len(ext)]
		pages = append(pages, Page{Name: slug, Title: titleOf(slug)})
	}
	sort.Slice(pages, func(i, j int) bool {
		// Home first, then alphabetical by title.
		if pages[i].Name == "Home" {
			return true
		}
		if pages[j].Name == "Home" {
			return false
		}
		return strings.ToLower(pages[i].Title) < strings.ToLower(pages[j].Title)
	})
	if pages == nil {
		pages = []Page{}
	}
	return pages, nil
}

func normalizePageName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".md") {
		name = name[:len(name)-3]
	}
	// GitHub wiki URLs use spaces as dashes; accept either.
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

func titleOf(name string) string {
	return strings.ReplaceAll(name, "-", " ")
}

func resolveInWiki(root, file string) (string, error) {
	if file == "" || strings.Contains(file, "..") || strings.ContainsAny(file, `/\`) {
		return "", fmt.Errorf("invalid page name")
	}
	joined := filepath.Clean(filepath.Join(root, file))
	prefix := root + string(os.PathSeparator)
	if joined != root && !strings.HasPrefix(joined, prefix) {
		return "", fmt.Errorf("path escapes wiki root")
	}
	return joined, nil
}

func readFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	if info.Size() > maxPageBytes {
		return "", fmt.Errorf("wiki page larger than %d bytes", maxPageBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
