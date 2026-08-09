package analytics

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

// queue is a disk-backed buffer of events that could not be delivered.
//
// Offline buffering matters more for a desktop dev tool than it does on the
// web: people work on planes, on hotel wifi, and behind corporate proxies.
// Without this, every event from an offline session is lost and the funnel
// silently under-counts exactly the sessions that are most interesting.
type queue struct {
	mu   sync.Mutex
	path string
	max  int
}

func newQueue(path string) *queue {
	return &queue{path: path, max: queueMax}
}

// append writes events as newline-delimited JSON, trimming the oldest once
// the file passes max. The cap keeps a long offline stretch from turning
// into an unbounded file in the user's home directory.
func (q *queue) append(batch []Event) {
	if q == nil || len(batch) == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	existing := q.readLocked()
	combined := append(existing, batch...)
	if len(combined) > q.max {
		combined = combined[len(combined)-q.max:]
	}
	q.writeLocked(combined)
}

// drain returns every queued event and clears the file. Called once at
// startup, so a failed session's events are retried on the next launch.
func (q *queue) drain() []Event {
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	events := q.readLocked()
	if len(events) > 0 {
		_ = os.Remove(q.path)
	}
	return events
}

// purge deletes the file outright. Called when the user turns analytics
// off: anything still buffered was collected under the old consent and
// must not be sent later.
func (q *queue) purge() {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	_ = os.Remove(q.path)
}

func (q *queue) readLocked() []Event {
	f, err := os.Open(q.path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []Event
	scanner := bufio.NewScanner(f)
	// Individual events are small; the default 64KB token limit is ample
	// and doubles as a guard against a corrupted file.
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err == nil && ev.Name != "" {
			out = append(out, ev)
		}
	}
	return out
}

func (q *queue) writeLocked(events []Event) {
	f, err := os.OpenFile(q.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			break
		}
	}
	_ = w.Flush()
}
