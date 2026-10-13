// Package arch reads JumpStart architecture documents stored at
// .jumpstart/architecture.xml — a detailed visual map of a project's
// features, data flows, databases, and code structure.
package arch

// Info is a lightweight summary used to decide whether to show loaded UI.
type Info struct {
	Present     bool             `json:"present"`
	RelPath     string           `json:"relPath"`
	GeneratedAt string           `json:"generatedAt,omitempty"`
	ProjectName string           `json:"projectName,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	Sections    []SectionSummary `json:"sections"`
}

// SectionSummary is sidebar metadata for one architecture section.
type SectionSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"` // graph | flow | db | table | mixed | overview
}

// Document is the full parsed architecture file for the visual explorer.
type Document struct {
	Version     string    `json:"version"`
	GeneratedAt string    `json:"generatedAt,omitempty"`
	Project     Project   `json:"project"`
	Overview    Overview  `json:"overview"`
	Sections    []Section `json:"sections"`
}

// Project identifies the repo the document describes.
type Project struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

// Overview is the top-level narrative.
type Overview struct {
	Description string   `json:"description,omitempty"`
	Stack       []string `json:"stack,omitempty"`
	Principles  []string `json:"principles,omitempty"`
}

// Section is one sidebar pane of the architecture explorer.
type Section struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Kind        string     `json:"kind"`
	Summary     string     `json:"summary,omitempty"`
	Nodes       []Node     `json:"nodes,omitempty"`
	Edges       []Edge     `json:"edges,omitempty"`
	Entities    []Entity   `json:"entities,omitempty"`
	Relations   []Relation `json:"relations,omitempty"`
	Tables      []Table    `json:"tables,omitempty"`
	Charts      []Chart    `json:"charts,omitempty"`
	Steps       []Step     `json:"steps,omitempty"` // ordered flow steps
}

// Node is a graph/flow node.
type Node struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`
}

// Edge connects two nodes.
type Edge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// Entity is a database table / collection.
type Entity struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Columns     []Column `json:"columns,omitempty"`
}

// Column is one field on an entity.
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	PK          bool   `json:"pk,omitempty"`
	Description string `json:"description,omitempty"`
}

// Relation links two entities.
type Relation struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type,omitempty"` // 1:1, 1:N, N:M
	Description string `json:"description,omitempty"`
}

// Table is a markdown-like data table.
type Table struct {
	Title   string     `json:"title,omitempty"`
	Headers []string   `json:"headers,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`
}

// Chart is a simple bar or pie series.
type Chart struct {
	Type   string       `json:"type"` // bar | pie
	Title  string       `json:"title,omitempty"`
	Points []ChartPoint `json:"points,omitempty"`
}

// ChartPoint is one labeled value.
type ChartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// Step is one ordered stage in a data-flow diagram.
type Step struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}
