package deps

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInspectNodeMarksInstalledAndMissingDependencies(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "package.json"), `{
		"dependencies": {"react": "^18.0.0", "vite": "^5.0.0"},
		"devDependencies": {"sass": "^1.0.0"}
	}`)
	writeTestFile(t, filepath.Join(dir, "node_modules", "react", "package.json"), `{}`)
	writeTestFile(t, filepath.Join(dir, "node_modules", "sass", "package.json"), `{}`)

	info, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if info.Manager != "npm" || info.InstallCommand != "npm install" {
		t.Fatalf("Inspect() manager=%q install=%q", info.Manager, info.InstallCommand)
	}
	if info.Installed != 2 || info.Missing != 1 || info.Unknown != 0 {
		t.Fatalf("Inspect() counts installed=%d missing=%d unknown=%d", info.Installed, info.Missing, info.Unknown)
	}
}

func TestInspectPythonUsesVenvWhenPresent(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "requirements.txt"), "requests>=2\nfastapi==0.100\n")
	writeTestFile(t, filepath.Join(dir, ".venv", "bin", "pip"), "")
	writeTestFile(t, filepath.Join(dir, ".venv", "lib", "python3.12", "site-packages", "requests", "__init__.py"), "")

	info, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if info.Manager != "pip" || info.InstallCommand != ".venv/bin/pip install -r requirements.txt" {
		t.Fatalf("Inspect() manager=%q install=%q", info.Manager, info.InstallCommand)
	}
	if info.Installed != 1 || info.Missing != 1 || info.Unknown != 0 {
		t.Fatalf("Inspect() counts installed=%d missing=%d unknown=%d", info.Installed, info.Missing, info.Unknown)
	}
}

func TestInspectCargoUsesCargoHomeCache(t *testing.T) {
	dir := t.TempDir()
	cargoHome := t.TempDir()
	t.Setenv("CARGO_HOME", cargoHome)
	writeTestFile(t, filepath.Join(dir, "Cargo.toml"), `[dependencies]
serde = "1.0"
tokio = "1.0"
`)
	writeTestFile(t, filepath.Join(cargoHome, "registry", "src", "index.crates.io-abc", "serde-1.0.0", "Cargo.toml"), "")

	info, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if info.Installed != 1 || info.Missing != 1 || info.Unknown != 0 {
		t.Fatalf("Inspect() counts installed=%d missing=%d unknown=%d", info.Installed, info.Missing, info.Unknown)
	}
}

func TestInspectRubyUsesVendorBundle(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "Gemfile"), "gem 'rails', '~> 7.1'\ngem 'puma'\n")
	writeTestFile(t, filepath.Join(dir, "vendor", "bundle", "ruby", "3.3.0", "gems", "rails-7.1.0", "README.md"), "")

	info, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	if info.Installed != 1 || info.Missing != 1 || info.Unknown != 0 {
		t.Fatalf("Inspect() counts installed=%d missing=%d unknown=%d", info.Installed, info.Missing, info.Unknown)
	}
}
