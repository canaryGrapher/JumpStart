package scripts

import "strings"

// tomlSection returns the raw "key = value" lines under a [section]
// heading. This is a deliberately small parser: it only needs to read
// flat script tables out of pyproject.toml / Cargo.toml, not the whole
// TOML grammar, so it avoids pulling in a dependency.
func tomlSection(data []byte, section string) map[string]string {
	out := map[string]string{}
	inSection := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inSection = line == "["+section+"]"
			continue
		}
		if !inSection {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[unquote(strings.TrimSpace(key))] = unquote(strings.TrimSpace(value))
	}
	return out
}

// tomlTableNames returns the trailing name of every "[prefix.<name>]"
// heading, e.g. tomlTableNames(data, "bin") over Cargo.toml.
func tomlTableNames(data []byte, prefix string) []string {
	var out []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "[["+prefix+"]]") && !strings.HasPrefix(line, "["+prefix+".") {
			continue
		}
		if name, ok := strings.CutPrefix(line, "["+prefix+"."); ok {
			out = append(out, unquote(strings.TrimSuffix(name, "]")))
		}
	}
	return out
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
