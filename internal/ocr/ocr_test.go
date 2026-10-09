package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"devdeck/internal/attachments"
	"devdeck/internal/model"
)

func fixture(t *testing.T) []byte {
	data, err := os.ReadFile("testdata/text.png")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func save(t *testing.T, dir, name string, data []byte) model.Attachment {
	a, err := attachments.Save(dir, "p1", "t1", name, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestKind(t *testing.T) {
	cases := map[string]model.Attachment{
		"text":  {Name: "notes.md"},
		"pdf":   {Name: "spec.pdf", Mime: "application/pdf"},
		"image": {Name: "shot.png", Mime: "image/png"},
		"":      {Name: "deck.pptx", Mime: "application/vnd.ms-powerpoint"},
	}
	for want, a := range cases {
		if got := Kind(a); got != want {
			t.Errorf("Kind(%s) = %q, want %q", a.Name, got, want)
		}
	}
}

func TestTesseract(t *testing.T) {
	if findTool("tesseract") == "" {
		t.Skip("tesseract not installed")
	}
	path := filepath.Join(t.TempDir(), "x.png")
	_ = os.WriteFile(path, fixture(t), 0o644)
	text, err := Image(context.Background(), t.TempDir(), Config{Engine: Tesseract}, path)
	if err != nil || !strings.Contains(text, "4471") || !strings.Contains(strings.ToLower(text), "ledger") {
		t.Fatalf("tesseract: %q %v", text, err)
	}
}

// The Vision helper is built by scripts/build-ocr-helper.sh; point
// JUMPSTART_OCR_HELPER at it to run this test.
func TestVisionHelper(t *testing.T) {
	helper := os.Getenv("JUMPSTART_OCR_HELPER")
	if helper == "" {
		t.Skip("set JUMPSTART_OCR_HELPER to the built jumpstart-ocr to test Vision")
	}
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	data, _ := os.ReadFile(helper)
	_ = os.WriteFile(filepath.Join(dir, "bin", "jumpstart-ocr"), data, 0o755)
	_ = exec.Command("codesign", "--force", "--sign", "-", filepath.Join(dir, "bin", "jumpstart-ocr")).Run()
	path := filepath.Join(dir, "x.png")
	_ = os.WriteFile(path, fixture(t), 0o644)
	text, err := Image(context.Background(), dir, Config{Engine: Vision}, path)
	if err != nil || !strings.Contains(text, "Invoice 4471 whiteboard") {
		t.Fatalf("vision: %q %v", text, err)
	}
}

func TestVisionMissingHelperIsAClearError(t *testing.T) {
	if HelperPath(t.TempDir()) != "" {
		t.Skip("a helper sits next to the test binary")
	}
	_, err := Image(context.Background(), t.TempDir(), Config{Engine: Vision}, "x.png")
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("err = %v", err)
	}
}

func mockOllama(t *testing.T, reply string, calls *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Images []string `json:"images"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Model != "mock-vision" || len(req.Messages) == 0 || len(req.Messages[0].Images) != 1 {
			http.Error(w, "bad request", 400)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": reply}, "done": true})
	}))
}

func TestOllamaEngine(t *testing.T) {
	var calls int32
	srv := mockOllama(t, "Invoice 4471", &calls)
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "x.png")
	_ = os.WriteFile(path, fixture(t), 0o644)
	text, err := Image(context.Background(), t.TempDir(), Config{Engine: Ollama, OllamaHost: srv.URL, OllamaModel: "mock-vision"}, path)
	if err != nil || text != "Invoice 4471" {
		t.Fatalf("ollama: %q %v", text, err)
	}
	if _, err := Image(context.Background(), t.TempDir(), Config{Engine: Ollama}, path); err == nil {
		t.Error("no model should be an error")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerQueuesCachesAndResets(t *testing.T) {
	dir := t.TempDir()
	var calls int32
	srv := mockOllama(t, "Invoice 4471", &calls)
	defer srv.Close()
	cfg := Config{Engine: Ollama, OllamaHost: srv.URL, OllamaModel: "mock-vision"}
	var changes int32
	w := NewWorker(dir, func() Config { return cfg }, func() { atomic.AddInt32(&changes, 1) })

	txt := save(t, dir, "notes.txt", []byte("Deploy on Friday"))
	if text, state := w.Text("p1", "t1", txt); state != stateReady || text != "Deploy on Friday" {
		t.Fatalf("text file: %q %s", text, state)
	}

	img := save(t, dir, "board.png", fixture(t))
	if _, state := w.Text("p1", "t1", img); state != statePending {
		t.Fatalf("first read should queue, got %s", state)
	}
	waitFor(t, func() bool { _, s := w.Text("p1", "t1", img); return s == stateReady })
	if text, _ := w.Text("p1", "t1", img); text != "Invoice 4471" {
		t.Fatalf("cached text %q", text)
	}
	waitFor(t, func() bool { return atomic.LoadInt32(&changes) > 0 })
	w.Text("p1", "t1", img)
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("cached result should not call the engine again, calls=%d", calls)
	}
	if info := w.Status("p1", "t1", img); info.State != stateReady || info.Engine != Ollama || info.Chars != 12 {
		t.Errorf("status %+v", info)
	}

	// A failure is cached (no retry storm) until the settings are fixed.
	bad := save(t, dir, "bad.png", fixture(t))
	cfg.OllamaModel = ""
	w.Text("p1", "t1", bad)
	waitFor(t, func() bool { return w.Status("p1", "t1", bad).Error != "" })
	if _, s := w.Text("p1", "t1", bad); s != stateNone {
		t.Errorf("failed entry state %s", s)
	}
	cfg.OllamaModel = "mock-vision"
	if n := w.Reset(true); n != 1 {
		t.Errorf("Reset(failedOnly) removed %d", n)
	}
	w.Text("p1", "t1", bad)
	waitFor(t, func() bool { _, s := w.Text("p1", "t1", bad); return s == stateReady })

	// OCR off: images are not searchable and nothing is cached.
	cfg.Engine = Off
	off := save(t, dir, "off.png", fixture(t))
	if _, s := w.Text("p1", "t1", off); s != stateNone {
		t.Errorf("off: %s", s)
	}

	// Forget re-runs one file; Prune drops text for removed attachments.
	w.Forget("p1", "t1", img)
	if w.Status("p1", "t1", img).State != stateNone {
		t.Error("forget should drop the cache")
	}
	w.Prune([]model.Project{{ID: "p1", Tasks: []model.Task{{ID: "t1", Attachments: []model.Attachment{img}}}}})
	if _, err := os.Stat(w.cachePath("p1", "t1", bad)); !os.IsNotExist(err) {
		t.Error("prune should remove text of attachments that are gone")
	}
}

func TestCachePathCannotEscape(t *testing.T) {
	w := &Worker{dataDir: "/data"}
	p := w.cachePath("../../etc", "..", model.Attachment{File: "../../passwd"})
	if !strings.HasPrefix(p, "/data/search-text/") || strings.Contains(p, "..") {
		t.Fatalf("cache path escaped: %s", p)
	}
}
