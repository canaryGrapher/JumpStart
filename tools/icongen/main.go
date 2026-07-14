package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !insideRoundedSquare(x, y, 112) {
				img.SetRGBA(x, y, color.RGBA{})
				continue
			}
			t := float64(x+y) / float64(size*2)
			img.SetRGBA(x, y, blend(color.RGBA{0, 122, 255, 255}, color.RGBA{40, 205, 65, 255}, t))
		}
	}

	drawCircle(img, 512, 512, 292, color.RGBA{255, 255, 255, 42})
	drawCircle(img, 512, 512, 236, color.RGBA{255, 255, 255, 58})
	drawChevron(img)
	drawSpark(img, 718, 270, 54)

	out, err := os.Create("build/appicon.png")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}

func insideRoundedSquare(x, y, radius int) bool {
	min := 72
	max := size - min - 1
	if x >= min+radius && x <= max-radius {
		return y >= min && y <= max
	}
	if y >= min+radius && y <= max-radius {
		return x >= min && x <= max
	}
	cx := min + radius
	if x > max-radius {
		cx = max - radius
	}
	cy := min + radius
	if y > max-radius {
		cy = max - radius
	}
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= radius*radius
}

func blend(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: 255,
	}
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				over(img, x, y, c)
			}
		}
	}
}

func drawChevron(img *image.RGBA) {
	white := color.RGBA{255, 255, 255, 240}
	drawThickLine(img, 342, 518, 500, 674, 74, white)
	drawThickLine(img, 500, 674, 704, 338, 74, white)
	drawThickLine(img, 412, 344, 704, 344, 66, white)
	drawThickLine(img, 704, 344, 704, 598, 66, white)
}

func drawSpark(img *image.RGBA, cx, cy, r int) {
	c := color.RGBA{255, 255, 255, 230}
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := math.Abs(float64(x - cx))
			dy := math.Abs(float64(y - cy))
			if dx+dy <= float64(r) {
				over(img, x, y, c)
			}
		}
	}
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1, width int, c color.RGBA) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length2 := dx*dx + dy*dy
	r := float64(width) / 2
	minX, maxX := min(x0, x1)-width, max(x0, x1)+width
	minY, maxY := min(y0, y1)-width, max(y0, y1)+width
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x - x0)
			py := float64(y - y0)
			t := (px*dx + py*dy) / length2
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			nearestX := float64(x0) + t*dx
			nearestY := float64(y0) + t*dy
			if math.Hypot(float64(x)-nearestX, float64(y)-nearestY) <= r {
				over(img, x, y, c)
			}
		}
	}
}

func over(img *image.RGBA, x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= size || y >= size || c.A == 0 {
		return
	}
	dst := img.RGBAAt(x, y)
	a := float64(c.A) / 255
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*a + float64(dst.R)*(1-a)),
		G: uint8(float64(c.G)*a + float64(dst.G)*(1-a)),
		B: uint8(float64(c.B)*a + float64(dst.B)*(1-a)),
		A: 255,
	})
}
