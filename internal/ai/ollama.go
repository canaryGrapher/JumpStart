// Package ai is a small client for a local Ollama server. It powers the
// "fill with AI" action in the task modal and the story-generating chat.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
		names = append(names, m.Name)
	}
	return names, nil
}

// Chat runs a non-streaming chat completion and returns the reply text.
// When jsonMode is true the model is asked to emit strict JSON.
func (c *Client) Chat(ctx context.Context, model string, msgs []ChatMessage, jsonMode bool) (string, error) {
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("no model selected; pick one in Preferences → AI")
	}
	body := map[string]any{
		"model":    model,
		"messages": msgs,
		"stream":   false,
	}
	if jsonMode {
		body["format"] = "json"
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/chat", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach Ollama at %s: %w", c.host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %s", resp.Status)
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != "" {
		return "", fmt.Errorf("%s", out.Error)
	}
	return out.Message.Content, nil
}
