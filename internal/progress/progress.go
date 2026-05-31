package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Sink receives download progress events. Implementations must be safe for
// concurrent Update calls (the parallel engine reports from many goroutines).
type Sink interface {
	Update(downloaded, total int64)
	Finish(path string)
	Error(err error)
}

// Silent ignores everything.
type Silent struct{}

func NewSilent() *Silent         { return &Silent{} }
func (Silent) Update(_, _ int64) {}
func (Silent) Finish(_ string)   {}
func (Silent) Error(_ error)     {}

// TTY renders a single-line progress bar to w.
type TTY struct {
	w     io.Writer
	name  string
	start time.Time
	mu    sync.Mutex
}

func NewTTY(w io.Writer, name string) *TTY {
	return &TTY{w: w, name: name, start: time.Now()}
}

func (t *TTY) Update(downloaded, total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	pct := 0.0
	if total > 0 {
		pct = float64(downloaded) / float64(total) * 100
	}
	elapsed := time.Since(t.start).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(downloaded) / elapsed
	}
	fmt.Fprintf(t.w, "\r%-24s %3.0f%%  %s/s", t.name, pct, humanBytes(int64(speed)))
}

func (t *TTY) Finish(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.w, "\r%-24s done  -> %s\n", t.name, path)
}

func (t *TTY) Error(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.w, "\r%-24s error: %v\n", t.name, err)
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}
