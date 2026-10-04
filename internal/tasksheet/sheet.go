// Package tasksheet renders a filtered Kanban board as Excel, PDF, or PNG
// so users can download a printable / shareable task sheet.
package tasksheet

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"strings"

	"github.com/phpdave11/gofpdf"
	"github.com/xuri/excelize/v2"

	"devdeck/internal/model"
	"devdeck/internal/taskcsv"
)

// Format is one downloadable sheet type.
type Format string

const (
	FormatCSV   Format = "csv"
	FormatExcel Format = "xlsx"
	FormatPDF   Format = "pdf"
	FormatPNG   Format = "png"
)

// Layout controls how visual formats (PDF) arrange tasks.
type Layout string

const (
	LayoutTable Layout = "table"
	LayoutBoard Layout = "board"
)

// ParseFormat maps a UI string onto Format. Unknown values default to Excel.
func ParseFormat(s string) Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "csv":
		return FormatCSV
	case "pdf":
		return FormatPDF
	case "png", "image":
		return FormatPNG
	default:
		return FormatExcel
	}
}

// ParseLayout maps a UI string onto Layout. Unknown values default to table.
func ParseLayout(s string) Layout {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "board", "kanban":
		return LayoutBoard
	default:
		return LayoutTable
	}
}

// Ext returns the file extension (with dot) for f.
func Ext(f Format) string {
	switch f {
	case FormatCSV:
		return ".csv"
	case FormatPDF:
		return ".pdf"
	case FormatPNG:
		return ".png"
	default:
		return ".xlsx"
	}
}

// ContentType is a best-effort MIME type for f.
func ContentType(f Format) string {
	switch f {
	case FormatCSV:
		return "text/csv"
	case FormatPDF:
		return "application/pdf"
	case FormatPNG:
		return "image/png"
	default:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
}

var headers = []string{
	"Title", "Type", "Status", "Priority", "Labels", "Assignee",
	"Sprint", "Story points", "Description",
}

// Encode writes tasks in the requested format. sprints supplies names for
// the Sprint column. layout is honoured for PDF (table vs kanban board);
// other formats ignore it.
func Encode(w io.Writer, format Format, tasks []model.Task, sprints []model.Sprint, title string, layout Layout) error {
	switch format {
	case FormatCSV:
		return taskcsv.Encode(w, tasks, sprints, nil)
	case FormatPDF:
		if layout == LayoutBoard {
			return encodePDFBoard(w, tasks, title)
		}
		return encodePDF(w, tasks, sprints, title)
	case FormatPNG:
		return encodePNG(w, tasks, sprints, title)
	default:
		return encodeExcel(w, tasks, sprints, title)
	}
}

func encodeExcel(w io.Writer, tasks []model.Task, sprints []model.Sprint, title string) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Tasks"
	_ = f.SetSheetName("Sheet1", sheet)
	if title == "" {
		title = "Task sheet"
	}
	_ = f.SetCellValue(sheet, "A1", title)
	_ = f.MergeCell(sheet, "A1", "I1")
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	_ = f.SetCellStyle(sheet, "A1", "A1", styleTitle)

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(sheet, cell, h)
	}
	styleHead, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8E8ED"}, Pattern: 1},
	})
	_ = f.SetCellStyle(sheet, "A3", "I3", styleHead)

	for i, t := range tasks {
		row := i + 4
		vals := rowValues(t, sprints)
		for c, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	_ = f.SetColWidth(sheet, "A", "A", 36)
	_ = f.SetColWidth(sheet, "B", "D", 12)
	_ = f.SetColWidth(sheet, "E", "E", 20)
	_ = f.SetColWidth(sheet, "F", "H", 14)
	_ = f.SetColWidth(sheet, "I", "I", 48)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}

func encodePDF(w io.Writer, tasks []model.Task, sprints []model.Sprint, title string) error {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	if title == "" {
		title = "Task sheet"
	}
	pdf.Cell(0, 10, title)
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 9)
	pdf.Cell(0, 6, fmt.Sprintf("%d tasks", len(tasks)))
	pdf.Ln(10)

	cols := []float64{55, 18, 22, 18, 30, 25, 25, 16, 60}
	writePDFHeader := func() {
		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(232, 232, 237)
		for i, h := range headers {
			pdf.CellFormat(cols[i], 7, h, "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)
		pdf.SetFont("Arial", "", 8)
	}
	writePDFHeader()

	for _, t := range tasks {
		vals := rowValues(t, sprints)
		if pdf.GetY() > 185 {
			pdf.AddPage()
			writePDFHeader()
		}
		for i, v := range vals {
			pdf.CellFormat(cols[i], 6, truncate(v, 48), "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func rowValues(t model.Task, sprints []model.Sprint) []string {
	sprint := "Backlog"
	if t.SprintID != "" {
		sprint = t.SprintID
		for _, s := range sprints {
			if s.ID == t.SprintID {
				sprint = s.Name
				break
			}
		}
	}
	typ := t.Type
	if typ == "" {
		typ = "task"
	}
	status := t.Status
	if status == "" {
		status = "todo"
	}
	return []string{
		t.Title,
		typ,
		status,
		t.Priority,
		strings.Join(t.Labels, ", "),
		t.Assignee,
		sprint,
		fmt.Sprintf("%d", t.StoryPoints),
		strings.TrimSpace(t.Description),
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	// ASCII ellipsis - core PDF fonts cannot draw the unicode glyph.
	return s[:n-1] + "..."
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx >= 0 && yy >= 0 && xx < img.Bounds().Max.X && yy < img.Bounds().Max.Y {
				img.Set(xx, yy, c)
			}
		}
	}
}

func hline(img *image.RGBA, x, y, w int, c color.Color) {
	for xx := x; xx < x+w; xx++ {
		if xx >= 0 && y >= 0 && xx < img.Bounds().Max.X && y < img.Bounds().Max.Y {
			img.Set(xx, y, c)
		}
	}
}
