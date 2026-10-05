package mcpserver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUnderRoot(t *testing.T) {
	root := t.TempDir()
	ok, err := resolveUnderRoot(root, "src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "src", "main.go")
	if ok != want {
		t.Fatalf("got %q want %q", ok, want)
	}

	if _, err := resolveUnderRoot(root, "../escape"); err == nil {
		t.Fatal("expected escape to fail")
	}
	if _, err := resolveUnderRoot(root, "/etc/passwd"); err == nil {
		t.Fatal("expected absolute path to fail")
	}
}

func TestReadWriteList(t *testing.T) {
	root := t.TempDir()
	if err := writeProjectFile(root, "hello.txt", "hi"); err != nil {
		t.Fatal(err)
	}
	got, err := readProjectFile(root, "hello.txt")
	if err != nil || got != "hi" {
		t.Fatalf("read: %q %v", got, err)
	}
	entries, err := listProjectDir(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "hello.txt" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Config{Enabled: true, Port: 9999, Token: "abc"}
	if err := SaveConfig(dir, c); err != nil {
		t.Fatal(err)
	}
	got := LoadConfig(dir)
	if !got.Enabled || got.Port != 9999 || got.Token != "abc" {
		t.Fatalf("got %+v", got)
	}
	// Missing file defaults to disabled with a token.
	empty := LoadConfig(filepath.Join(dir, "missing"))
	if empty.Enabled || empty.Token == "" || empty.Port != DefaultPort {
		t.Fatalf("default: %+v", empty)
	}
	_ = os.RemoveAll(filepath.Join(dir, "missing"))
}
