package ghsync

import (
	"context"
	"errors"
	"sync"
	"time"

	"devdeck/internal/github"
)

// Polling cadence. GitHub has no push channel a desktop app can listen
// on, so "real-time" here means a moderate interval while the user is
// looking at the board, and a long one when they are not. Local edits
// still push (debounced) without waiting for a tick; the tick only
// catches changes made on github.com. These intervals are intentionally
// conservative — a full reconcile costs several GraphQL points per card.
const (
	FocusedInterval = 90 * time.Second
	IdleInterval    = 10 * time.Minute
	MinInterval     = 30 * time.Second
	MaxBackoff      = 15 * time.Minute
)

// SyncFunc runs one reconcile pass for a project. It returns the pass's
// result so the scheduler can react to rate limits and errors.
type SyncFunc func(ctx context.Context, projectID string) (*Result, error)

// Scheduler drives the polling loop for whichever project's board is on
// screen. Only one project polls at a time: the board the user is not
// looking at does not need a live view.
type Scheduler struct {
	mu       sync.Mutex
	sync     SyncFunc
	cancel   context.CancelFunc
	current  string
	focused  bool
	interval time.Duration
	backoff  time.Duration
}

// NewScheduler returns a scheduler that calls fn on each tick.
func NewScheduler(fn SyncFunc) *Scheduler {
	return &Scheduler{sync: fn, focused: true, interval: FocusedInterval}
}

// Watch starts polling projectID, replacing whatever was polling before.
// Passing an empty id stops polling entirely.
func (s *Scheduler) Watch(projectID string, override time.Duration) {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.current = projectID
	s.backoff = 0
	if override > 0 {
		s.interval = maxDuration(override, MinInterval)
	} else {
		s.interval = FocusedInterval
	}
	if projectID == "" {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()

	go s.loop(ctx, projectID)
}

// Stop ends polling.
func (s *Scheduler) Stop() { s.Watch("", 0) }

// SetFocused switches between the fast and slow cadence. The frontend
// calls it when the Tasks view gains or loses focus, and when the window
// is hidden.
func (s *Scheduler) SetFocused(focused bool) {
	s.mu.Lock()
	s.focused = focused
	s.mu.Unlock()
}

// Current reports the project being polled, mainly so the API layer can
// avoid restarting a loop that is already running.
func (s *Scheduler) Current() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *Scheduler) loop(ctx context.Context, projectID string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.nextDelay()):
		}
		if ctx.Err() != nil {
			return
		}

		_, err := s.sync(ctx, projectID)
		s.mu.Lock()
		if err != nil {
			// Honour GitHub's Retry-After when present; otherwise grow
			// exponentially so a revoked token or outage costs one
			// request every few minutes, not every tick.
			var rate *github.RateLimitError
			if errors.As(err, &rate) && rate.RetryAfter > 0 {
				s.backoff = maxDuration(rate.RetryAfter, MinInterval)
				if s.backoff > MaxBackoff {
					s.backoff = MaxBackoff
				}
			} else if s.backoff == 0 {
				s.backoff = 30 * time.Second
			} else if s.backoff < MaxBackoff {
				s.backoff *= 2
				if s.backoff > MaxBackoff {
					s.backoff = MaxBackoff
				}
			}
		} else {
			s.backoff = 0
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) nextDelay() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backoff > 0 {
		return s.backoff
	}
	if s.focused {
		return maxDuration(s.interval, MinInterval)
	}
	return IdleInterval
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
