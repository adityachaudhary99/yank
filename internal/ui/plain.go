package ui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// plainInterval throttles plain progress lines so a long transfer prints a
// readable trickle (one line a second), not a flood — while still showing
// liveness in CI logs and to screen readers.
const plainInterval = time.Second

// plainSink is a line-oriented, append-only progress.Sink: no carriage returns,
// no color, no spinner or sparkline — pure ASCII lines, one per event. It is
// used for screen readers, CI logs, and dumb terminals (Capabilities.Plain).
type plainSink struct {
	w           io.Writer
	name        string
	sum         string // checksum algo (e.g. "sha256"); shown as ", <algo> ok" on finish
	now         func() time.Time
	start       time.Time
	last        time.Time
	lastEmitted bool
	lastN       int64
	lastT       int64
	mu          sync.Mutex
}

func newPlainSink(w io.Writer, now func() time.Time, name, sum string) *plainSink {
	return &plainSink{w: w, now: now, name: name, sum: sum, start: now()}
}

func (p *plainSink) Update(done, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastN, p.lastT = done, total
	now := p.now()
	// Emit the first update immediately (a "starting" line), then throttle.
	if p.lastEmitted && now.Sub(p.last) < plainInterval {
		return
	}
	p.last = now
	p.lastEmitted = true
	speed := "0B/s"
	if elapsed := now.Sub(p.start).Seconds(); elapsed > 0 {
		speed = humanBytes(int64(float64(done)/elapsed)) + "/s"
	}
	if total > 0 {
		fmt.Fprintf(p.w, "%s: %d%% (%s/%s) at %s\n", p.name, pct(done, total), humanBytes(done), humanBytes(total), speed)
	} else {
		fmt.Fprintf(p.w, "%s: %s at %s\n", p.name, humanBytes(done), speed)
	}
}

// Resuming reports a resumed transfer (the engine's optional resumeNotifier).
func (p *plainSink) Resuming(done, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "%s: resuming from %d%%\n", p.name, pct(done, total))
}

func (p *plainSink) Finish(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	size := p.lastT
	if p.lastN > size {
		size = p.lastN
	}
	detail := fmt.Sprintf("%s in %s", humanBytes(size), humanDur(p.now().Sub(p.start)))
	if p.sum != "" {
		detail += ", " + p.sum + " ok"
	}
	if path != "" {
		fmt.Fprintf(p.w, "done: %s (%s) -> %s\n", p.name, detail, path)
	} else {
		fmt.Fprintf(p.w, "done: %s (%s)\n", p.name, detail)
	}
}

func (p *plainSink) Error(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "error: %s: %v\n", p.name, err)
}

var _ progress.Sink = (*plainSink)(nil)
