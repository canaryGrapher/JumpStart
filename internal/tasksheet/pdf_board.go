package tasksheet

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/phpdave11/gofpdf"

	"devdeck/internal/model"
)

// Kanban columns rendered in board-layout PDF exports. Order matches the
// in-app board (Backlog, To Do, In Progress, Testing, Done).
var boardColumns = []struct {
	id    string
	label string
}{
	{"backlog", "Backlog"},
	{"todo", "To Do"},
	{"inprogress", "In Progress"},
	{"testing", "Testing"},
	{"done", "Done"},
}

func encodePDFBoard(w io.Writer, tasks []model.Task, title string) error {
	pdf := gofpdf.New("L", "mm", "A4", "")
	if title == "" {
		title = "Task board"
	}
	pdf.SetTitle(title, false)
	// We paginate columns ourselves. Auto breaks scramble multi-column layout.
	pdf.SetAutoPageBreak(false, 0)

	byStatus := groupTasksByStatus(tasks)
	queues := make([][]model.Task, len(boardColumns))
	totals := make([]int, len(boardColumns))
	for i, col := range boardColumns {
		queues[i] = byStatus[col.id]
		totals[i] = len(queues[i])
	}

	page := 0
	for {
		before := 0
		for _, q := range queues {
			before += len(q)
		}
		if before == 0 && page > 0 {
			break
		}
		page++
		pdf.AddPage()
		drawBoardPage(pdf, title, len(tasks), queues, totals, page)
		after := 0
		for _, q := range queues {
			after += len(q)
		}
		if after == 0 {
			break
		}
		// Oversized card that cannot fit - drop it so we don't loop forever.
		if after == before {
			for i := range queues {
				if len(queues[i]) > 0 {
					queues[i] = queues[i][1:]
					break
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func drawBoardPage(pdf *gofpdf.Fpdf, title string, total int, queues [][]model.Task, totals []int, page int) {
	pageW, pageH := pdf.GetPageSize()
	margin := 12.0
	gutter := 5.0
	headerBand := 16.0
	colHeaderH := 9.0
	cardGap := 3.0
	footerY := pageH - 8
	colTop := margin + headerBand
	colBot := footerY - 4
	colH := colBot - colTop

	// Soft page wash
	pdf.SetFillColor(250, 250, 252)
	pdf.Rect(0, 0, pageW, pageH, "F")

	// Header
	pdf.SetTextColor(20, 20, 24)
	pdf.SetFont("Helvetica", "B", 15)
	pdf.SetXY(margin, margin-0.5)
	pdf.CellFormat(pageW-margin*2-36, 7, title, "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 102, 108)
	right := fmt.Sprintf("%d tasks", total)
	if page > 1 {
		right = fmt.Sprintf("Continued  -  p.%d", page)
	}
	pdf.SetXY(margin, margin-0.5)
	pdf.CellFormat(pageW-margin*2, 7, right, "", 0, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(130, 132, 138)
	pdf.SetXY(margin, margin+6.5)
	pdf.CellFormat(pageW-margin*2, 4, "Kanban board", "", 0, "L", false, 0, "")

	usable := pageW - margin*2
	colW := (usable - gutter*float64(len(boardColumns)-1)) / float64(len(boardColumns))

	for i, col := range boardColumns {
		x := margin + float64(i)*(colW+gutter)
		drawBoardColumn(pdf, col.label, totals[i], len(queues[i]), x, colTop, colW, colH, colHeaderH)

		y := colTop + colHeaderH + 3
		innerPad := 3.0
		cardW := colW - innerPad*2
		cardX := x + innerPad

		for len(queues[i]) > 0 {
			t := queues[i][0]
			h := boardCardHeight(pdf, t, cardW-7)
			if y+h > colBot-innerPad {
				break
			}
			drawBoardCard(pdf, t, cardX, y, cardW, h)
			y += h + cardGap
			queues[i] = queues[i][1:]
		}
	}

	// Footer
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(140, 142, 148)
	pdf.SetXY(margin, footerY-1)
	pdf.CellFormat(pageW-margin*2, 4, fmt.Sprintf("Page %d", page), "", 0, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

func drawBoardColumn(pdf *gofpdf.Fpdf, label string, total, remaining int, x, y, w, h, headerH float64) {
	// Column well - slightly cooler grey so white cards lift off it
	pdf.SetFillColor(232, 234, 239)
	roundedRect(pdf, x, y, w, h, 3, "F")

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(70, 74, 82)
	pdf.SetXY(x+4, y+2.4)
	pdf.CellFormat(w-18, 5, strings.ToUpper(label), "", 0, "L", false, 0, "")

	// Count pill
	count := total
	if remaining < total {
		// On continued pages show how many are left in this column.
		count = remaining
	}
	pill := fmt.Sprintf("%d", count)
	pdf.SetFont("Helvetica", "B", 8)
	pw := pdf.GetStringWidth(pill) + 5
	px := x + w - pw - 3.5
	pdf.SetFillColor(255, 255, 255)
	roundedRect(pdf, px, y+2, pw, 5.2, 2, "F")
	pdf.SetTextColor(70, 74, 82)
	pdf.SetXY(px, y+2.3)
	pdf.CellFormat(pw, 4.6, pill, "", 0, "C", false, 0, "")
}

func boardCardHeight(pdf *gofpdf.Fpdf, t model.Task, textW float64) float64 {
	padTop := 3.5
	padBot := 3.0
	lineH := 3.8
	metaH := 4.5
	pdf.SetFont("Helvetica", "B", 8.5)
	titleLines := wrapPDFText(pdf, cleanPDFText(t.Title), textW)
	if len(titleLines) == 0 {
		titleLines = []string{"Untitled"}
	}
	if len(titleLines) > 2 {
		titleLines = titleLines[:2]
	}
	h := padTop + float64(len(titleLines))*lineH + 1.5 + metaH + padBot
	if h < 16 {
		h = 16
	}
	return h
}

func drawBoardCard(pdf *gofpdf.Fpdf, t model.Task, x, y, w, h float64) {
	pad := 3.5
	accentW := 1.6

	// Card body with light stroke
	pdf.SetFillColor(255, 255, 255)
	pdf.SetDrawColor(214, 216, 222)
	roundedRect(pdf, x, y, w, h, 2.2, "FD")

	// Type accent
	r, g, b := typeAccent(t.Type)
	pdf.SetFillColor(r, g, b)
	pdf.Rect(x+0.4, y+2.2, accentW, h-4.4, "F")

	textW := w - pad*2 - accentW
	tx := x + pad + accentW
	ty := y + 3.2

	pdf.SetFont("Helvetica", "B", 8.5)
	pdf.SetTextColor(28, 30, 34)
	title := cleanPDFText(t.Title)
	if title == "" {
		title = "Untitled"
	}
	lines := wrapPDFText(pdf, title, textW)
	if len(lines) > 2 {
		lines = lines[:2]
		lines[1] = ellipsizeWidth(pdf, lines[1], textW)
	}
	for _, line := range lines {
		pdf.SetXY(tx, ty)
		pdf.CellFormat(textW, 3.8, line, "", 0, "L", false, 0, "")
		ty += 3.8
	}

	// Meta row: type · priority · assignee  (labels if room)
	metaY := y + h - 6.5
	drawCardMeta(pdf, t, tx, metaY, textW)
	pdf.SetTextColor(0, 0, 0)
}

func drawCardMeta(pdf *gofpdf.Fpdf, t model.Task, x, y, maxW float64) {
	pdf.SetFont("Helvetica", "", 7)
	cx := x

	typ := t.Type
	if typ == "" {
		typ = "task"
	}
	tr, tg, tb := typeAccent(typ)
	cx = drawMetaChip(pdf, strings.ToUpper(typ[:1])+typ[1:], tr, tg, tb, cx, y, maxW-(cx-x))

	if p := strings.TrimSpace(t.Priority); p != "" {
		pr, pg, pb := priorityColor(p)
		cx = drawMetaChip(pdf, p, pr, pg, pb, cx+1.5, y, maxW-(cx-x)-1.5)
	}

	rest := make([]string, 0, 2)
	if a := strings.TrimSpace(t.Assignee); a != "" {
		rest = append(rest, "@"+cleanPDFText(a))
	}
	if len(t.Labels) > 0 {
		rest = append(rest, cleanPDFText(strings.Join(t.Labels, ", ")))
	}
	if len(rest) == 0 {
		return
	}
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(120, 122, 128)
	label := strings.Join(rest, "  |  ")
	avail := maxW - (cx - x) - 2
	if avail < 8 {
		return
	}
	label = ellipsizeWidth(pdf, label, avail)
	pdf.SetXY(cx+2, y)
	pdf.CellFormat(avail, 3.5, label, "", 0, "L", false, 0, "")
}

func drawMetaChip(pdf *gofpdf.Fpdf, text string, r, g, b int, x, y, maxW float64) float64 {
	text = cleanPDFText(text)
	if text == "" || maxW < 8 {
		return x
	}
	pdf.SetFont("Helvetica", "B", 6.5)
	tw := pdf.GetStringWidth(text) + 3.2
	if tw > maxW {
		text = ellipsizeWidth(pdf, text, maxW-3.2)
		tw = pdf.GetStringWidth(text) + 3.2
	}
	// Tinted chip background
	pdf.SetFillColor(
		min255(r+180),
		min255(g+180),
		min255(b+180),
	)
	roundedRect(pdf, x, y, tw, 3.6, 1.2, "F")
	pdf.SetTextColor(r, g, b)
	pdf.SetXY(x, y+0.3)
	pdf.CellFormat(tw, 3.2, text, "", 0, "C", false, 0, "")
	return x + tw
}

func roundedRect(pdf *gofpdf.Fpdf, x, y, w, h, r float64, style string) {
	if r <= 0 || r*2 > w || r*2 > h {
		pdf.Rect(x, y, w, h, style)
		return
	}
	// Approximate rounded corners with a filled path.
	pdf.RoundedRect(x, y, w, h, r, "1234", style)
}

func priorityColor(p string) (int, int, int) {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "high":
		return 200, 60, 50
	case "medium":
		return 180, 120, 20
	case "low":
		return 70, 130, 80
	default:
		return 110, 112, 118
	}
}

func typeAccent(typ string) (int, int, int) {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "story":
		return 95, 92, 200
	case "bug":
		return 210, 65, 55
	default:
		return 40, 140, 200
	}
}

func groupTasksByStatus(tasks []model.Task) map[string][]model.Task {
	out := make(map[string][]model.Task, len(boardColumns))
	for _, col := range boardColumns {
		out[col.id] = nil
	}
	for _, t := range tasks {
		st := t.Status
		if st == "" {
			if t.Done {
				st = "done"
			} else {
				st = "todo"
			}
		}
		if _, ok := out[st]; !ok {
			st = "backlog"
		}
		out[st] = append(out[st], t)
	}
	return out
}

// cleanPDFText strips characters the core Helvetica face cannot draw.
func cleanPDFText(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "…", "...")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "–", "-")
	s = strings.ReplaceAll(s, "·", "-")
	s = strings.ReplaceAll(s, "“", "\"")
	s = strings.ReplaceAll(s, "”", "\"")
	s = strings.ReplaceAll(s, "‘", "'")
	s = strings.ReplaceAll(s, "’", "'")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\u00A0' {
			b.WriteByte(' ')
			continue
		}
		// Core PDF fonts are WinAnsi - keep printable Latin-1.
		if r < 32 || r > 255 {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func ellipsizeWidth(pdf *gofpdf.Fpdf, s string, maxW float64) string {
	s = cleanPDFText(s)
	if s == "" || pdf.GetStringWidth(s) <= maxW {
		return s
	}
	const ellipsis = "..."
	for len(s) > 0 {
		s = s[:len(s)-1]
		// Avoid chopping mid-space awkwardly
		s = strings.TrimRight(s, " ")
		if pdf.GetStringWidth(s+ellipsis) <= maxW {
			return s + ellipsis
		}
	}
	return ellipsis
}

// wrapPDFText splits s into lines that fit within maxW mm using the
// current font. Newlines in s become hard breaks.
func wrapPDFText(pdf *gofpdf.Fpdf, s string, maxW float64) []string {
	s = cleanPDFText(s)
	if s == "" {
		return nil
	}
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		cur := words[0]
		for _, word := range words[1:] {
			trial := cur + " " + word
			if pdf.GetStringWidth(trial) <= maxW {
				cur = trial
				continue
			}
			lines = append(lines, cur)
			cur = word
		}
		lines = append(lines, cur)
	}
	return lines
}

func min255(v int) int {
	if v > 245 {
		return 245
	}
	if v < 0 {
		return 0
	}
	return v
}
