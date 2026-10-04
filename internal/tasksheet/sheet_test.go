package tasksheet

import (
	"bytes"
	"testing"

	"devdeck/internal/model"
)

func TestEncodeFormats(t *testing.T) {
	tasks := []model.Task{
		{
			Title: "Wire auth", Type: "task", Status: "todo", Priority: "high",
			Labels: []string{"auth"}, Description: "Session cookies",
		},
		{
			Title: "Login story", Type: "story", Status: "backlog", Priority: "medium",
		},
		{
			Title: "Crash on save", Type: "bug", Status: "inprogress",
		},
		{
			Title: "Ship it", Type: "task", Status: "done",
		},
	}
	sprints := []model.Sprint{{ID: "s1", Name: "Sprint 1"}}

	for _, format := range []Format{FormatCSV, FormatExcel, FormatPDF, FormatPNG} {
		var buf bytes.Buffer
		if err := Encode(&buf, format, tasks, sprints, "Demo board", LayoutTable); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("%s: empty output", format)
		}
	}

	var boardBuf bytes.Buffer
	if err := Encode(&boardBuf, FormatPDF, tasks, sprints, "Demo board", LayoutBoard); err != nil {
		t.Fatalf("pdf board: %v", err)
	}
	if boardBuf.Len() == 0 {
		t.Fatal("pdf board: empty output")
	}
	// PDF magic
	if !bytes.HasPrefix(boardBuf.Bytes(), []byte("%PDF")) {
		t.Fatalf("pdf board: missing PDF header, got %q", boardBuf.Bytes()[:min(8, boardBuf.Len())])
	}
}

func TestParseFormat(t *testing.T) {
	if ParseFormat("excel") != FormatExcel {
		t.Fatal("excel")
	}
	if ParseFormat("image") != FormatPNG {
		t.Fatal("image")
	}
	if Ext(FormatPDF) != ".pdf" {
		t.Fatal("ext")
	}
	if ParseLayout("board") != LayoutBoard {
		t.Fatal("board layout")
	}
	if ParseLayout("") != LayoutTable {
		t.Fatal("default layout")
	}
}
