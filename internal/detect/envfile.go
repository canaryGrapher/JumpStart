package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// envFileNames are checked in this order.
var envFileNames = []string{".env", ".env.local", ".env.development", ".env.production"}

// findEnvFiles returns the env file names present in dir.
func findEnvFiles(dir string) []string {
	var found []string
	for _, name := range envFileNames {
		if exists(dir, name) {
			found = append(found, name)
		}
	}
	return found
}

// ParseEnvFile reads a dotenv-style file into a map.
// Supports comments, blank lines, `export ` prefixes and quoted values.
func ParseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		env[key] = cleanValue(val)
	}
	return env, scanner.Err()
}

func cleanValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	// strip trailing inline comment
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}
