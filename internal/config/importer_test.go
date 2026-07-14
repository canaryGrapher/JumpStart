package config

import (
	"os"
	"path/filepath"
	"testing"

	"devdeck/internal/model"
)

func TestImportPathUsesJumpStartDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := ImportPath()
	if err != nil {
		t.Fatalf("ImportPath() error = %v", err)
	}

	want := filepath.Join(home, ".jumpstart", "import.json")
	if path != want {
		t.Fatalf("ImportPath() = %q, want %q", path, want)
	}
}

func TestMergePreservesExistingUsageStats(t *testing.T) {
	existing := []model.Project{{
		ID:         "existing-id",
		Name:       "JumpStart",
		LastUsedAt: 123,
		UseCount:   7,
	}}
	imported := []model.Project{{
		Name: "JumpStart",
		Root: "/repo",
	}}

	got, added, updated := Merge(existing, imported)
	if added != 0 || updated != 1 {
		t.Fatalf("Merge() added=%d updated=%d, want added=0 updated=1", added, updated)
	}
	if got[0].ID != "existing-id" || got[0].LastUsedAt != 123 || got[0].UseCount != 7 {
		t.Fatalf("Merge() did not preserve identity/usage: %+v", got[0])
	}
}

func TestLoadFillsMissingIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".jumpstart")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"projects":[{"name":"JumpStart","processes":[{"name":"api"}],"tasks":[{"title":"Ship"}]}]}`)
	if err := os.WriteFile(filepath.Join(dir, "import.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	projects, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if projects[0].ID == "" || projects[0].Processes[0].ID == "" || projects[0].Tasks[0].ID == "" {
		t.Fatalf("Load() did not fill IDs: %+v", projects[0])
	}
}

func TestParseAcceptsBareArrayAndFillsIDs(t *testing.T) {
	data := []byte(`[{"name":"Alpha","processes":[{"name":"api"}]}]`)
	projects, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "Alpha" {
		t.Fatalf("Parse() = %+v, want one project named Alpha", projects)
	}
	if projects[0].ID == "" || projects[0].Processes[0].ID == "" {
		t.Fatalf("Parse() did not fill IDs: %+v", projects[0])
	}
}

func TestParseRejectsMissingName(t *testing.T) {
	if _, err := Parse([]byte(`{"projects":[{"root":"/x"}]}`)); err == nil {
		t.Fatal("Parse() should reject a project with no name")
	}
}

func TestWriteImportFileRoundTrips(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	data := []byte(`{"projects":[{"name":"Beta"}]}`)
	if err := WriteImportFile(data); err != nil {
		t.Fatalf("WriteImportFile() error = %v", err)
	}
	projects, err := Load()
	if err != nil {
		t.Fatalf("Load() after write error = %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "Beta" {
		t.Fatalf("round trip = %+v, want one project named Beta", projects)
	}
}
