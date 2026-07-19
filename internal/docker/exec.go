package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// cmdTimeout bounds every docker invocation so a hung daemon can't wedge the UI.
const cmdTimeout = 30 * time.Second

// run executes `docker <args...>` in dir and returns combined stdout. On a
// non-zero exit it returns an error carrying the trimmed stderr so the UI can
// surface a useful message.
func run(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("docker %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

// runJSONLines runs a docker command whose output is newline-delimited JSON
// objects (the default `--format json` shape in Compose v2 and the
// `{{json .}}` shape for base docker commands) and decodes each line into a
// generic map. Blank lines and undecodable lines are skipped rather than
// failing the whole call.
func runJSONLines(dir string, args ...string) ([]map[string]any, error) {
	out, err := run(dir, args...)
	if err != nil {
		return nil, err
	}
	return parseJSONLines(out), nil
}

func parseJSONLines(out string) []map[string]any {
	var rows []map[string]any

	trimmed := strings.TrimSpace(out)
	// Some docker/compose versions emit a single JSON array instead of NDJSON.
	if strings.HasPrefix(trimmed, "[") {
		var arr []map[string]any
		if json.Unmarshal([]byte(trimmed), &arr) == nil {
			return arr
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) == nil {
			rows = append(rows, row)
		}
	}
	return rows
}

// str reads a string field from a decoded JSON row, tolerating missing keys
// and non-string values.
func str(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// firstStr returns the first non-empty value among the given keys.
func firstStr(row map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := str(row, k); s != "" {
			return s
		}
	}
	return ""
}

// sizeMB reads a byte-count field from a decoded JSON row and formats it as
// megabytes. Docker reports image size as a raw float64 (bytes), which
// json.Unmarshal into map[string]any leaves as-is; formatting it with %v
// prints ugly scientific notation (e.g. "4.903162e+07") for larger images.
func sizeMB(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}

	var numBytes float64
	switch n := v.(type) {
	case float64:
		numBytes = n
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			// Already human-formatted (e.g. "46.8MB") — pass through as-is.
			return n
		}
		numBytes = f
	default:
		return fmt.Sprintf("%v", v)
	}

	return fmt.Sprintf("%.1f MB", numBytes/(1024*1024))
}
