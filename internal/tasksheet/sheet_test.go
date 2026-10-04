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
	}
	sprints := []model.Sprint{{ID: "s1", Name: "Sprint 1"}}

	for _, format := range []Format{FormatCSV, FormatExcel, FormatPDF, FormatPNG} {
		var buf bytes.Buffer
		if err := Encode(&buf, format, tasks, sprints, "Demo board"); err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("%s: empty output", format)
		}
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
}
