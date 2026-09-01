package analytics

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Client is the only thing call sites touch. A nil *Client is safe: every
// method returns immediately, so code paths that run before Startup (or in
// tests) need no guards.
type Client struct {
	opts     Options
	distinct string
	session  string

	globalMu sync.RWMutex
	global   map[string]any

	enabled atomic.Bool
	closed  atomic.Bool
	counter atomic.Int64

	prefsMu    sync.RWMutex
	detailLevel string
	categories  map[string]bool

	events chan Event
	queue  *queue
	sender *sender

	done      chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	closeOnce sync.Once
	seen      sync.Map // keys for once-per-session events
}

// New builds a client and resolves identity. It does not open a socket, a
// goroutine, or the queue file: the worker starts on the first event that
// actually passes the consent gate, so a user with analytics off pays
// nothing beyond a struct.
func New(opts Options) *Client {
	if opts.Dir == "" {
		opts.Dir = DataDir()
	}

	id, first := installID(opts.Dir)
	consent := LoadConsent(opts.Dir)
	c := &Client{
		opts:        opts,
		distinct:    id,
		session:     randomID(),
		events:      make(chan Event, bufferSize),
		queue:       newQueue(filepath.Join(opts.Dir, "analytics_queue.ndjson")),
		sender:      newSender(opts),
		done:        make(chan struct{}),
		detailLevel: consent.DetailLevel,
		categories:  MergeCategories(consent.Categories),
	}
	c.global = globalProps(opts, first, c.session)
	c.enabled.Store(consent.Enabled)
	return c
}

// Enabled reports whether events are being collected.
func (c *Client) Enabled() bool {
	return c != nil && c.enabled.Load()
}

// SetEnabled persists the user's choice and applies it immediately.
// Turning analytics off also drops anything still buffered on disk: that
// data was collected under the old answer and must not be sent later.
func (c *Client) SetEnabled(enabled bool) error {
	if c == nil {
		return nil
	}
	c.enabled.Store(enabled)
	if !enabled {
		c.queue.purge()
	}
	return SaveConsent(c.opts.Dir, enabled)
}

// Prefs returns the current master switch, detail level, and categories.
func (c *Client) Prefs() Prefs {
	if c == nil {
		return Prefs{
			Enabled:     false,
			DetailLevel: LevelFull,
			Categories:  DefaultCategories(),
		}
	}
	c.prefsMu.RLock()
	defer c.prefsMu.RUnlock()
	cats := MergeCategories(c.categories)
	return Prefs{
		Enabled:     c.enabled.Load(),
		DetailLevel: c.detailLevel,
		Categories:  cats,
	}
}

// SetDetailLevel applies a named preset (full/balanced/minimal/none) and persists it.
func (c *Client) SetDetailLevel(level string) error {
	if c == nil {
		return nil
	}
	level = NormalizeDetailLevel(level)
	if level == LevelCustom {
		level = LevelFull
	}
	cats := CategoriesForLevel(level)
	c.prefsMu.Lock()
	c.detailLevel = level
	c.categories = cats
	c.prefsMu.Unlock()
	return SaveAnalyticsPrefs(c.opts.Dir, Prefs{
		Enabled:     c.enabled.Load(),
		DetailLevel: level,
		Categories:  cats,
	})
}

// SetCategories stores per-category toggles, marks detail level custom (or
// snaps back to a matching preset), and persists.
func (c *Client) SetCategories(cats map[string]bool) error {
	if c == nil {
		return nil
	}
	merged := MergeCategories(cats)
	level := InferDetailLevel(merged)
	c.prefsMu.Lock()
	c.categories = merged
	c.detailLevel = level
	c.prefsMu.Unlock()
	return SaveAnalyticsPrefs(c.opts.Dir, Prefs{
		Enabled:     c.enabled.Load(),
		DetailLevel: level,
		Categories:  merged,
	})
}

// categoryAllowed reports whether the event's category is currently enabled.
func (c *Client) categoryAllowed(name string) bool {
	if c == nil {
		return false
	}
	cat := string(EventCategory(name))
	c.prefsMu.RLock()
	defer c.prefsMu.RUnlock()
	return boolOr(c.categories, cat)
}

// Configured reports whether this build can send at all. Dev builds have no
// GA4 credentials, so the UI can explain why the toggle does nothing.
func (c *Client) Configured() bool {
	return c != nil && c.opts.MeasurementID != "" && c.opts.APISecret != ""
}

