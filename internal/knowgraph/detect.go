package knowgraph

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Known relative layouts that indicate an AI knowledge graph.
var markers = []struct {
	kind    string
	relDir  string
	dbName  string // empty = any *.db inside dir, or knowledge-graph.db at root
}{
	{kind: "code-review-graph", relDir: ".code-review-graph", dbName: "graph.db"},
	{kind: "knowledge-graph", relDir: ".knowledge-graph", dbName: "graph.db"},
}

const agentMentionNeedle1 = "code-review-graph"
const agentMentionNeedle2 = "knowledge graph"
const agentMentionNeedle3 = "knowledge-graph"

// Detect reports whether projectRoot contains a knowledge-graph store.
func Detect(projectRoot string) (Info, error) {
	absRoot, err := absProject(projectRoot)
	if err != nil {
		return Info{}, err
	}

	info := Info{
		Present:     false,
		NodesByKind: map[string]int64{},
		EdgesByKind: map[string]int64{},
		Metadata:    map[string]string{},
		AgentFiles:  findAgentMentions(absRoot),
	}

	for _, m := range markers {
		dir := filepath.Join(absRoot, m.relDir)
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			continue
		}
		info.Present = true
		info.Kind = m.kind
		info.RelPath = filepath.ToSlash(m.relDir)

		dbPath := filepath.Join(dir, m.dbName)
		if dbSt, err := os.Stat(dbPath); err == nil && !dbSt.IsDir() {
			info.DBRelPath = filepath.ToSlash(filepath.Join(m.relDir, m.dbName))
			info.DBSize = dbSt.Size()
			info.DBMTime = dbSt.ModTime().UTC().Format(time.RFC3339)
			fillStats(&info, dbPath)
			return info, nil
		}
		// Dir present but no named DB — still show the tab.
		return info, nil
	}

	// Root-level knowledge-graph.db
	rootDB := filepath.Join(absRoot, "knowledge-graph.db")
	if dbSt, err := os.Stat(rootDB); err == nil && !dbSt.IsDir() {
		info.Present = true
		info.Kind = "knowledge-graph"
		info.RelPath = "knowledge-graph.db"
		info.DBRelPath = "knowledge-graph.db"
		info.DBSize = dbSt.Size()
		info.DBMTime = dbSt.ModTime().UTC().Format(time.RFC3339)
		fillStats(&info, rootDB)
		return info, nil
	}

	// Agent files alone are not enough to show the tab — need a store.
	info.AgentFiles = nil
	return info, nil
}

func fillStats(info *Info, dbPath string) {
	stats, err := readStats(dbPath)
	if err != nil {
		info.BrowseOK = false
		info.BrowseError = err.Error()
		return
	}
	info.BrowseOK = true
	info.NodeCount = stats.NodeCount
	info.EdgeCount = stats.EdgeCount
	info.CommunityCount = stats.CommunityCount
	info.FlowCount = stats.FlowCount
	info.NodesByKind = stats.NodesByKind
	info.EdgesByKind = stats.EdgesByKind
	info.Metadata = stats.Metadata
}

func absProject(projectRoot string) (string, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", errEmptyRoot
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func findAgentMentions(absRoot string) []string {
	var hits []string
	candidates := []string{
		"AGENTS.md",
		"CLAUDE.md",
		"agents.md",
		"claude.md",
	}
	for _, rel := range candidates {
		path := filepath.Join(absRoot, rel)
		if fileMentionsGraph(path) {
			hits = append(hits, filepath.ToSlash(rel))
		}
	}
	// .cursor/rules/*.mdc and *.md
	rulesDir := filepath.Join(absRoot, ".cursor", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".mdc") && !strings.HasSuffix(lower, ".md") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(".cursor", "rules", name))
			if fileMentionsGraph(filepath.Join(rulesDir, name)) {
				hits = append(hits, rel)
			}
		}
	}
	return hits
}

func fileMentionsGraph(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Cap scan to 256 KiB.
	if len(data) > 256<<10 {
		data = data[:256<<10]
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, agentMentionNeedle1) ||
		strings.Contains(lower, agentMentionNeedle2) ||
		strings.Contains(lower, agentMentionNeedle3)
}
