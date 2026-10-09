package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeOllama streams the given chunks as NDJSON and records request bodies.
func fakeOllama(t *testing.T, chunks []string, pause time.Duration) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		json.NewDecoder(r.Body).Decode(&b)
		mu.Lock()
		bodies = append(bodies, b)
		mu.Unlock()
		if r.URL.Path == "/api/show" {
			model, _ := b["model"].(string)
			caps := []string{"completion"}
			if strings.Contains(model, "think") || strings.Contains(model, "gpt-oss") {
				caps = append(caps, "thinking")
			}
			if strings.Contains(model, "vision") {
				caps = append(caps, "vision")
			}
			json.NewEncoder(w).Encode(map[string]any{"capabilities": caps})
			return
		}
		fl, _ := w.(http.Flusher)
		for _, c := range chunks {
			fmt.Fprintln(w, c)
			if fl != nil {
				fl.Flush()
			}
			select {
			case <-r.Context().Done():
				return
			case <-time.After(pause):
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

func TestChatWithStreamsThinkingAndContent(t *testing.T) {
	srv, bodies := fakeOllama(t, []string{
		`{"message":{"thinking":"Let me "}}`,
		`{"message":{"thinking":"think."}}`,
		`{"message":{"content":"{\"reply\":"}}`,
		`{"message":{"content":"\"hi\"}"},"done":true}`,
	}, 0)
	var thinking, content strings.Builder
	out, err := New(srv.URL).ChatWith(context.Background(), "qwen3-think", []ChatMessage{{Role: "user", Content: "x"}}, true,
		ChatOptions{Think: ThinkHigh, OnDelta: func(d Delta) { thinking.WriteString(d.Thinking); content.WriteString(d.Content) }})
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"reply":"hi"}` || content.String() != out || thinking.String() != "Let me think." {
		t.Errorf("out=%q content=%q thinking=%q", out, content.String(), thinking.String())
	}
	b := (*bodies)[0]
	if b["stream"] != true || b["format"] != "json" || b["think"] != true {
		t.Errorf("request body = %v (a level on an on/off model must be sent as true)", b)
	}
}

func TestThinkValueMapping(t *testing.T) {
	cases := []struct {
		level  string
		levels bool
		want   any
	}{
		{ThinkAuto, false, nil}, {"", true, nil}, {ThinkOff, true, false}, {ThinkOff, false, false},
		{ThinkLow, false, true}, {ThinkHigh, false, true}, {ThinkLow, true, "low"}, {ThinkMedium, true, "medium"}, {ThinkHigh, true, "high"},
	}
	for _, c := range cases {
		if got := thinkValue(c.level, c.levels); got != c.want {
			t.Errorf("thinkValue(%q, %v) = %v, want %v", c.level, c.levels, got, c.want)
		}
	}
	srv, bodies := fakeOllama(t, []string{`{"message":{"content":"ok"},"done":true}`}, 0)
	New(srv.URL).ChatWith(context.Background(), "m", nil, false, ChatOptions{Think: ThinkAuto})
	if _, sent := (*bodies)[0]["think"]; sent {
		t.Error("auto must omit the think field")
	}
}

func TestChatWithCancel(t *testing.T) {
	srv, _ := fakeOllama(t, []string{`{"message":{"thinking":"a"}}`, `{"message":{"thinking":"b"}}`, `{"message":{"thinking":"c"}}`}, 300*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)
	start := time.Now()
	_, err := New(srv.URL).ChatWith(ctx, "m", nil, false, ChatOptions{})
	if err != ErrCanceled {
		t.Fatalf("err = %v, want ErrCanceled", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Error("cancel did not stop the request promptly")
	}
}

func TestChatWithIdleTimeout(t *testing.T) {
	srv, _ := fakeOllama(t, []string{`{"message":{"thinking":"a"}}`, `{"message":{"content":"late"},"done":true}`}, 2*time.Second)
	_, err := New(srv.URL).ChatWith(context.Background(), "m", nil, false, ChatOptions{IdleTimeout: 300 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "sent nothing") {
		t.Fatalf("err = %v, want idle timeout", err)
	}
}

func TestChatWithStreamError(t *testing.T) {
	srv, _ := fakeOllama(t, []string{`{"error":"model requires more system memory"}`}, 0)
	_, err := New(srv.URL).ChatWith(context.Background(), "m", nil, false, ChatOptions{})
	if err == nil || !strings.Contains(err.Error(), "more system memory") {
		t.Fatalf("err = %v", err)
	}
}

func TestChatStillWorksForOtherCallers(t *testing.T) {
	srv, _ := fakeOllama(t, []string{`{"message":{"content":"Hello"}}`, `{"message":{"content":" world"},"done":true}`}, 0)
	out, err := New(srv.URL).Chat(context.Background(), "m", nil, false)
	if err != nil || out != "Hello world" {
		t.Fatalf("Chat = %q, %v", out, err)
	}
}

func TestShowModel(t *testing.T) {
	srv, _ := fakeOllama(t, nil, 0)
	c := New(srv.URL)
	for model, want := range map[string]ModelInfo{
		"llama3.2":        {},
		"qwen3-think":     {Thinking: true},
		"gpt-oss:20b":     {Thinking: true, ThinkLevels: true},
		"llava-vision:7b": {Vision: true},
	} {
		got, err := c.ShowModel(context.Background(), model)
		if err != nil || got != want {
			t.Errorf("ShowModel(%q) = %+v, %v; want %+v", model, got, err, want)
		}
	}
}
