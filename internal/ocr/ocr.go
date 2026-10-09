// Package ocr reads searchable text out of attachments: plain text files
// directly, PDFs through pdftotext, and images through the OCR engine the
// user picked in Settings > Search (macOS Vision, Tesseract, an Ollama
// vision model, or off).
//
// Image and PDF text is slow to produce, so a Worker does it in the
// background and caches the result under <data dir>/search-text. Searches
// never wait for it; they report how many files are still pending.
package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"devdeck/internal/ai"
	"devdeck/internal/attachments"
	"devdeck/internal/model"
)

// Engines.
const (
	Vision    = "vision"
	Tesseract = "tesseract"
	Ollama    = "ollama"
	Off       = "off"
)

// MaxText caps how much text one attachment contributes to the index.
const MaxText = 512 * 1024

// ErrUnsupported means nothing searchable can be read from the file with
// the current tools and settings.
var ErrUnsupported = errors.New("no text can be read from this file")

// Config is the engine choice plus what each engine needs.
type Config struct {
	Engine      string
	OllamaHost  string
	OllamaModel string
}

// Kind classifies an attachment for extraction.
func Kind(a model.Attachment) string {
	switch {
	case attachments.IsText(a):
		return "text"
	case strings.EqualFold(a.Mime, "application/pdf") || strings.EqualFold(filepath.Ext(a.Name), ".pdf"):
		return "pdf"
	case attachments.IsImage(a.Mime):
		return "image"
	}
	return ""
}

// findTool looks on PATH and in the usual Homebrew locations, because an
// app started from Finder gets a minimal PATH.
func findTool(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// HelperPath finds the bundled Vision helper: next to the app binary
// (Contents/MacOS/jumpstart-ocr) or in <data dir>/bin.
func HelperPath(dataDir string) string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "jumpstart-ocr"))
	}
	candidates = append(candidates, filepath.Join(dataDir, "bin", "jumpstart-ocr"))
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

// Status says whether each engine can run here, for Settings.
type Status struct {
	Engine    string `json:"engine"`
	Available bool   `json:"available"`
	Detail    string `json:"detail"`
}

// Engines reports availability of every engine and of PDF text.
func Engines(dataDir string, cfg Config) []Status {
	vision := Status{Engine: Vision}
	if p := HelperPath(dataDir); p != "" {
		vision.Available, vision.Detail = true, "Built in (macOS Vision)."
	} else {
		vision.Detail = "The Vision helper is not installed in this build."
	}
	tess := Status{Engine: Tesseract}
	if p := findTool("tesseract"); p != "" {
		tess.Available, tess.Detail = true, p
	} else {
		tess.Detail = "Install with: brew install tesseract"
	}
	oll := Status{Engine: Ollama, Available: cfg.OllamaModel != ""}
	if oll.Available {
		oll.Detail = "Uses " + cfg.OllamaModel + "."
	} else {
		oll.Detail = "Pick a vision model (for example llava or qwen2.5vl)."
	}
	pdf := Status{Engine: "pdf"}
	if p := findTool("pdftotext"); p != "" {
		pdf.Available, pdf.Detail = true, p
	} else {
		pdf.Detail = "PDF text needs pdftotext: brew install poppler"
	}
	return []Status{vision, tess, oll, {Engine: Off, Available: true, Detail: "Images are not searched."}, pdf}
}

func clean(b []byte) string {
	if len(b) > MaxText {
		b = b[:MaxText]
	}
	s := string(b)
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, " ")
	}
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", " "))
}

// ReadText reads a plain-text attachment, capped at MaxText.
func ReadText(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, MaxText)
	n, _ := f.Read(buf)
	return clean(buf[:n]), nil
}

func runTool(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s: %s", filepath.Base(name), msg)
	}
	return clean(out.Bytes()), nil
}

// Extract produces the searchable text of one stored attachment file.
func Extract(ctx context.Context, dataDir string, cfg Config, path string, a model.Attachment) (string, error) {
	switch Kind(a) {
	case "text":
		return ReadText(path)
	case "pdf":
		tool := findTool("pdftotext")
		if tool == "" {
			return "", ErrUnsupported
		}
		return runTool(ctx, tool, "-q", "-enc", "UTF-8", path, "-")
	case "image":
		return Image(ctx, dataDir, cfg, path)
	}
	return "", ErrUnsupported
}

// Image runs the configured OCR engine on an image file.
func Image(ctx context.Context, dataDir string, cfg Config, path string) (string, error) {
	switch cfg.Engine {
	case Vision:
		helper := HelperPath(dataDir)
		if helper == "" {
			return "", fmt.Errorf("the Vision OCR helper is not installed; pick another engine in Settings > Search")
		}
		return runTool(ctx, helper, path)
	case Tesseract:
		tool := findTool("tesseract")
		if tool == "" {
			return "", fmt.Errorf("tesseract is not installed (brew install tesseract)")
		}
		return runTool(ctx, tool, path, "stdout", "--psm", "3", "quiet")
	case Ollama:
		if cfg.OllamaModel == "" {
			return "", fmt.Errorf("pick an Ollama vision model for OCR in Settings > Search")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		reply, err := ai.New(cfg.OllamaHost).ChatWith(ctx, cfg.OllamaModel, []ai.ChatMessage{{
			Role: "user",
			Content: "Transcribe all text visible in this image exactly as written, line by line. " +
				"Reply with only the text. If there is no text, reply with nothing.",
			Images: []string{base64.StdEncoding.EncodeToString(data)},
		}}, false, ai.ChatOptions{Think: "off", IdleTimeout: 2 * time.Minute})
		if err != nil {
			return "", err
		}
		return clean([]byte(reply)), nil
	case Off, "":
		return "", ErrUnsupported
	}
	return "", fmt.Errorf("unknown OCR engine %q", cfg.Engine)
}
