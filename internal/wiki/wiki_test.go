package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	info, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Present {
		t.Fatal("expected no wiki")
	}
}

func TestDetectDotWiki(t *testing.T) {
	root := t.TempDir()
	wikiDir := filepath.Join(root, ".wiki")
	if err := os.Mkdir(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wikiDir, "Home.md"), "# Hello\n")
	write(t, filepath.Join(wikiDir, "Architecture.md"), "# Arch\n")
	write(t, filepath.Join(wikiDir, "_Sidebar.md"), "* [Home](Home)\n")
	write(t, filepath.Join(wikiDir, "_Footer.md"), "footer\n")

	info, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present || info.RelRoot != ".wiki" {
		t.Fatalf("present=%v root=%q", info.Present, info.RelRoot)
	}
	if !info.HasSidebar || info.SidebarMD == "" {
		t.Fatal("expected sidebar")
	}
	if len(info.Pages) != 2 {
		t.Fatalf("pages=%d", len(info.Pages))
	}
	if info.Pages[0].Name != "Home" {
		t.Fatalf("Home should be first, got %q", info.Pages[0].Name)
	}

	page, err := ReadPage(root, "Architecture")
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasPage || page.Title != "Architecture" || page.Body == "" {
		t.Fatalf("%+v", page)
	}
	if page.Footer != "footer\n" {
		t.Fatalf("footer=%q", page.Footer)
	}
}

func TestDetectDocsWiki(t *testing.T) {
	root := t.TempDir()
	wikiDir := filepath.Join(root, "docs", "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wikiDir, "Home.md"), "# Docs wiki\n")

	info, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present || info.RelRoot != "docs/wiki" {
		t.Fatalf("present=%v root=%q", info.Present, info.RelRoot)
	}
}

func TestDetectSiblingWiki(t *testing.T) {
	parent := t.TempDir()
	proj := filepath.Join(parent, "acme")
	sib := filepath.Join(parent, "acme.wiki")
	if err := os.Mkdir(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sib, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(sib, "Home.md"), "# Sib\n")

	info, err := Detect(proj)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present {
		t.Fatal("expected sibling wiki")
	}
}

func TestPathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	wikiDir := filepath.Join(root, ".wiki")
	if err := os.Mkdir(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wikiDir, "Home.md"), "# H\n")

	_, err := ReadPage(root, "../etc/passwd")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizePageName(t *testing.T) {
	cases := map[string]string{
		"Home":           "Home",
		"Home.md":        "Home",
		"Code Structure": "Code-Structure",
		"./Architecture": "",
		"../x":           "",
		"":               "",
	}
	for in, want := range cases {
		if got := normalizePageName(in); got != want {
			t.Errorf("normalizePageName(%q)=%q want %q", in, got, want)
		}
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
