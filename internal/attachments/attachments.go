// Package attachments stores files attached to tasks and validates task
// links. Files are copied under <data dir>/attachments/<projectId>/<taskId>/
// and always addressed by IDs the app generated, so a caller can never reach
// outside that tree by supplying a path.
package attachments

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"devdeck/internal/model"
)

const (
	// MaxFileBytes caps one attachment. Files are copied, so this also
	// bounds the disk a single upload can take.
	MaxFileBytes = 100 << 20
	// MaxPreviewBytes caps what is inlined as a data: URL for in-app preview.
	MaxPreviewBytes = 25 << 20
)

var (
	idPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
	filePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}(\.[A-Za-z0-9]{1,10})?$`)
)

var imageMimes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true,
	"image/webp": true, "image/bmp": true, "image/svg+xml": true,
}

// IsImage reports whether the MIME type can be previewed inline.
func IsImage(mimeType string) bool { return imageMimes[strings.ToLower(mimeType)] }

var textExts = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".csv": true, ".tsv": true, ".json": true,
	".yaml": true, ".yml": true, ".xml": true, ".html": true, ".htm": true, ".log": true,
	".toml": true, ".ini": true, ".sql": true, ".js": true, ".ts": true, ".jsx": true,
	".tsx": true, ".go": true, ".py": true, ".java": true, ".rb": true, ".sh": true,
	".css": true, ".scss": true, ".cls": true, ".trigger": true, ".apex": true, ".soql": true,
}

// IsText reports whether an attachment can be read as plain text (by MIME
// type or a known source/text extension).
func IsText(a model.Attachment) bool {
	if strings.HasPrefix(strings.ToLower(a.Mime), "text/") {
		return true
	}
	return textExts[strings.ToLower(filepath.Ext(a.Name))]
}

func newID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("att%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// taskDir returns the directory holding one task's files. Both IDs must be
// plain identifiers so they cannot contain path separators or "..".
func taskDir(dataDir, projectID, taskID string) (string, error) {
	if !idPattern.MatchString(projectID) || !idPattern.MatchString(taskID) {
		return "", fmt.Errorf("invalid project or task id")
	}
	return filepath.Join(dataDir, "attachments", projectID, taskID), nil
}

// cleanName turns an untrusted file name into a safe display name.
func cleanName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || name == "/" {
		return "file"
	}
	if len(name) > 200 {
		ext := filepath.Ext(name)
		if len(ext) > 12 {
			ext = ""
		}
		name = name[:200-len(ext)] + ext
	}
	return name
}

// safeExt returns the file's extension if it is short and alphanumeric.
func safeExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if len(ext) < 2 || len(ext) > 11 {
		return ""
	}
	for _, r := range ext[1:] {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return ""
		}
	}
	return ext
}

// Save copies r into the task's attachment directory, enforcing MaxFileBytes.
func Save(dataDir, projectID, taskID, name string, r io.Reader) (model.Attachment, error) {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return model.Attachment{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return model.Attachment{}, err
	}
	name = cleanName(name)
	id := newID()
	file := id + safeExt(name)
	dest := filepath.Join(dir, file)
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return model.Attachment{}, err
	}
	// Read one byte past the limit so an oversize file is detected, not truncated.
	written, err := io.Copy(f, io.LimitReader(r, MaxFileBytes+1))
	closeErr := f.Close()
	if err == nil && written > MaxFileBytes {
		err = fmt.Errorf("%s is larger than the %d MB attachment limit", name, MaxFileBytes>>20)
	}
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(dest)
		return model.Attachment{}, err
	}
	mt := mime.TypeByExtension(safeExt(name))
	if mt == "" {
		mt = sniff(dest)
	}
	if i := strings.Index(mt, ";"); i >= 0 {
		mt = strings.TrimSpace(mt[:i])
	}
	return model.Attachment{
		ID: id, Name: name, File: file, Mime: mt,
		Size: written, AddedAt: time.Now().UnixMilli(),
	}, nil
}

// sniff guesses a MIME type from the first bytes of a file.
func sniff(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	return http.DetectContentType(head[:n])
}

// SaveBase64 stores base64-encoded bytes, e.g. a pasted screenshot.
func SaveBase64(dataDir, projectID, taskID, name, b64 string) (model.Attachment, error) {
	if i := strings.Index(b64, ","); i >= 0 && strings.HasPrefix(b64, "data:") {
		b64 = b64[i+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return model.Attachment{}, fmt.Errorf("attachment is not valid base64")
	}
	return Save(dataDir, projectID, taskID, name, strings.NewReader(string(raw)))
}

// Path resolves an attachment to its absolute file path. It re-validates the
// stored file name and confirms the result stays inside the task directory.
func Path(dataDir, projectID, taskID string, att model.Attachment) (string, error) {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return "", err
	}
	if !filePattern.MatchString(att.File) {
		return "", fmt.Errorf("invalid attachment file name")
	}
	p := filepath.Join(dir, att.File)
	if filepath.Dir(p) != dir {
		return "", fmt.Errorf("invalid attachment path")
	}
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("attachment file is missing: %s", att.Name)
	}
	return p, nil
}

// DataURL reads an attachment as a data: URL for inline preview. Only
// images up to MaxPreviewBytes are allowed.
func DataURL(dataDir, projectID, taskID string, att model.Attachment) (string, error) {
	if !IsImage(att.Mime) {
		return "", fmt.Errorf("%s is not a previewable image", att.Name)
	}
	p, err := Path(dataDir, projectID, taskID, att)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if info.Size() > MaxPreviewBytes {
		return "", fmt.Errorf("%s is too large to preview in the app; use Open", att.Name)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return "data:" + att.Mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

// Remove deletes one stored file. A missing file is not an error.
func Remove(dataDir, projectID, taskID string, att model.Attachment) error {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return err
	}
	if !filePattern.MatchString(att.File) {
		return fmt.Errorf("invalid attachment file name")
	}
	err = os.Remove(filepath.Join(dir, att.File))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// RemoveTask deletes every file stored for a task.
func RemoveTask(dataDir, projectID, taskID string) error {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// RemoveProject deletes every file stored for a project.
func RemoveProject(dataDir, projectID string) error {
	if !idPattern.MatchString(projectID) {
		return fmt.Errorf("invalid project id")
	}
	return os.RemoveAll(filepath.Join(dataDir, "attachments", projectID))
}

// Removed lists attachments present on tasks in before but absent from after,
// keyed by task ID, plus the IDs of tasks deleted outright.
func Removed(before, after []model.Task) (files map[string][]model.Attachment, deletedTasks []string) {
	kept := make(map[string]map[string]bool, len(after))
	for _, t := range after {
		ids := make(map[string]bool, len(t.Attachments))
		for _, a := range t.Attachments {
			ids[a.ID] = true
		}
		kept[t.ID] = ids
	}
	files = map[string][]model.Attachment{}
	for _, t := range before {
		ids, stillThere := kept[t.ID]
		if !stillThere {
			if len(t.Attachments) > 0 {
				deletedTasks = append(deletedTasks, t.ID)
			}
			continue
		}
		for _, a := range t.Attachments {
			if !ids[a.ID] {
				files[t.ID] = append(files[t.ID], a)
			}
		}
	}
	return files, deletedTasks
}

// NormalizeURL validates a link URL. Only http, https, and mailto are
// allowed, so javascript:, file:, and data: links can never be opened. A bare
// "example.com/page" is treated as https.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("link URL is empty")
	}
	if !hasScheme(raw) {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid link URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return "", fmt.Errorf("link URL has no host")
		}
	case "mailto":
		if u.Opaque == "" && u.Path == "" {
			return "", fmt.Errorf("mailto link has no address")
		}
	default:
		return "", fmt.Errorf("only http, https, and mailto links are allowed")
	}
	return u.String(), nil
}

// hasScheme reports whether s starts with "<letters>:" other than a bare
// host:port such as "localhost:3000".
func hasScheme(s string) bool {
	i := strings.Index(s, ":")
	if i <= 0 {
		return false
	}
	for _, r := range s[:i] {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '-' || r == '.') {
			return false
		}
	}
	rest := s[i+1:]
	if strings.HasPrefix(rest, "//") {
		return true
	}
	// "localhost:3000/x" has digits after the colon; "mailto:a@b.c" does not.
	if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
		end := strings.IndexAny(rest, "/?#")
		port := rest
		if end >= 0 {
			port = rest[:end]
		}
		allDigits := port != ""
		for _, r := range port {
			if r < '0' || r > '9' {
				allDigits = false
			}
		}
		if allDigits {
			return false
		}
	}
	return true
}

const trashDirName = ".trash"

// TrashRetention is how long removed attachment files stay recoverable.
const TrashRetention = 7 * 24 * time.Hour

// Trash moves a removed attachment into <data dir>/attachments/.trash instead
// of deleting it, so a stale window or an agent overwriting a task cannot
// destroy a file for good. PurgeTrash removes old entries.
func Trash(dataDir, projectID, taskID string, att model.Attachment) error {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return err
	}
	if !filePattern.MatchString(att.File) {
		return fmt.Errorf("invalid attachment file name")
	}
	src := filepath.Join(dir, att.File)
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return moveToTrash(dataDir, projectID, taskID, src, att.File)
}

// TrashTask moves everything stored for a task into the trash.
func TrashTask(dataDir, projectID, taskID string) error {
	dir, err := taskDir(dataDir, projectID, taskID)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := moveToTrash(dataDir, projectID, taskID, filepath.Join(dir, e.Name()), e.Name()); err != nil {
			return err
		}
	}
	return os.Remove(dir)
}

func moveToTrash(dataDir, projectID, taskID, src, file string) error {
	stamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	dest := filepath.Join(dataDir, "attachments", trashDirName, stamp, projectID, taskID)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return os.Rename(src, filepath.Join(dest, file))
}

// PurgeTrash deletes trash batches older than maxAge. Batch folders are named
// by the unix-ms time they were trashed.
func PurgeTrash(dataDir string, maxAge time.Duration, now time.Time) error {
	root := filepath.Join(dataDir, "attachments", trashDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := now.Add(-maxAge).UnixMilli()
	for _, e := range entries {
		var ms int64
		if _, err := fmt.Sscanf(e.Name(), "%d", &ms); err != nil || ms >= cutoff {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
