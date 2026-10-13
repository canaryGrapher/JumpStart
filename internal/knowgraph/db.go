package knowgraph

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	maxSearchLimit = 100
	maxListLimit   = 200
	maxEdgeLimit   = 200
	maxFlowLimit   = 100
)

var errEmptyRoot = fmt.Errorf("project root is empty")
var errNoGraph = fmt.Errorf("no knowledge graph found")
var errNoBrowse = fmt.Errorf("knowledge graph database is not readable")

type dbStats struct {
	NodeCount      int64
	EdgeCount      int64
	CommunityCount int64
	FlowCount      int64
	NodesByKind    map[string]int64
	EdgesByKind    map[string]int64
	Metadata       map[string]string
}

func openRO(dbPath string) (*sql.DB, error) {
	// URI form with mode=ro; immutable=1 avoids lock contention with writers.
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?mode=ro&_pragma=query_only(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func resolveDB(projectRoot string) (dbPath string, err error) {
	info, err := Detect(projectRoot)
	if err != nil {
		return "", err
	}
	if !info.Present {
		return "", errNoGraph
	}
	if info.DBRelPath == "" {
		return "", errNoBrowse
	}
	absRoot, err := absProject(projectRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(absRoot, filepath.FromSlash(info.DBRelPath)), nil
}

func readStats(dbPath string) (dbStats, error) {
	db, err := openRO(dbPath)
	if err != nil {
		return dbStats{}, err
	}
	defer db.Close()

	out := dbStats{
		NodesByKind: map[string]int64{},
		EdgesByKind: map[string]int64{},
		Metadata:    map[string]string{},
	}

	_ = db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&out.NodeCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&out.EdgeCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM communities`).Scan(&out.CommunityCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM flows`).Scan(&out.FlowCount)

	if rows, err := db.Query(`SELECT kind, COUNT(*) FROM nodes GROUP BY kind`); err == nil {
		for rows.Next() {
			var k string
			var n int64
			if rows.Scan(&k, &n) == nil {
				out.NodesByKind[k] = n
			}
		}
		rows.Close()
	}
	if rows, err := db.Query(`SELECT kind, COUNT(*) FROM edges GROUP BY kind`); err == nil {
		for rows.Next() {
			var k string
			var n int64
			if rows.Scan(&k, &n) == nil {
				out.EdgesByKind[k] = n
			}
		}
		rows.Close()
	}
	if rows, err := db.Query(`SELECT key, value FROM metadata`); err == nil {
		for rows.Next() {
			var k, v string
			if rows.Scan(&k, &v) == nil {
				out.Metadata[k] = v
			}
		}
		rows.Close()
	}
	return out, nil
}

// Search finds nodes by FTS or LIKE fallback.
func Search(projectRoot, query string, limit int) ([]SearchHit, error) {
	dbPath, err := resolveDB(projectRoot)
	if err != nil {
		return nil, err
	}
	limit = clamp(limit, 1, maxSearchLimit)
	q := strings.TrimSpace(query)
	if q == "" {
		return []SearchHit{}, nil
	}

	db, err := openRO(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	hits, err := searchFTS(db, q, limit)
	if err == nil {
		return hits, nil
	}
	return searchLike(db, q, limit)
}

func searchFTS(db *sql.DB, q string, limit int) ([]SearchHit, error) {
	// Escape FTS special chars lightly; quote as phrase if spaces.
	fts := sanitizeFTS(q)
	rows, err := db.Query(`
		SELECT n.id, n.kind, n.name, n.qualified_name, n.file_path,
		       COALESCE(n.line_start, 0), COALESCE(n.line_end, 0),
		       COALESCE(n.language, ''), COALESCE(n.parent_name, ''),
		       COALESCE(n.signature, ''), COALESCE(n.community_id, 0),
		       COALESCE(n.is_test, 0)
		FROM nodes_fts f
		JOIN nodes n ON n.rowid = f.rowid
		WHERE nodes_fts MATCH ?
		LIMIT ?`, fts, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func searchLike(db *sql.DB, q string, limit int) ([]SearchHit, error) {
	pat := "%" + escapeLike(q) + "%"
	rows, err := db.Query(`
		SELECT id, kind, name, qualified_name, file_path,
		       COALESCE(line_start, 0), COALESCE(line_end, 0),
		       COALESCE(language, ''), COALESCE(parent_name, ''),
		       COALESCE(signature, ''), COALESCE(community_id, 0),
		       COALESCE(is_test, 0)
		FROM nodes
		WHERE name LIKE ? ESCAPE '\'
		   OR qualified_name LIKE ? ESCAPE '\'
		   OR file_path LIKE ? ESCAPE '\'
		ORDER BY name
		LIMIT ?`, pat, pat, pat, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func scanHits(rows *sql.Rows) ([]SearchHit, error) {
	var out []SearchHit
	for rows.Next() {
		var h SearchHit
		var isTest int
		if err := rows.Scan(
			&h.ID, &h.Kind, &h.Name, &h.QualifiedName, &h.FilePath,
			&h.LineStart, &h.LineEnd, &h.Language, &h.ParentName,
			&h.Signature, &h.CommunityID, &isTest,
		); err != nil {
			return nil, err
		}
		h.IsTest = isTest != 0
		out = append(out, h)
	}
	if out == nil {
		out = []SearchHit{}
	}
	return out, rows.Err()
}

// ListNodes returns a page of nodes, optionally filtered by kind.
func ListNodes(projectRoot, kind string, limit, offset int) ([]Node, error) {
	dbPath, err := resolveDB(projectRoot)
	if err != nil {
		return nil, err
	}
	limit = clamp(limit, 1, maxListLimit)
	if offset < 0 {
		offset = 0
	}

	db, err := openRO(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	kind = strings.TrimSpace(kind)
	var rows *sql.Rows
	if kind == "" {
		rows, err = db.Query(`
			SELECT id, kind, name, qualified_name, file_path,
			       COALESCE(line_start, 0), COALESCE(line_end, 0),
			       COALESCE(language, ''), COALESCE(parent_name, ''),
			       COALESCE(signature, ''), COALESCE(community_id, 0),
			       COALESCE(is_test, 0)
			FROM nodes
			ORDER BY kind, name
			LIMIT ? OFFSET ?`, limit, offset)
	} else {
		rows, err = db.Query(`
			SELECT id, kind, name, qualified_name, file_path,
			       COALESCE(line_start, 0), COALESCE(line_end, 0),
			       COALESCE(language, ''), COALESCE(parent_name, ''),
			       COALESCE(signature, ''), COALESCE(community_id, 0),
			       COALESCE(is_test, 0)
			FROM nodes
			WHERE kind = ?
			ORDER BY name
			LIMIT ? OFFSET ?`, kind, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if out == nil {
		out = []Node{}
	}
	return out, rows.Err()
}

// GetNode loads one node by qualified name plus edges.
func GetNode(projectRoot, qualifiedName string) (NodeDetail, error) {
	dbPath, err := resolveDB(projectRoot)
	if err != nil {
		return NodeDetail{}, err
	}
	qn := strings.TrimSpace(qualifiedName)
	if qn == "" {
		return NodeDetail{}, fmt.Errorf("qualified name is empty")
	}

	db, err := openRO(dbPath)
	if err != nil {
		return NodeDetail{}, err
	}
	defer db.Close()

	row := db.QueryRow(`
		SELECT id, kind, name, qualified_name, file_path,
		       COALESCE(line_start, 0), COALESCE(line_end, 0),
		       COALESCE(language, ''), COALESCE(parent_name, ''),
		       COALESCE(signature, ''), COALESCE(community_id, 0),
		       COALESCE(is_test, 0)
		FROM nodes WHERE qualified_name = ?`, qn)
	node, err := scanNodeRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return NodeDetail{}, fmt.Errorf("node not found")
		}
		return NodeDetail{}, err
	}

	out := NodeDetail{Node: node, Outgoing: []Edge{}, Incoming: []Edge{}}
	if edges, err := listEdges(db, `source_qualified = ?`, qn); err == nil {
		out.Outgoing = edges
	}
	if edges, err := listEdges(db, `target_qualified = ?`, qn); err == nil {
		out.Incoming = edges
	}
	return out, nil
}

func listEdges(db *sql.DB, where string, arg string) ([]Edge, error) {
	rows, err := db.Query(`
		SELECT id, kind, source_qualified, target_qualified, file_path, COALESCE(line, 0)
		FROM edges WHERE `+where+`
		ORDER BY kind, id
		LIMIT ?`, arg, maxEdgeLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.ID, &e.Kind, &e.SourceQualified, &e.TargetQualified, &e.FilePath, &e.Line); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if out == nil {
		out = []Edge{}
	}
	return out, rows.Err()
}

// ListCommunities returns community clusters.
func ListCommunities(projectRoot string) ([]Community, error) {
	dbPath, err := resolveDB(projectRoot)
	if err != nil {
		return nil, err
	}
	db, err := openRO(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, name, level, COALESCE(parent_id, 0), cohesion, size,
		       COALESCE(dominant_language, ''), COALESCE(description, '')
		FROM communities
		ORDER BY cohesion DESC, size DESC
		LIMIT ?`, maxListLimit)
	if err != nil {
		// Table may not exist on older graphs.
		return []Community{}, nil
	}
	defer rows.Close()
	var out []Community
	for rows.Next() {
		var c Community
		if err := rows.Scan(&c.ID, &c.Name, &c.Level, &c.ParentID, &c.Cohesion, &c.Size, &c.DominantLanguage, &c.Description); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if out == nil {
		out = []Community{}
	}
	return out, rows.Err()
}

// ListFlows returns execution flows ordered by criticality.
func ListFlows(projectRoot string, limit int) ([]Flow, error) {
	dbPath, err := resolveDB(projectRoot)
	if err != nil {
		return nil, err
	}
	limit = clamp(limit, 1, maxFlowLimit)
	db, err := openRO(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, name, entry_point_id, depth, node_count, file_count, criticality
		FROM flows
		ORDER BY criticality DESC, node_count DESC
		LIMIT ?`, limit)
	if err != nil {
		return []Flow{}, nil
	}
	defer rows.Close()
	var out []Flow
	for rows.Next() {
		var f Flow
		if err := rows.Scan(&f.ID, &f.Name, &f.EntryPointID, &f.Depth, &f.NodeCount, &f.FileCount, &f.Criticality); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if out == nil {
		out = []Flow{}
	}
	return out, rows.Err()
}

func scanNode(rows *sql.Rows) (Node, error) {
	var n Node
	var isTest int
	err := rows.Scan(
		&n.ID, &n.Kind, &n.Name, &n.QualifiedName, &n.FilePath,
		&n.LineStart, &n.LineEnd, &n.Language, &n.ParentName,
		&n.Signature, &n.CommunityID, &isTest,
	)
	n.IsTest = isTest != 0
	return n, err
}

func scanNodeRow(row *sql.Row) (Node, error) {
	var n Node
	var isTest int
	err := row.Scan(
		&n.ID, &n.Kind, &n.Name, &n.QualifiedName, &n.FilePath,
		&n.LineStart, &n.LineEnd, &n.Language, &n.ParentName,
		&n.Signature, &n.CommunityID, &isTest,
	)
	n.IsTest = isTest != 0
	return n, err
}

func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func sanitizeFTS(q string) string {
	// Wrap as a phrase if it contains spaces; strip quotes.
	q = strings.ReplaceAll(q, `"`, "")
	q = strings.TrimSpace(q)
	if strings.ContainsAny(q, " \t") {
		return `"` + q + `"`
	}
	return q + "*"
}
