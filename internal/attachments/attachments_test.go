package attachments

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devdeck/internal/model"
)

func TestSaveAndPath(t *testing.T) {
	dir := t.TempDir()
	att, err := Save(dir, "proj1", "task1", "../../etc/Report Final.PDF", strings.NewReader("%PDF-1.4 data"))
	if err != nil {
		t.Fatal(err)
	}
	if att.Name != "Report Final.PDF" {
		t.Errorf("display name = %q, path components must be stripped", att.Name)
	}
	if !strings.HasSuffix(att.File, ".pdf") || strings.ContainsAny(att.File, "/\\ ") {
		t.Errorf("stored file name = %q", att.File)
	}
	if att.Mime != "application/pdf" || att.Size != int64(len("%PDF-1.4 data")) {
		t.Errorf("mime/size = %q/%d", att.Mime, att.Size)
	}
	p, err := Path(dir, "proj1", "task1", att)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "attachments", "proj1", "task1", att.File); p != want {
		t.Errorf("path = %q, want %q", p, want)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "%PDF-1.4 data" {
		t.Errorf("stored bytes = %q", got)
	}
}

func TestPathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"../secret", "a/b.png", "..", "", "x y.png", "a.b.c.d"} {
		if _, err := Path(dir, "p", "t", model.Attachment{File: file}); err == nil {
			t.Errorf("file %q should be rejected", file)
		}
	}
	for _, id := range [][2]string{{"../x", "t"}, {"p", "a/b"}, {"", "t"}, {"p", ".."}} {
		if _, err := Save(dir, id[0], id[1], "f.txt", strings.NewReader("x")); err == nil {
			t.Errorf("ids %v should be rejected", id)
		}
	}
}

func TestSaveRejectsOversize(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates >100MB")
	}
	dir := t.TempDir()
	big := bytes.NewReader(make([]byte, MaxFileBytes+1))
	if _, err := Save(dir, "p", "t", "big.bin", big); err == nil {
		t.Fatal("oversize upload should fail")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "attachments", "p", "t"))
	if len(entries) != 0 {
		t.Errorf("failed upload left %d files behind", len(entries))
	}
}

func TestMimeFromContentWhenNoExtension(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("0", 32))
	att, err := Save(t.TempDir(), "p", "t", "screenshot", bytes.NewReader(png))
	if err != nil {
		t.Fatal(err)
	}
	if att.Mime != "image/png" || !IsImage(att.Mime) {
		t.Errorf("mime = %q", att.Mime)
	}
}

func TestSaveBase64AndDataURL(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("1", 16))
	att, err := SaveBase64(dir, "p", "t", "pasted.png", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	url, err := DataURL(dir, "p", "t", att)
	if err != nil || !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("data url = %q (%v)", url, err)
	}
	doc, _ := Save(dir, "p", "t", "notes.txt", strings.NewReader("hello"))
	if _, err := DataURL(dir, "p", "t", doc); err == nil {
		t.Error("non-image must not be previewable inline")
	}
	if _, err := SaveBase64(dir, "p", "t", "x.png", "!!!not base64"); err == nil {
		t.Error("invalid base64 should fail")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	att, _ := Save(dir, "p", "t", "a.txt", strings.NewReader("a"))
	if err := Remove(dir, "p", "t", att); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "p", "t", att); err != nil {
		t.Errorf("removing twice should not fail: %v", err)
	}
	if _, err := Path(dir, "p", "t", att); err == nil {
		t.Error("removed file should be missing")
	}
	other, _ := Save(dir, "p", "t2", "b.txt", strings.NewReader("b"))
	if err := RemoveTask(dir, "p", "t2"); err != nil {
		t.Fatal(err)
	}
	if _, err := Path(dir, "p", "t2", other); err == nil {
		t.Error("task removal should delete its files")
	}
	if err := RemoveProject(dir, "p"); err != nil {
		t.Fatal(err)
	}
}

func TestRemoved(t *testing.T) {
	a1, a2 := model.Attachment{ID: "a1"}, model.Attachment{ID: "a2"}
	before := []model.Task{
		{ID: "keep", Attachments: []model.Attachment{a1, a2}},
		{ID: "gone", Attachments: []model.Attachment{a1}},
		{ID: "bare"},
	}
	after := []model.Task{{ID: "keep", Attachments: []model.Attachment{a1}}}
	files, deleted := Removed(before, after)
	if len(files["keep"]) != 1 || files["keep"][0].ID != "a2" {
		t.Errorf("files = %+v", files)
	}
	if len(deleted) != 1 || deleted[0] != "gone" {
		t.Errorf("deleted = %v (tasks without attachments need no cleanup)", deleted)
	}
}

func TestNormalizeURL(t *testing.T) {
	ok := map[string]string{
		"https://example.com/a?b=1":  "https://example.com/a?b=1",
		"example.com/docs":           "https://example.com/docs",
		"  http://localhost:3000/x ": "http://localhost:3000/x",
		"localhost:3000/x":           "https://localhost:3000/x",
		"mailto:me@example.com":      "mailto:me@example.com",
	}
	for in, want := range ok {
		got, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{
		"", "   ", "javascript:alert(1)", "JaVaScRiPt:alert(1)", "file:///etc/passwd",
		"data:text/html,<script>", "ftp://example.com", "http://", "vbscript:x",
	} {
		if got, err := NormalizeURL(bad); err == nil {
			t.Errorf("NormalizeURL(%q) = %q, want error", bad, got)
		}
	}
}

func TestTrashKeepsFilesRecoverableAndPurgeExpiresThem(t *testing.T) {
	dir := t.TempDir()
	att, _ := Save(dir, "p", "t", "keep.txt", strings.NewReader("precious"))
	if err := Trash(dir, "p", "t", att); err != nil {
		t.Fatal(err)
	}
	if _, err := Path(dir, "p", "t", att); err == nil {
		t.Error("trashed file should leave the task folder")
	}
	var found string
	filepath.Walk(filepath.Join(dir, "attachments", ".trash"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Base(p) == att.File {
			found = p
		}
		return nil
	})
	if found == "" {
		t.Fatal("file not found in trash")
	}
	if got, _ := os.ReadFile(found); string(got) != "precious" {
		t.Errorf("trashed content = %q", got)
	}
	// Trashing twice (already gone) is fine.
	if err := Trash(dir, "p", "t", att); err != nil {
		t.Errorf("second trash: %v", err)
	}

	// A fresh batch survives a purge; a week-old one does not.
	if err := PurgeTrash(dir, TrashRetention, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(found); err != nil {
		t.Error("recent trash must survive purge")
	}
	if err := PurgeTrash(dir, TrashRetention, time.Now().Add(8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(found); err == nil {
		t.Error("expired trash should be purged")
	}
}

func TestTrashTaskMovesEverything(t *testing.T) {
	dir := t.TempDir()
	a, _ := Save(dir, "p", "t", "a.txt", strings.NewReader("a"))
	b, _ := Save(dir, "p", "t", "b.txt", strings.NewReader("b"))
	if err := TrashTask(dir, "p", "t"); err != nil {
		t.Fatal(err)
	}
	for _, att := range []model.Attachment{a, b} {
		if _, err := Path(dir, "p", "t", att); err == nil {
			t.Errorf("%s still in task folder", att.Name)
		}
	}
	if err := TrashTask(dir, "p", "never-existed"); err != nil {
		t.Errorf("trashing a task with no files should be a no-op: %v", err)
	}
}
