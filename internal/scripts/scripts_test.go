package scripts

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commands(found []Found) map[string]bool {
	out := map[string]bool{}
	for _, f := range found {
		out[f.Command] = true
	}
	return out
}

func TestDetectNodeSkipsDevAndPlaceholder(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{
		"dev":"next dev",
		"migrate":"prisma migrate deploy",
		"test":"echo \"Error: no test specified\" && exit 1"
	}}`)

	found, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := commands(found)
	if !got["npm run migrate"] {
		t.Errorf("expected npm run migrate, got %v", got)
	}
	if got["npm run dev"] {
		t.Error("dev is a long-running command and should be skipped")
	}
	if got["npm run test"] {
		t.Error("placeholder test script should be skipped")
	}
}

func TestDetectNodeUsesLockfileRunner(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{"build":"tsc"}}`)
	write(t, dir, "pnpm-lock.yaml", "lockfileVersion: 6.0")

	found, _ := Detect(dir)
	if !commands(found)["pnpm run build"] {
		t.Errorf("expected pnpm runner, got %v", commands(found))
	}
}

func TestDetectMakefileTargets(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Makefile", "VERSION = 1.0\n\n.PHONY: migrate\n\nmigrate:\n\tgo run . --migrate\n\nserve:\n\tgo run .\n")

	got := commands(mustDetect(t, dir))
	if !got["make migrate"] {
		t.Errorf("expected make migrate, got %v", got)
	}
	if got["make serve"] {
		t.Error("serve is long-running and should be skipped")
	}
	if got["make VERSION"] {
		t.Error("variable assignment should not be read as a target")
	}
}

func TestDetectGoFlags(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	write(t, dir, "main.go", `package main

import "flag"

func main() {
	migrate := flag.Bool("migrate", false, "run migrations")
	var seed string
	flag.StringVar(&seed, "seed", "", "seed data")
	flag.Int("port", 8080, "port")
	_ = migrate
}
`)

	got := commands(mustDetect(t, dir))
	for _, want := range []string{"go run . --migrate", "go run . --seed", "go build ./...", "go mod tidy"} {
		if !got[want] {
			t.Errorf("expected %q, got %v", want, got)
		}
	}
	if got["go run . --port"] {
		t.Error("port is a config flag, not a task")
	}
}

func TestDetectDeduplicatesCommands(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/app\n")
	write(t, dir, "Makefile", "build:\n\tgo build ./...\n")

	found := mustDetect(t, dir)
	seen := map[string]int{}
	for _, f := range found {
		seen[f.Command]++
	}
	for cmd, n := range seen {
		if n > 1 {
			t.Errorf("command %q appears %d times", cmd, n)
		}
	}
}

func TestDetectEmptyDirReturnsNoScripts(t *testing.T) {
	found, err := Detect(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("expected no scripts, got %v", found)
	}
}

func TestDetectMissingDirErrors(t *testing.T) {
	if _, err := Detect(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("expected an error for a missing directory")
	}
}

func TestLabel(t *testing.T) {
	cases := map[string]string{
		"db:migrate": "Db Migrate",
		"build-prod": "Build Prod",
		"migrate":    "Migrate",
		"seed_data":  "Seed Data",
	}
	for in, want := range cases {
		if got := label(in); got != want {
			t.Errorf("label(%q) = %q, want %q", in, got, want)
		}
	}
}

func mustDetect(t *testing.T, dir string) []Found {
	t.Helper()
	found, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	return found
}
