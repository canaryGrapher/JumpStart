package arch

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	relFile  = ".jumpstart/architecture.xml"
	maxBytes = 8 << 20 // 8 MiB
)

var errEmptyRoot = fmt.Errorf("project root is empty")

// xmlDocument mirrors the on-disk architecture.xml schema.
type xmlDocument struct {
	XMLName     xml.Name    `xml:"architecture"`
	Version     string      `xml:"version,attr"`
	GeneratedAt string      `xml:"generatedAt,attr"`
	Project     xmlProject  `xml:"project"`
	Overview    xmlOverview `xml:"overview"`
	Sections    []xmlSection `xml:"section"`
}

type xmlProject struct {
	Name    string `xml:"name,attr"`
	Summary string `xml:"summary,attr"`
}

type xmlOverview struct {
	Description string `xml:"description,attr"`
	Stack       string `xml:"stack,attr"`
	Principles  string `xml:"principles,attr"`
	// Element body also accepted as description fallback.
	Body string `xml:",chardata"`
}

type xmlSection struct {
	ID        string       `xml:"id,attr"`
	Title     string       `xml:"title,attr"`
	Kind      string       `xml:"kind,attr"`
	Summary   string       `xml:"summary"`
	Nodes     []xmlNode    `xml:"node"`
	Edges     []xmlEdge    `xml:"edge"`
	Entities  []xmlEntity  `xml:"entity"`
	Relations []xmlRelation `xml:"relation"`
	Tables    []xmlTable   `xml:"table"`
	Charts    []xmlChart   `xml:"chart"`
	Steps     []xmlStep    `xml:"step"`
}

type xmlNode struct {
	ID          string `xml:"id,attr"`
	Label       string `xml:"label,attr"`
	Kind        string `xml:"kind,attr"`
	Description string `xml:"description,attr"`
}

type xmlEdge struct {
	From        string `xml:"from,attr"`
	To          string `xml:"to,attr"`
	Label       string `xml:"label,attr"`
	Description string `xml:"description,attr"`
}

type xmlEntity struct {
	Name        string      `xml:"name,attr"`
	Description string      `xml:"description,attr"`
	Columns     []xmlColumn `xml:"column"`
}

type xmlColumn struct {
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	PK          string `xml:"pk,attr"`
	Description string `xml:"description,attr"`
}

type xmlRelation struct {
	From        string `xml:"from,attr"`
	To          string `xml:"to,attr"`
	Type        string `xml:"type,attr"`
	Description string `xml:"description,attr"`
}

type xmlTable struct {
	Title   string    `xml:"title,attr"`
	Headers string    `xml:"headers,attr"` // pipe-separated
	Rows    []xmlRow  `xml:"row"`
}

type xmlRow struct {
	Cells string `xml:"cells,attr"` // pipe-separated
}

type xmlChart struct {
	Type   string      `xml:"type,attr"`
	Title  string      `xml:"title,attr"`
	Points []xmlPoint  `xml:"point"`
}

type xmlPoint struct {
	Label string `xml:"label,attr"`
	Value string `xml:"value,attr"`
}

type xmlStep struct {
	ID          string `xml:"id,attr"`
	Label       string `xml:"label,attr"`
	Description string `xml:"description,attr"`
}

func absRoot(projectRoot string) (string, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return "", errEmptyRoot
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func architecturePath(absRoot string) string {
	return filepath.Join(absRoot, filepath.FromSlash(relFile))
}

// Detect reports whether .jumpstart/architecture.xml exists and lists sections.
func Detect(projectRoot string) (Info, error) {
	root, err := absRoot(projectRoot)
	if err != nil {
		return Info{}, err
	}
	path := architecturePath(root)
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return Info{Present: false, Sections: []SectionSummary{}}, nil
	}
	doc, err := readFile(path)
	if err != nil {
		// Present but unreadable — still mark present so UI can show error.
		return Info{
			Present:  true,
			RelPath:  relFile,
			Sections: []SectionSummary{},
			Summary:  err.Error(),
		}, nil
	}
	return infoFromDoc(doc), nil
}

// Read loads and parses the full architecture document.
func Read(projectRoot string) (Document, error) {
	root, err := absRoot(projectRoot)
	if err != nil {
		return Document{}, err
	}
	path := architecturePath(root)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, fmt.Errorf("no architecture file at %s", relFile)
		}
		return Document{}, err
	}
	if st.IsDir() {
		return Document{}, fmt.Errorf("%s is a directory", relFile)
	}
	return readFile(path)
}

