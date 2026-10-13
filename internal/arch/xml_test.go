package arch

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<architecture version="1" generatedAt="2026-10-13T00:00:00Z">
  <project name="Demo" summary="A demo app"/>
  <overview description="Local-first demo" stack="Go,React" principles="Thin bindings, events for push"/>
  <section id="code-arch" title="Code Architecture" kind="graph">
    <summary>Layered Wails app</summary>
    <node id="app" label="App" kind="service" description="Wails bindings"/>
    <node id="ui" label="Frontend" kind="ui" description="React UI"/>
    <edge from="ui" to="app" label="RPC" description="Wails calls"/>
    <table title="Packages" headers="Name|Role">
      <row cells="app.go|Bindings"/>
      <row cells="internal/store|Persistence"/>
    </table>
    <chart type="bar" title="LOC by area">
      <point label="Go" value="40"/>
      <point label="JS" value="60"/>
    </chart>
  </section>
  <section id="data-flow" title="Data Flow" kind="flow">
    <step id="s1" label="UI action" description="User clicks Start"/>
    <step id="s2" label="StartProcess" description="Go binding"/>
    <step id="s3" label="procman" description="Spawns subprocess"/>
  </section>
  <section id="db-arch" title="Database" kind="db">
    <entity name="projects" description="Saved projects">
      <column name="id" type="uuid" pk="true" description="Primary key"/>
      <column name="name" type="text" description="Display name"/>
    </entity>
    <entity name="tasks" description="Kanban tasks">
      <column name="id" type="uuid" pk="true"/>
      <column name="project_id" type="uuid"/>
    </entity>
    <relation from="tasks" to="projects" type="N:1" description="Task belongs to project"/>
  </section>
</architecture>
`

func TestDetectMissing(t *testing.T) {
	dir := t.TempDir()
	info, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Present {
		t.Fatal("expected absent")
	}
}

func TestDetectAndRead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".jumpstart")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "architecture.xml")
	if err := os.WriteFile(path, []byte(sampleXML), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Present || info.ProjectName != "Demo" {
		t.Fatalf("%+v", info)
	}
	if len(info.Sections) < 4 { // overview + 3 sections
		t.Fatalf("sections=%d %+v", len(info.Sections), info.Sections)
	}
	if info.Sections[0].ID != "overview" {
		t.Fatalf("first section should be overview, got %q", info.Sections[0].ID)
	}

	doc, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Project.Name != "Demo" || len(doc.Overview.Stack) != 2 {
		t.Fatalf("%+v", doc)
	}
	if len(doc.Sections) != 3 {
		t.Fatalf("sections=%d", len(doc.Sections))
	}
	code := doc.Sections[0]
	if len(code.Nodes) != 2 || len(code.Edges) != 1 {
		t.Fatalf("graph: %+v", code)
	}
	if len(code.Tables) != 1 || len(code.Tables[0].Rows) != 2 {
		t.Fatalf("tables: %+v", code.Tables)
	}
	if len(code.Charts) != 1 || code.Charts[0].Points[0].Value != 40 {
		t.Fatalf("charts: %+v", code.Charts)
	}
	db := doc.Sections[2]
	if len(db.Entities) != 2 || !db.Entities[0].Columns[0].PK {
		t.Fatalf("db: %+v", db)
	}
	if len(db.Relations) != 1 {
		t.Fatalf("relations: %+v", db.Relations)
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse([]byte(`<not-architecture/>`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEmptyRoot(t *testing.T) {
	_, err := Detect("")
	if err == nil {
		t.Fatal("expected error")
	}
}
