package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsNestedMonorepoApps(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "apps", "web", "package.json"), `{"scripts":{"dev":"vite"},"dependencies":{"vite":"^5.0.0"}}`)
	writeFile(t, filepath.Join(root, "services", "api", "go.mod"), "module example.com/api\n")
	writeFile(t, filepath.Join(root, "node_modules", "ignored", "package.json"), `{"scripts":{"dev":"vite"}}`)

	got, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	byDir := map[string]Detected{}
	for _, d := range got {
		byDir[d.Dir] = d
	}
	if _, ok := byDir[filepath.Join(root, "apps", "web")]; !ok {
		t.Fatalf("Scan() did not find nested web app: %+v", got)
	}
	if _, ok := byDir[filepath.Join(root, "services", "api")]; !ok {
		t.Fatalf("Scan() did not find nested api app: %+v", got)
	}
	if _, ok := byDir[filepath.Join(root, "node_modules", "ignored")]; ok {
		t.Fatalf("Scan() should ignore node_modules, got %+v", got)
	}
}

func TestScanKeepsRunnableRootAndSkipsNestedDependencyFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"dev":"vite"},"dependencies":{"vite":"^5.0.0"}}`)
	writeFile(t, filepath.Join(root, "node_modules", "vite", "package.json"), `{}`)

	got, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Scan() returned %d entries, want 1: %+v", len(got), got)
	}
	if got[0].Dir != root {
		t.Fatalf("Scan()[0].Dir = %q, want root %q", got[0].Dir, root)
	}
}
