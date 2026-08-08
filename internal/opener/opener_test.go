package opener

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDirRejectsEmpty(t *testing.T) {
	if _, err := resolveDir("   "); err == nil {
		t.Fatal("expected an error for a blank path")
	}
}

func TestResolveDirRejectsMissing(t *testing.T) {
	_, err := resolveDir(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected an error for a missing path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("want a not-exist message, got %v", err)
	}
}

func TestResolveDirReturnsAbsoluteDir(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveDir(dir)
	if err != nil {
		t.Fatalf("resolveDir: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("want an absolute path, got %q", got)
	}
}

// A path saved as a file (e.g. someone pasted the package.json) should still
// open somewhere useful rather than erroring.
func TestResolveDirFallsBackToParentForFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "package.json")
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := resolveDir(file)
	if err != nil {
		t.Fatalf("resolveDir: %v", err)
	}
	if want, _ := filepath.EvalSymlinks(dir); want != "" {
		gotEval, _ := filepath.EvalSymlinks(got)
		if gotEval != want {
			t.Fatalf("want %q, got %q", want, gotEval)
		}
	}
}

func TestEnvCandidateEmptyWhenUnset(t *testing.T) {
	t.Setenv("JUMPSTART_TEST_TERMINAL", "")
	if c := envCandidate("JUMPSTART_TEST_TERMINAL"); c != nil {
		t.Fatalf("want no candidate, got %v", c)
	}
}

func TestEnvCandidateUsesEnvValue(t *testing.T) {
	t.Setenv("JUMPSTART_TEST_TERMINAL", " kitty ")
	got := envCandidate("JUMPSTART_TEST_TERMINAL", "--directory", dirToken)
	if len(got) != 1 || got[0].bin != "kitty" {
		t.Fatalf("want a single kitty candidate, got %v", got)
	}
}

// launch must report a usable error rather than panicking when nothing on the
// list is installed, since that is the whole message the UI shows.
func TestLaunchReportsNothingInstalled(t *testing.T) {
	err := launch(t.TempDir(), "nothing installed", []candidate{
		{bin: "jumpstart-definitely-not-a-real-binary"},
	})
	if err == nil || err.Error() != "nothing installed" {
		t.Fatalf("want the not-found message, got %v", err)
	}
}
