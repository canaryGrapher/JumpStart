// Package knowgraph detects and reads AI knowledge-graph stores kept in a
// project (typically .code-review-graph/graph.db) so JumpStart can show a
// Graph tab with summary stats and browse/search.
package knowgraph

// Info describes a detected knowledge graph and high-level stats.
type Info struct {
	Present       bool           `json:"present"`
	Kind          string         `json:"kind"`          // code-review-graph | knowledge-graph
	RelPath       string         `json:"relPath"`       // e.g. ".code-review-graph"
	DBRelPath     string         `json:"dbRelPath"`     // e.g. ".code-review-graph/graph.db"
	DBSize        int64          `json:"dbSize"`
	DBMTime       string         `json:"dbMTime"` // RFC3339, empty if unknown
	BrowseOK      bool           `json:"browseOk"`
	BrowseError   string         `json:"browseError,omitempty"`
	NodeCount     int64          `json:"nodeCount"`
	EdgeCount     int64          `json:"edgeCount"`
	CommunityCount int64         `json:"communityCount"`
	FlowCount     int64          `json:"flowCount"`
	NodesByKind   map[string]int64 `json:"nodesByKind,omitempty"`
	EdgesByKind   map[string]int64 `json:"edgesByKind,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	AgentFiles    []string       `json:"agentFiles,omitempty"` // related rule files mentioning the graph
}

// Node is one graph node (function, class, file, …).
type Node struct {
	ID            int64  `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualifiedName"`
	FilePath      string `json:"filePath"`
	LineStart     int    `json:"lineStart"`
	LineEnd       int    `json:"lineEnd"`
	Language      string `json:"language,omitempty"`
	ParentName    string `json:"parentName,omitempty"`
	Signature     string `json:"signature,omitempty"`
	CommunityID   int64  `json:"communityId,omitempty"`
	IsTest        bool   `json:"isTest"`
}

// Edge is a directed relationship between two qualified names.
type Edge struct {
	ID             int64  `json:"id"`
	Kind           string `json:"kind"`
	SourceQualified string `json:"sourceQualified"`
	TargetQualified string `json:"targetQualified"`
	FilePath       string `json:"filePath"`
	Line           int    `json:"line"`
}

// NodeDetail is a node plus its immediate edges.
type NodeDetail struct {
	Node     Node   `json:"node"`
	Outgoing []Edge `json:"outgoing"`
	Incoming []Edge `json:"incoming"`
}

// Community is a community cluster from the graph.
type Community struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Level            int     `json:"level"`
	ParentID         int64   `json:"parentId,omitempty"`
	Cohesion         float64 `json:"cohesion"`
	Size             int     `json:"size"`
	DominantLanguage string  `json:"dominantLanguage,omitempty"`
	Description      string  `json:"description,omitempty"`
}

// Flow is a named execution path through the graph.
type Flow struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	EntryPointID int64  `json:"entryPointId"`
	Depth       int     `json:"depth"`
	NodeCount   int     `json:"nodeCount"`
	FileCount   int     `json:"fileCount"`
	Criticality float64 `json:"criticality"`
}

// SearchHit is one FTS / name search result.
type SearchHit struct {
	Node
	Rank float64 `json:"rank,omitempty"`
}
