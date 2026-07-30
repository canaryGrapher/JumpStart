// Package codectx builds and queries a searchable index of a project's
// source code. The index powers project-aware answers in the story chat:
// "how do I start this?", "is ant design installed?", "where is routing
// configured?".
//
// Everything runs locally. Retrieval is lexical (BM25) rather than
// embedding-based so no extra model download is required.
package codectx

// Chunk is a contiguous slice of one source file. Chunks are the unit of
// retrieval: a query returns the highest-scoring chunks, which are then
// pasted into the model's system prompt.
type Chunk struct {
	Path      string `json:"path"`  // repo-relative, forward slashes
	Lang      string `json:"lang"`  // "go", "jsx", "md", ...
	Start     int    `json:"start"` // 1-based first line
	End       int    `json:"end"`   // inclusive last line
	Text      string `json:"text"`
	tokens    []string
	tokenFreq map[string]int
}

// FileEntry is one indexed file, used for the tree overview.
type FileEntry struct {
	Path  string `json:"path"`
	Lang  string `json:"lang"`
	Size  int64  `json:"size"`
	Lines int    `json:"lines"`
}

// Manifest summarises one dependency manifest found in the project.
type Manifest struct {
	Dir       string   `json:"dir"`       // repo-relative folder
	Manager   string   `json:"manager"`   // npm, go modules, pip, ...
	File      string   `json:"file"`      // package.json, go.mod, ...
	Install   string   `json:"install"`   // suggested install command
	Packages  []string `json:"packages"`  // "name@version"
	Installed int      `json:"installed"`
	Missing   int      `json:"missing"`
}

// RunScript is a runnable entry point discovered in a manifest.
type RunScript struct {
	Dir     string `json:"dir"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"source"`
}

// Overview is the cheap, always-included summary of a project. It is small
// enough to sit in every prompt regardless of the retrieved chunks.
type Overview struct {
	Name       string      `json:"name"`
	Root       string      `json:"root"`
	Languages  []LangCount `json:"languages"`
	Frameworks []string    `json:"frameworks"`
	Manifests  []Manifest  `json:"manifests"`
	Scripts    []RunScript `json:"scripts"`
	Tree       []string    `json:"tree"`    // condensed directory outline
	Readme     string      `json:"readme"`  // truncated README text
	EnvKeys    []string    `json:"envKeys"` // .env variable names (never values)
}

// LangCount is a per-language file tally.
type LangCount struct {
	Lang  string `json:"lang"`
	Files int    `json:"files"`
}

// Index is the persisted code context for one project.
type Index struct {
	ProjectID string      `json:"projectId"`
	Root      string      `json:"root"`
	BuiltAt   int64       `json:"builtAt"` // unix ms
	Overview  Overview    `json:"overview"`
	Files     []FileEntry `json:"files"`
	Chunks    []Chunk     `json:"chunks"`

	// Skipped counts files ignored for being binary, generated or oversized.
	Skipped int `json:"skipped"`

	bm25 *bm25Stats
}

// Stats is the lightweight status returned to the UI after a build.
type Stats struct {
	ProjectID  string `json:"projectId"`
	BuiltAt    int64  `json:"builtAt"`
	Files      int    `json:"files"`
	Chunks     int    `json:"chunks"`
	Skipped    int    `json:"skipped"`
	Frameworks int    `json:"frameworks"`
	Packages   int    `json:"packages"`
}

// Stats summarises the index for display.
func (ix *Index) Stats() Stats {
	pkgs := 0
	for _, m := range ix.Overview.Manifests {
		pkgs += len(m.Packages)
	}
	return Stats{
		ProjectID:  ix.ProjectID,
		BuiltAt:    ix.BuiltAt,
		Files:      len(ix.Files),
		Chunks:     len(ix.Chunks),
		Skipped:    ix.Skipped,
		Frameworks: len(ix.Overview.Frameworks),
		Packages:   pkgs,
	}
}
