package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Event is the offline-queue and in-memory batch unit.
type Event struct {
	Name      string         `json:"name"`
	ClientID  string         `json:"client_id"`
	Params    map[string]any `json:"params"`
	Timestamp string         `json:"timestamp"`
}

type sender struct {
	measurementID string
	apiSecret     string
	baseURL       string
	client        *http.Client
}

func newSender(opts Options) *sender {
	base := opts.CollectURL
	if base == "" {
		base = DefaultCollectURL
	}
	return &sender{
		measurementID: opts.MeasurementID,
		apiSecret:     opts.APISecret,
		baseURL:       strings.TrimRight(base, "/"),
		client:        &http.Client{Timeout: sendTimeout},
	}
}

type mpEvent struct {
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

type mpPayload struct {
	ClientID           string    `json:"client_id"`
	TimestampMicros    int64     `json:"timestamp_micros,omitempty"`
	NonPersonalizedAds bool      `json:"non_personalized_ads"`
	Events             []mpEvent `json:"events"`
}

func (s *sender) send(batch []Event) error {
	if len(batch) == 0 || s.measurementID == "" || s.apiSecret == "" {
		return nil
	}
	// GA4 allows max 25 events per request.
	for i := 0; i < len(batch); i += 25 {
		end := i + 25
		if end > len(batch) {
			end = len(batch)
		}
		if err := s.sendChunk(batch[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *sender) sendChunk(batch []Event) error {
	clientID := batch[0].ClientID
	events := make([]mpEvent, 0, len(batch))
	var oldestMicros int64
	for _, ev := range batch {
		if ev.ClientID != "" {
			clientID = ev.ClientID
		}
		events = append(events, mpEvent{Name: ev.Name, Params: ev.Params})
		if micros := rfc3339ToMicros(ev.Timestamp); micros > 0 && (oldestMicros == 0 || micros < oldestMicros) {
			oldestMicros = micros
		}
	}
	body, err := json.Marshal(mpPayload{
		ClientID:           clientID,
		TimestampMicros:    oldestMicros,
		NonPersonalizedAds: true,
		Events:             events,
	})
	if err != nil {
		return nil
	}

	u, err := url.Parse(s.baseURL)
	if err != nil {
		return nil
	}
	q := u.Query()
	q.Set("measurement_id", s.measurementID)
	q.Set("api_secret", s.apiSecret)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
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

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("ga4 mp: status %d", resp.StatusCode)
	}
	return fmt.Errorf("ga4 mp: status %d", resp.StatusCode)
}

func rfc3339ToMicros(ts string) int64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	return t.UnixMicro()
}

func nowTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
