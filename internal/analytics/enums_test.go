package analytics

import "testing"

func TestRuntimeClassification(t *testing.T) {
	cases := []struct {
		language string
		command  string
		want     string
	}{
		{"TypeScript", "npm run dev", "node"},
		{"", "pnpm dev", "node"},
		{"", "go run ./cmd/api", "go"},
		{"Go", "", "go"},
		{"", "uvicorn app:main --reload", "python"},
		{"", "cargo run", "rust"},
		{"", "docker compose up", "docker"},
		{"", "./gradlew bootRun", "java"},
		{"", "", "unknown"},
		{"", "make serve", "other"},
	}
	for _, c := range cases {
		if got := Runtime(c.language, c.command); got != c.want {
			t.Errorf("Runtime(%q, %q) = %q, want %q", c.language, c.command, got, c.want)
		}
	}
}

// Every enum helper feeds Sanitize, so its output has to survive it.
// A helper that returns something Sanitize rejects is a silently broken
// breakdown in GA4.
func TestEnumOutputSurvivesSanitize(t *testing.T) {
	values := []string{
		Runtime("TypeScript", "npm run dev"),
		Runtime("", "make serve"),
		PackageManager("go modules"),
		PackageManager("pnpm"),
		TestFramework("node", "vitest run"),
		TestFramework("", "make test"),
		AIModelFamily("qwen2.5-coder:7b"),
		AIParamSize("qwen2.5-coder:7b"),
		Provider("github.com"),
		Bucket(0.8),
		FailureReason(nil),
	}
	for _, v := range values {
		if got := sanitizeString(v); got != v {
			t.Errorf("enum value %q was rejected by Sanitize as %q", v, got)
		}
	}
}

func TestPackageManager(t *testing.T) {
	cases := map[string]string{
		"go modules": "go", "npm": "npm", "pnpm": "pnpm", "yarn": "yarn",
		"pip": "pip", "cargo": "cargo", "composer": "composer",
		"bundler": "bundler", "": "none", "nix": "other",
	}
	for in, want := range cases {
		if got := PackageManager(in); got != want {
			t.Errorf("PackageManager(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAIModelSplit(t *testing.T) {
	cases := []struct{ model, family, size string }{
		{"qwen2.5-coder:7b", "qwen", "7b"},
		{"llama3.1:8b", "llama", "8b"},
		{"codellama:13b", "codellama", "13b"},
		{"mistral", "mistral", "unknown"},
		{"my-private-tune:latest", "other", "unknown"},
		{"", "none", "unknown"},
	}
	for _, c := range cases {
		if got := AIModelFamily(c.model); got != c.family {
			t.Errorf("AIModelFamily(%q) = %q, want %q", c.model, got, c.family)
		}
		if got := AIParamSize(c.model); got != c.size {
			t.Errorf("AIParamSize(%q) = %q, want %q", c.model, got, c.size)
		}
	}
}
