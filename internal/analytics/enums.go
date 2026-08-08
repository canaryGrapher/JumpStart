package analytics

import "strings"

// Every free-text value the app knows about is collapsed onto a fixed set
// before it becomes a property. Two reasons: an unbounded property blows up
// PostHog's cardinality and makes breakdowns useless, and a user's command
// or package name is content we have no business collecting.

// Runtime classifies a process from its command and detected language.
// "other" is a real answer, not a failure: a large "other" slice is the
// signal that a runtime is worth detecting properly.
func Runtime(language, command string) string {
	l := strings.ToLower(language)
	switch {
	case strings.Contains(l, "node"), strings.Contains(l, "javascript"),
		strings.Contains(l, "typescript"):
		return "node"
	case strings.Contains(l, "go"):
		return "go"
	case strings.Contains(l, "python"):
		return "python"
	case strings.Contains(l, "rust"):
		return "rust"
	case strings.Contains(l, "ruby"):
		return "ruby"
	case strings.Contains(l, "php"):
		return "php"
	case strings.Contains(l, "java"):
		return "java"
	case strings.Contains(l, "docker"):
		return "docker"
	}

	c := strings.ToLower(command)
	switch {
	case c == "":
		return "unknown"
	case hasAnyPrefix(c, "npm ", "pnpm ", "yarn ", "bun ", "node ", "npx ", "deno "):
		return "node"
	case hasAnyPrefix(c, "go ", "air", "gow "):
		return "go"
	case hasAnyPrefix(c, "python", "py ", "uvicorn ", "flask ", "django-admin", "poetry ", "uv "):
		return "python"
	case hasAnyPrefix(c, "cargo "):
		return "rust"
	case hasAnyPrefix(c, "bundle ", "rails ", "ruby "):
		return "ruby"
	case hasAnyPrefix(c, "php ", "artisan", "composer "):
		return "php"
	case hasAnyPrefix(c, "docker", "docker-compose"):
		return "docker"
	case hasAnyPrefix(c, "mvn ", "gradle ", "./gradlew", "java "):
		return "java"
	case hasAnyPrefix(c, "dotnet "):
		return "dotnet"
	}
	return "other"
}

// PackageManager collapses internal/deps' manager strings ("go modules",
// "npm", "pip", ...) onto stable tokens.
func PackageManager(manager string) string {
	m := strings.ToLower(strings.TrimSpace(manager))
	switch {
	case m == "":
		return "none"
	case strings.HasPrefix(m, "go"):
		return "go"
	case strings.Contains(m, "pnpm"):
		return "pnpm"
	case strings.Contains(m, "yarn"):
		return "yarn"
	// "bundle" before "bun": bundler contains bun as a substring.
	case strings.Contains(m, "bundle"):
		return "bundler"
	case strings.Contains(m, "bun"):
		return "bun"
	case strings.Contains(m, "npm"):
		return "npm"
	case strings.Contains(m, "pip"), strings.Contains(m, "poetry"), strings.Contains(m, "uv"):
		return "pip"
	case strings.Contains(m, "cargo"):
		return "cargo"
	case strings.Contains(m, "composer"):
		return "composer"
	}
	return "other"
}

// TestFramework collapses internal/testrunner's detected kind plus the
// command into a bounded framework name.
func TestFramework(kind, command string) string {
	c := strings.ToLower(command)
	switch {
	case strings.Contains(c, "vitest"):
		return "vitest"
	case strings.Contains(c, "jest"):
		return "jest"
	case strings.Contains(c, "playwright"):
		return "playwright"
	case strings.Contains(c, "cypress"):
		return "cypress"
	case strings.Contains(c, "pytest"):
		return "pytest"
	case strings.Contains(c, "go test"):
		return "go_test"
	case strings.Contains(c, "cargo test"):
		return "cargo_test"
	}
	switch strings.ToLower(kind) {
	case "node":
		return "node_other"
	case "go":
		return "go_test"
	case "python":
		return "pytest"
	}
	return "custom"
}

// AIModelFamily and AIParamSize split an Ollama model tag such as
// "qwen2.5-coder:7b" into two low-cardinality properties. The raw tag can
// encode a user's private fine-tune name, so it is never sent.
func AIModelFamily(model string) string {
	m := strings.ToLower(model)
	for _, family := range []string{"llama", "qwen", "mistral", "gemma", "phi", "deepseek", "codellama", "starcoder", "granite"} {
		if strings.Contains(m, family) {
			// "codellama" must win over "llama"; the loop order handles the
			// rest, and this keeps the more specific name when both match.
			if family == "llama" && strings.Contains(m, "codellama") {
				return "codellama"
			}
			return family
		}
	}
	if m == "" {
		return "none"
	}
	return "other"
}

// AIParamSize returns the parameter-count tag ("7b", "13b") when the model
// name carries one.
func AIParamSize(model string) string {
	m := strings.ToLower(model)
	idx := strings.LastIndex(m, ":")
	if idx < 0 || idx == len(m)-1 {
		return "unknown"
	}
	tag := m[idx+1:]
	// Longest first: "13b" must win before "3b" matches inside it.
	for _, size := range []string{
		"0.5b", "1.5b",
		"70b", "34b", "32b", "30b", "27b", "20b", "14b", "13b",
		"9b", "8b", "7b", "4b", "3b", "2b", "1b",
	} {
		if strings.Contains(tag, size) {
			return size
		}
	}
	return "unknown"
}

// Provider classifies a git remote host without revealing the remote.
func Provider(host string) string {
	h := strings.ToLower(host)
	switch {
	case strings.Contains(h, "github"):
		return "github"
	case strings.Contains(h, "gitlab"):
		return "gitlab"
	case strings.Contains(h, "bitbucket"):
		return "bitbucket"
	case h == "":
		return "unknown"
	}
	return "other"
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
