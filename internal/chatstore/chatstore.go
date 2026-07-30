// Package chatstore persists story-assistant conversations per project to
// ~/.jumpstart/chats/<projectID>.json. Chats live outside config.json so a
// long conversation never bloats or rewrites the main project config.
package chatstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// maxSessions caps how many conversations are kept per project; the
// oldest are dropped once the cap is exceeded.
const maxSessions = 50

// Message is one turn. Stories carries the assistant's structured output
// so a reopened chat still shows its add-to-board previews.
type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"` // user | assistant
	Content   string          `json:"content"`
	Stories   json.RawMessage `json:"stories,omitempty"` // []GeneratedStory
	Sources   []string        `json:"sources,omitempty"` // files cited from the code index
	CreatedAt int64           `json:"createdAt"`
}

// Session is one conversation thread.
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64     `json:"updatedAt"`
}

// Summary is a session without its messages, for the sidebar list.
type Summary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Messages  int    `json:"messages"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type file struct {
	ProjectID string    `json:"projectId"`
	Sessions  []Session `json:"sessions"`
}

var mu sync.Mutex

// --- paths -------------------------------------------------------------

func chatPath(projectID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".jumpstart", "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, safeName(projectID)+".json"), nil
}

func safeName(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "project"
	}
	return b.String()
}

// --- io ----------------------------------------------------------------

func read(projectID string) (*file, error) {
	path, err := chatPath(projectID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &file{ProjectID: projectID, Sessions: []Session{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return &file{ProjectID: projectID, Sessions: []Session{}}, nil
	}
	if f.Sessions == nil {
		f.Sessions = []Session{}
	}
	return &f, nil
}

func write(f *file) error {
	sort.Slice(f.Sessions, func(a, b int) bool {
		return f.Sessions[a].UpdatedAt > f.Sessions[b].UpdatedAt
	})
	if len(f.Sessions) > maxSessions {
		f.Sessions = f.Sessions[:maxSessions]
	}

	path, err := chatPath(f.ProjectID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// --- API ---------------------------------------------------------------

// List returns every session for a project, newest first, without bodies.
func List(projectID string) ([]Summary, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(f.Sessions))
	for _, s := range f.Sessions {
		out = append(out, Summary{
			ID: s.ID, Title: s.Title, Messages: len(s.Messages),
			CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
		})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].UpdatedAt > out[b].UpdatedAt })
	return out, nil
}

// Get returns one full session.
func Get(projectID, sessionID string) (*Session, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return nil, err
	}
	for i := range f.Sessions {
		if f.Sessions[i].ID == sessionID {
			s := f.Sessions[i]
			return &s, nil
		}
	}
	return nil, fmt.Errorf("chat %s not found", sessionID)
}

// Create starts a new empty session.
func Create(projectID, title string) (*Session, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	s := Session{
		ID:        newID(),
		Title:     strings.TrimSpace(title),
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if s.Title == "" {
		s.Title = "New chat"
	}
	f.Sessions = append(f.Sessions, s)
	if err := write(f); err != nil {
		return nil, err
	}
	return &s, nil
}

// Append adds messages to a session and returns the updated session. The
// session title is derived from the first user message.
func Append(projectID, sessionID string, msgs []Message) (*Session, error) {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range f.Sessions {
		if f.Sessions[i].ID == sessionID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("chat %s not found", sessionID)
	}

	now := time.Now().UnixMilli()
	for _, m := range msgs {
		if m.ID == "" {
			m.ID = newID()
		}
		if m.CreatedAt == 0 {
			m.CreatedAt = now
		}
		f.Sessions[idx].Messages = append(f.Sessions[idx].Messages, m)
	}
	f.Sessions[idx].UpdatedAt = now

	if t := f.Sessions[idx].Title; t == "" || t == "New chat" {
		f.Sessions[idx].Title = deriveTitle(f.Sessions[idx].Messages)
	}
	if err := write(f); err != nil {
		return nil, err
	}
	s := f.Sessions[idx]
	return &s, nil
}

// Rename sets a session's title.
func Rename(projectID, sessionID, title string) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return err
	}
	for i := range f.Sessions {
		if f.Sessions[i].ID == sessionID {
			f.Sessions[i].Title = strings.TrimSpace(title)
			return write(f)
		}
	}
	return fmt.Errorf("chat %s not found", sessionID)
}

// Delete removes one session.
func Delete(projectID, sessionID string) error {
	mu.Lock()
	defer mu.Unlock()

	f, err := read(projectID)
	if err != nil {
		return err
	}
	out := f.Sessions[:0]
	for _, s := range f.Sessions {
		if s.ID != sessionID {
			out = append(out, s)
		}
	}
	f.Sessions = out
	return write(f)
}

// DeleteProject removes every chat for a project.
func DeleteProject(projectID string) error {
	mu.Lock()
	defer mu.Unlock()

	path, err := chatPath(projectID)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// deriveTitle names a session after its first user message.
func deriveTitle(msgs []Message) string {
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		t := strings.TrimSpace(strings.ReplaceAll(m.Content, "\n", " "))
		if len(t) > 48 {
			t = strings.TrimSpace(t[:48]) + "…"
		}
		if t != "" {
			return t
		}
	}
	return "New chat"
}

var idSeq struct {
	sync.Mutex
	n int64
}

func newID() string {
	idSeq.Lock()
	idSeq.n++
	n := idSeq.n
	idSeq.Unlock()
	return fmt.Sprintf("%d-%d", time.Now().UnixMilli(), n)
}