// SessionID identifies one app run. It is regenerated every launch, so it
// groups a session's events without acting as a persistent identifier.
func (c *Client) SessionID() string {
	if c == nil {
		return ""
	}
	return c.session
}

// Ref maps a local ID to a per-install opaque token. See identity.Ref.
func (c *Client) Ref(value string) string {
	if c == nil {
		return ""
	}
	return Ref(c.distinct, value)
}

// EventCount is how many events this session has emitted.
func (c *Client) EventCount() int64 {
	if c == nil {
		return 0
	}
	return c.counter.Load()
}

// Track queues one event. It never blocks and never returns an error: it is
// called from Wails bindings on the UI's critical path, and analytics must
// never be the reason a button feels slow. When the buffer is full the
// event is dropped, because a dropped event is cheaper than a stalled UI.
func (c *Client) Track(name string, props map[string]any) {
	if c == nil || name == "" || !c.enabled.Load() || !c.Configured() || c.closed.Load() {
		return
	}
	if !c.categoryAllowed(name) {
		return
	}
	c.start()
	c.counter.Add(1)

	select {
	case c.events <- c.build(name, props):
	default:
	}
}

// TrackOnce emits at most one event per key per session. Used for things
// that are interesting as reach but noisy as volume, most importantly
// panel_opened: instrumenting every click would multiply event volume by an
// order of magnitude and answer a question nobody asked.
func (c *Client) TrackOnce(key, name string, props map[string]any) {
	if c == nil {
		return
	}
	if _, seen := c.seen.LoadOrStore(key, struct{}{}); seen {
		return
	}
	c.Track(name, props)
}

// Close flushes what is buffered and stops the worker. The timeout exists
// because Shutdown must not hang: an unreachable network on quit costs the
// app_closed event, not the user's patience.
func (c *Client) Close(timeout time.Duration) {
	if c == nil {
		return
	}
	c.closed.Store(true)
	c.closeOnce.Do(func() { close(c.done) })

	finished := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(timeout):
	}
}

// SetGlobal overrides one global property for the rest of the session.
// It exists for facts the Go side cannot know at Startup, most notably the
// update channel, which the frontend owns.
func (c *Client) SetGlobal(key string, value any) {
	if c == nil || key == "" {
		return
	}
	c.globalMu.Lock()
	defer c.globalMu.Unlock()
	c.global[key] = value
}

func (c *Client) build(name string, props map[string]any) Event {
	c.globalMu.RLock()
	merged := make(map[string]any, len(c.global)+len(props)+2)
	for k, v := range c.global {
		merged[k] = v
	}
	c.globalMu.RUnlock()

	for k, v := range Sanitize(props) {
		merged[k] = v
	}
	if IsKeyEvent(name, merged) {
		merged["is_key_event"] = true
	}
	return Event{
		Name:      name,
		ClientID:  c.distinct,
		Params:    merged,
		Timestamp: nowTimestamp(),
	}
}

func (c *Client) start() {
	c.startOnce.Do(func() {
		c.wg.Add(1)
		go c.run()
	})
}

func (c *Client) run() {
	defer c.wg.Done()

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	// Whatever the last session could not deliver goes out first, so an
	// offline session's events are not permanently lost.
	batch := c.flush(c.queue.drain())

	for {
		select {
		case ev := <-c.events:
			batch = append(batch, ev)
			if len(batch) >= batchSize {
				batch = c.flush(batch)
			}
		case <-ticker.C:
			batch = c.flush(batch)
		case <-c.done:
			batch = append(batch, c.drainChannel()...)
			c.flush(batch)
			return
		}
	}
}

// drainChannel empties the buffered channel without blocking, so a clean
// shutdown does not lose events that were queued microseconds earlier.
func (c *Client) drainChannel() []Event {
	var out []Event
	for {
		select {
		case ev := <-c.events:
			out = append(out, ev)
		default:
			return out
		}
	}
}

// flush sends a batch and returns an empty slice to accumulate into.
// Failures go to the disk queue for the next launch.
func (c *Client) flush(batch []Event) []Event {
	if len(batch) == 0 {
		return batch[:0]
	}
	if err := c.sender.send(batch); err != nil {
		c.queue.append(batch)
	}
	return batch[:0]
}
