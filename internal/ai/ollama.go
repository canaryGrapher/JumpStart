// Package ai is a small client for a local Ollama server. It powers the
// "fill with AI" action in the task modal and the story-generating chat.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultHost is Ollama's local address.
const DefaultHost = "http://localhost:11434"

// Client talks to one Ollama server.
type Client struct {
	host string
	http *http.Client
}

// New returns a client for host. Empty host falls back to the OLLAMA_HOST
// env var, then DefaultHost.
func New(host string) *Client {
	host = strings.TrimSpace(host)
	if host == "" {
		host = os.Getenv("OLLAMA_HOST")
	}
	if host == "" {
		host = DefaultHost
	}
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return &Client{
		host: strings.TrimRight(host, "/"),
		http: &http.Client{Timeout: 5 * time.Minute},
	}
}

// ChatMessage is one turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"` // system | user | assistant
	Content string `json:"content"`
	// Images are base64 image bytes for vision-capable models.
	Images []string `json:"images,omitempty"`
}

// ListModels returns the names of every model installed on the server.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Ollama at %s: %w", c.host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned %s", resp.Status)
	}
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		// Embedding-only models reject /api/chat; keep them out of the
		// picker so "Populate with AI" cannot silently pick one.
		if isEmbeddingModel(m.Name) {
			continue
		}
		names = append(names, m.Name)
	}
	return names, nil
}

func isEmbeddingModel(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "embed") ||
		strings.Contains(n, "nomic-embed") ||
		strings.HasPrefix(n, "bge-") ||
		strings.Contains(n, "e5-")
}

// Chat runs a chat completion and returns the reply text. When jsonMode is
// true the model is asked to emit strict JSON. It uses the model's default
// thinking behavior; see ChatWith for effort, streaming and cancellation.
func (c *Client) Chat(ctx context.Context, model string, msgs []ChatMessage, jsonMode bool) (string, error) {
	return c.ChatWith(ctx, model, msgs, jsonMode, ChatOptions{})
}

// Think levels understood by ChatOptions.Think.
const (
	ThinkAuto   = "auto"
	ThinkOff    = "off"
	ThinkLow    = "low"
	ThinkMedium = "medium"
	ThinkHigh   = "high"
)

// Delta is one streamed piece of a reply. Thinking is the model's reasoning
// trace (only for thinking-capable models); Content is answer text.
type Delta struct {
	Thinking string
	Content  string
}

// ChatOptions tunes one chat call.
type ChatOptions struct {
	// Think is auto|off|low|medium|high. Auto sends nothing (model default).
	Think string
	// ThinkLevels reports that the model accepts low/medium/high (e.g.
	// gpt-oss). Other thinking models only accept on/off, so any level
	// other than off is sent as true.
	ThinkLevels bool
	// OnDelta, when set, receives the reply as it streams.
	OnDelta func(Delta)
	// IdleTimeout aborts when the model sends nothing for this long.
	// Zero means DefaultIdleTimeout.
	IdleTimeout time.Duration
}

// DefaultIdleTimeout is how long a streaming reply may go silent before the
// call fails with a clear error. Thinking models stream their reasoning, so
// a long silence means the server or model has stalled.
const DefaultIdleTimeout = 3 * time.Minute

// thinkValue maps an effort level to Ollama's `think` field, or nil to omit it.
func thinkValue(level string, levels bool) any {
	switch level {
	case ThinkOff:
		return false
	case ThinkLow, ThinkMedium, ThinkHigh:
		if levels {
			return level
		}
		return true
	}
	return nil
}

// ChatWith streams a chat completion, honoring ctx cancellation and an idle
// timeout, and returns the full reply text.
func (c *Client) ChatWith(ctx context.Context, model string, msgs []ChatMessage, jsonMode bool, opts ChatOptions) (string, error) {
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("no model selected; pick one in Preferences → AI")
	}
	body := map[string]any{
		"model":    model,
		"messages": msgs,
		"stream":   true,
	}
	if jsonMode {
		body["format"] = "json"
	}
	if v := thinkValue(opts.Think, opts.ThinkLevels); v != nil {
		body["think"] = v
	}
	buf, _ := json.Marshal(body)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	idle := opts.IdleTimeout
	if idle <= 0 {
		idle = DefaultIdleTimeout
	}
	errIdle := fmt.Errorf("the model sent nothing for %s; it may be overloaded or stuck — try a smaller model or lower thinking effort", idle)
	timer := time.AfterFunc(idle, func() { cancel(errIdle) })
	defer timer.Stop()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/chat", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Streaming replies can legitimately run for many minutes, so the
	// whole-request timeout is replaced by the idle timer above.
	stream := &http.Client{Transport: c.http.Transport}
	resp, err := stream.Do(req)
	if err != nil {
		if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
			return "", cause
		}
		if ctx.Err() != nil {
			return "", ErrCanceled
		}
		return "", fmt.Errorf("cannot reach Ollama at %s: %w", c.host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error != "" {
			return "", fmt.Errorf("ollama: %s", e.Error)
		}
		return "", fmt.Errorf("ollama returned %s", resp.Status)
	}

	var out strings.Builder
	dec := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Message struct {
				Content  string `json:"content"`
				Thinking string `json:"thinking"`
			} `json:"message"`
			Done  bool   `json:"done"`
			Error string `json:"error"`
		}
		if err := dec.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			if cause := context.Cause(ctx); cause != nil {
				if cause == context.Canceled {
					return out.String(), ErrCanceled
				}
				return out.String(), cause
			}
			return out.String(), fmt.Errorf("reading Ollama stream: %w", err)
		}
		timer.Reset(idle)
		if chunk.Error != "" {
			return out.String(), fmt.Errorf("%s", chunk.Error)
		}
		out.WriteString(chunk.Message.Content)
		if opts.OnDelta != nil && (chunk.Message.Content != "" || chunk.Message.Thinking != "") {
			opts.OnDelta(Delta{Thinking: chunk.Message.Thinking, Content: chunk.Message.Content})
		}
		if chunk.Done {
			break
		}
	}
	return out.String(), nil
}

// ErrCanceled is returned when the caller stops a request.
var ErrCanceled = fmt.Errorf("stopped")

// ModelInfo describes what a model can do.
type ModelInfo struct {
	Thinking    bool `json:"thinking"`
	ThinkLevels bool `json:"thinkLevels"`
	Vision      bool `json:"vision"`
}

// ShowModel asks Ollama for a model's capabilities. Older servers that do
// not report capabilities yield a zero ModelInfo, not an error.
func (c *Client) ShowModel(ctx context.Context, model string) (ModelInfo, error) {
	var info ModelInfo
	buf, _ := json.Marshal(map[string]string{"model": model})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/show", bytes.NewReader(buf))
	if err != nil {
		return info, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return info, fmt.Errorf("cannot reach Ollama at %s: %w", c.host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("ollama returned %s", resp.Status)
	}
	var out struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return info, err
	}
	for _, c := range out.Capabilities {
		switch c {
		case "thinking":
			info.Thinking = true
		case "vision":
			info.Vision = true
		}
	}
	// Only gpt-oss style models take low/medium/high; others take true/false.
	if info.Thinking && strings.Contains(strings.ToLower(model), "gpt-oss") {
		info.ThinkLevels = true
	}
	return info, nil
}
