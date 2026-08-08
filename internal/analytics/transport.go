package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Event is one captured event in PostHog's wire format. It is exported only
// so the offline queue can serialise it; call sites use Client.Track.
type Event struct {
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Properties map[string]any `json:"properties"`
	Timestamp  string         `json:"timestamp"`
}

type batchPayload struct {
	APIKey string  `json:"api_key"`
	Batch  []Event `json:"batch"`
}

// sender posts batches to PostHog's capture endpoint. It is deliberately a
// hand-rolled 60 lines rather than posthog-go: the only feature this app
// needs is "POST some JSON", and a direct client keeps the retry policy in
// one place with the offline queue.
type sender struct {
	key    string
	url    string
	client *http.Client
}

func newSender(key, host string) *sender {
	if host == "" {
		host = DefaultHost
	}
	return &sender{
		key:    key,
		url:    strings.TrimRight(host, "/") + "/batch/",
		client: &http.Client{Timeout: sendTimeout},
	}
}

// send delivers a batch. A returned error means "retry this later" and the
// caller queues the batch to disk. A rejected-but-final response (4xx) is
// reported as success: retrying a malformed payload forever would grow the
// queue file without ever draining it.
func (s *sender) send(batch []Event) error {
	if len(batch) == 0 || s.key == "" {
		return nil
	}

	body, err := json.Marshal(batchPayload{APIKey: s.key, Batch: batch})
	if err != nil {
		return nil // unserialisable payload is a bug, not a network problem
	}

	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "jumpstart-analytics/1")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("posthog: status %d", resp.StatusCode)
	}
	return nil
}

func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
