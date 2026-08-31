package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// maxIconBytes caps the source image PickIconImage will read. Rasterizable
// formats get downscaled well below this regardless (see iconTargetPx); the
// cap mainly guards formats we pass through unchanged (SVG, or anything the
// decoder below doesn't recognize).
const maxIconBytes = 2 * 1024 * 1024

// iconTargetPx is the stored resolution for a raster icon: plenty sharp for
// the 40px header/sidebar use and any foreseeable HiDPI display, while
// keeping the data URI small since it lives inline in the project's config.
const iconTargetPx = 256

// iconMimeTypes maps a picked file's extension to the MIME type used when a
// file is passed through unresized (SVG, or a raster format the decoder
// below doesn't recognize). Anything else falls back to image/png.
var iconMimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
}

// PickIconImage opens a native image picker and returns the chosen file
// encoded as a data URI, ready to store on Project.Icon and render directly
// in an <img> src. defaultDir, when non-empty, is where the dialog opens
// (typically the current project's root). Returns "" (no error) if the
// user cancels.
//
// A raster image (anything image.Decode recognizes: PNG/JPEG/GIF, not SVG)
// is center-cropped to a square and downscaled to iconTargetPx before being
// re-encoded as PNG, so a multi-megapixel photo doesn't get embedded at
// full size. Formats we can't decode (SVG, WebP — the stdlib has no WebP
// decoder) are stored as-is; they're capped by maxIconBytes instead.
func (a *App) PickIconImage(defaultDir string) (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select project icon",
		DefaultDirectory: defaultDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Images (*.png, *.jpg, *.jpeg, *.gif, *.webp, *.svg)", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.svg"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.Size() > maxIconBytes {
		return "", fmt.Errorf("image is too large (max %dMB)", maxIconBytes/(1024*1024))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if resized, ok := resizeIcon(data); ok {
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(resized), nil
	}

	mime := iconMimeTypes[strings.ToLower(filepath.Ext(path))]
	if mime == "" {
		mime = "image/png"
	}
	return fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data)), nil
}

// resizeIcon center-crops src to a square and downscales it to
// iconTargetPx, returning the re-encoded PNG bytes. ok is false when src
// isn't a format the stdlib can decode (SVG, WebP), in which case the
// caller falls back to storing the original bytes.
func resizeIcon(src []byte) (out []byte, ok bool) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, false
	}

	square := cropToSquare(img)
	small := downscale(square, iconTargetPx)

	var buf bytes.Buffer
	if err := png.Encode(&buf, small); err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// cropToSquare returns the largest centered square crop of img.
func cropToSquare(img image.Image) image.Image {
	b := img.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	origin := image.Pt(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2)
	rect := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(side, side))}

	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
	return dst
}

// downscale nearest-neighbor resizes a square image down to target x
// target. It never scales up: an icon smaller than target is returned as-is.
func downscale(img image.Image, target int) image.Image {
	side := img.Bounds().Dx()
	if side <= target {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, target, target))
	scale := float64(side) / float64(target)
	b := img.Bounds()
	for y := 0; y < target; y++ {
		sy := b.Min.Y + int(float64(y)*scale)
		for x := 0; x < target; x++ {
			sx := b.Min.X + int(float64(x)*scale)
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	return dst
}
