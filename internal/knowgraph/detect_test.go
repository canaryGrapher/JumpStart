package knowgraph

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDetectEmpty(t *testing.T) {
	dir := t.TempDir()
	info, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Present {
		t.Fatal("expected no graph")
	}
}

func TestDetectCodeReviewGraphDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".code-review-graph")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present || info.Kind != "code-review-graph" || info.RelPath != ".code-review-graph" {
		t.Fatalf("%+v", info)
	}
	if info.BrowseOK {
		t.Fatal("no db, browse should be false")
	}
}

func TestDetectWithDBAndSearch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".code-review-graph")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "graph.db")
	seedTinyGraph(t, dbPath)

	write(t, filepath.Join(root, "AGENTS.md"), "Use code-review-graph MCP tools.\n")

	info, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present || !info.BrowseOK {
		t.Fatalf("%+v", info)
	}
	if info.NodeCount < 1 || info.EdgeCount < 1 {
		t.Fatalf("counts: nodes=%d edges=%d", info.NodeCount, info.EdgeCount)
	}
	if len(info.AgentFiles) == 0 {
		t.Fatal("expected AGENTS.md mention")
	}

	hits, err := Search(root, "Hello", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected search hit")
	}

	nodes, err := ListNodes(root, "Function", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}

	detail, err := GetNode(root, "pkg.Hello")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Node.Name != "Hello" {
		t.Fatalf("%+v", detail)
	}
	if len(detail.Outgoing) == 0 && len(detail.Incoming) == 0 {
		t.Fatal("expected edges")
	}

	comms, err := ListCommunities(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(comms) == 0 {
		t.Fatal("expected community")
	}

	flows, err := ListFlows(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(flows) == 0 {
		t.Fatal("expected flow")
	}
}

func TestDetectEmptyRoot(t *testing.T) {
	_, err := Detect("")
	if err == nil {
		t.Fatal("expected error")
	}
}

func seedTinyGraph(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			qualified_name TEXT NOT NULL UNIQUE,
			file_path TEXT NOT NULL,
			line_start INTEGER,
			line_end INTEGER,
			language TEXT,
			parent_name TEXT,
			params TEXT,
			return_type TEXT,
			modifiers TEXT,
			is_test INTEGER DEFAULT 0,
			file_hash TEXT,
			extra TEXT DEFAULT '{}',
			updated_at REAL NOT NULL,
			signature TEXT,
			community_id INTEGER
		)`,
		`CREATE TABLE edges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			source_qualified TEXT NOT NULL,
			target_qualified TEXT NOT NULL,
			file_path TEXT NOT NULL,
			line INTEGER DEFAULT 0,
			extra TEXT DEFAULT '{}',
			updated_at REAL NOT NULL
		)`,
		`CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE communities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			level INTEGER NOT NULL DEFAULT 0,
			parent_id INTEGER,
			cohesion REAL NOT NULL DEFAULT 0.0,
			size INTEGER NOT NULL DEFAULT 0,
			dominant_language TEXT,
			description TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE flows (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			entry_point_id INTEGER NOT NULL,
			depth INTEGER NOT NULL,
			node_count INTEGER NOT NULL,
			file_count INTEGER NOT NULL,
			criticality REAL NOT NULL DEFAULT 0.0,
			path_json TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE VIRTUAL TABLE nodes_fts USING fts5(
			name, qualified_name, file_path, signature,
			tokenize='porter unicode61'
		)`,
		`INSERT INTO nodes (kind, name, qualified_name, file_path, line_start, line_end, language, signature, community_id, updated_at)
		 VALUES ('Function', 'Hello', 'pkg.Hello', 'pkg/hello.go', 1, 10, 'go', 'func Hello()', 1, 1)`,
		`INSERT INTO nodes (kind, name, qualified_name, file_path, line_start, line_end, language, signature, community_id, updated_at)
		 VALUES ('Function', 'World', 'pkg.World', 'pkg/world.go', 1, 5, 'go', 'func World()', 1, 1)`,
		`INSERT INTO edges (kind, source_qualified, target_qualified, file_path, line, updated_at)
		 VALUES ('CALLS', 'pkg.Hello', 'pkg.World', 'pkg/hello.go', 3, 1)`,
		`INSERT INTO communities (name, level, cohesion, size, dominant_language, description)
		 VALUES ('core', 0, 0.9, 2, 'go', 'core package')`,
		`INSERT INTO flows (name, entry_point_id, depth, node_count, file_count, criticality, path_json)
		 VALUES ('hello-world', 1, 2, 2, 2, 1.0, '[]')`,
		`INSERT INTO metadata (key, value) VALUES ('version', '1')`,
		`INSERT INTO nodes_fts (rowid, name, qualified_name, file_path, signature)
		 SELECT id, name, qualified_name, file_path, COALESCE(signature, '') FROM nodes`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%v\n%s", err, s)
		}
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
