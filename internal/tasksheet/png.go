package tasksheet

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"devdeck/internal/model"
)

func encodePNG(w io.Writer, tasks []model.Task, sprints []model.Sprint, title string) error {
	const (
		pad    = 24
		rowH   = 22
		headH  = 28
		titleH = 52
	)
	colW := []int{260, 64, 88, 64, 130, 96, 96, 56, 300}
	width := pad * 2
	for _, c := range colW {
		width += c
	}
	rows := len(tasks)
	if rows == 0 {
		rows = 1
	}
	// Cap image height so a huge board still exports.
	maxRows := 80
	omitted := 0
	if rows > maxRows {
		omitted = rows - maxRows
		rows = maxRows
	}
	height := pad*2 + titleH + headH + rows*rowH
	if omitted > 0 {
		height += rowH
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{R: 250, G: 250, B: 252, A: 255}
	ink := color.RGBA{R: 28, G: 28, B: 30, A: 255}
	dim := color.RGBA{R: 110, G: 110, B: 115, A: 255}
	lineCol := color.RGBA{R: 210, G: 210, B: 215, A: 255}
	headBg := color.RGBA{R: 232, G: 232, B: 237, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)

	if title == "" {
		title = "Task sheet"
	}
	drawText(img, pad, pad+22, title, ink, true)
	drawText(img, pad, pad+40, fmt.Sprintf("%d tasks", len(tasks)), dim, false)

	y := pad + titleH
	fillRect(img, pad, y, width-pad*2, headH, headBg)
	x := pad
	for i, h := range headers {
		drawText(img, x+6, y+18, h, ink, true)
		x += colW[i]
	}
	hline(img, pad, y+headH-1, width-pad*2, lineCol)

	y += headH
	shown := tasks
	if len(shown) > maxRows {
		shown = shown[:maxRows]
	}
	if len(shown) == 0 {
		drawText(img, pad+6, y+16, "No tasks match the current filters.", dim, false)
	}
	for _, t := range shown {
		vals := rowValues(t, sprints)
		x = pad
		hline(img, pad, y+rowH-1, width-pad*2, lineCol)
		for i, v := range vals {
			drawText(img, x+6, y+15, clipWidth(v, colW[i]), ink, false)
			x += colW[i]
		}
		y += rowH
	}
	if omitted > 0 {
		drawText(img, pad+6, y+15, fmt.Sprintf("…and %d more tasks", omitted), dim, false)
	}

	return png.Encode(w, img)
}

func drawText(img *image.RGBA, x, y int, s string, c color.Color, bold bool) {
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	if bold {
		d.DrawString(s)
		d.Dot = fixed.P(x+1, y)
	}
	d.DrawString(s)
}

func clipWidth(s string, px int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	// basicfont is ~7px wide per glyph.
	max := px / 7
	if max < 4 {
		max = 4
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
