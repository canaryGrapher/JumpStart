package procman

import "sync"

const maxLogLines = 2000

// logBuf is a bounded ring buffer of log lines.
type logBuf struct {
	mu    sync.Mutex
	lines []string
}

func (b *logBuf) add(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, line)
	if len(b.lines) > maxLogLines {
		b.lines = b.lines[len(b.lines)-maxLogLines:]
	}
}

func (b *logBuf) all() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.lines))
	copy(out, b.lines)
	return out
}