func readFile(path string) (Document, error) {
	st, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}
	if st.Size() > maxBytes {
		return Document{}, fmt.Errorf("architecture file larger than %d bytes", maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	return Parse(data)
}

// Parse unmarshals architecture XML bytes into a Document.
func Parse(data []byte) (Document, error) {
	var raw xmlDocument
	if err := xml.Unmarshal(data, &raw); err != nil {
		return Document{}, fmt.Errorf("invalid architecture XML: %w", err)
	}
	if raw.XMLName.Local != "" && raw.XMLName.Local != "architecture" {
		return Document{}, fmt.Errorf("root element must be <architecture>")
	}
	return toDocument(raw), nil
}

func infoFromDoc(doc Document) Info {
	info := Info{
		Present:     true,
		RelPath:     relFile,
		GeneratedAt: doc.GeneratedAt,
		ProjectName: doc.Project.Name,
		Summary:     doc.Project.Summary,
		Sections:    []SectionSummary{},
	}
	// Synthetic overview entry when overview has content.
	if doc.Overview.Description != "" || len(doc.Overview.Stack) > 0 || len(doc.Overview.Principles) > 0 {
		info.Sections = append(info.Sections, SectionSummary{
			ID:    "overview",
			Title: "Overview",
			Kind:  "overview",
		})
	}
	for _, s := range doc.Sections {
		info.Sections = append(info.Sections, SectionSummary{
			ID:    s.ID,
			Title: s.Title,
			Kind:  s.Kind,
		})
	}
	return info
}

func toDocument(raw xmlDocument) Document {
	ovDesc := strings.TrimSpace(raw.Overview.Description)
	if ovDesc == "" {
		ovDesc = strings.TrimSpace(raw.Overview.Body)
	}
	doc := Document{
		Version:     raw.Version,
		GeneratedAt: raw.GeneratedAt,
		Project: Project{
			Name:    raw.Project.Name,
			Summary: raw.Project.Summary,
		},
		Overview: Overview{
			Description: ovDesc,
			Stack:       splitCSV(raw.Overview.Stack),
			Principles:  splitCSV(raw.Overview.Principles),
		},
		Sections: []Section{},
	}
	for _, s := range raw.Sections {
		sec := Section{
			ID:        s.ID,
			Title:     s.Title,
			Kind:      s.Kind,
			Summary:   strings.TrimSpace(s.Summary),
			Nodes:     []Node{},
			Edges:     []Edge{},
			Entities:  []Entity{},
			Relations: []Relation{},
			Tables:    []Table{},
			Charts:    []Chart{},
			Steps:     []Step{},
		}
		for _, n := range s.Nodes {
			sec.Nodes = append(sec.Nodes, Node{
				ID: n.ID, Label: n.Label, Kind: n.Kind, Description: n.Description,
			})
		}
		for _, e := range s.Edges {
			sec.Edges = append(sec.Edges, Edge{
				From: e.From, To: e.To, Label: e.Label, Description: e.Description,
			})
		}
		for _, ent := range s.Entities {
			e := Entity{Name: ent.Name, Description: ent.Description, Columns: []Column{}}
			for _, c := range ent.Columns {
				e.Columns = append(e.Columns, Column{
					Name: c.Name, Type: c.Type, PK: truthy(c.PK), Description: c.Description,
				})
			}
			sec.Entities = append(sec.Entities, e)
		}
		for _, r := range s.Relations {
			sec.Relations = append(sec.Relations, Relation{
				From: r.From, To: r.To, Type: r.Type, Description: r.Description,
			})
		}
		for _, t := range s.Tables {
			tbl := Table{
				Title:   t.Title,
				Headers: splitPipe(t.Headers),
				Rows:    [][]string{},
			}
			for _, row := range t.Rows {
				tbl.Rows = append(tbl.Rows, splitPipe(row.Cells))
			}
			sec.Tables = append(sec.Tables, tbl)
		}
		for _, c := range s.Charts {
			ch := Chart{Type: c.Type, Title: c.Title, Points: []ChartPoint{}}
			for _, p := range c.Points {
				v, _ := strconv.ParseFloat(strings.TrimSpace(p.Value), 64)
				ch.Points = append(ch.Points, ChartPoint{Label: p.Label, Value: v})
			}
			sec.Charts = append(sec.Charts, ch)
		}
		for _, st := range s.Steps {
			sec.Steps = append(sec.Steps, Step{
				ID: st.ID, Label: st.Label, Description: st.Description,
			})
		}
		doc.Sections = append(doc.Sections, sec)
	}
	return doc
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitPipe(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}
