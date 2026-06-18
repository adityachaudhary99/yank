package ui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// Stack renders one progress line per named transfer plus an aggregate footer.
// Returned children satisfy progress.Sink; updating any child recomputes the
// shared totals.
type Stack struct {
	w     io.Writer
	theme Theme
	caps  Capabilities
	now   func() time.Time
	start time.Time
	mu    sync.Mutex // guards kids' counters
	kids  []*stackChild
	wmu   sync.Mutex // serializes writes to w (many goroutines render)
	last  time.Time  // last redraw, for throttling
}

type stackChild struct {
	s     *Stack
	name  string
	done  int64
	total int64
	errd  bool
}

// New builds a Stack and one progress.Sink per name.
func New(w io.Writer, theme Theme, caps Capabilities, names []string) ([]progress.Sink, *Stack) {
	st := &Stack{w: w, theme: theme, caps: caps, now: time.Now, start: time.Now()}
	sinks := make([]progress.Sink, 0, len(names))
	for _, n := range names {
		c := &stackChild{s: st, name: n}
		st.kids = append(st.kids, c)
		sinks = append(sinks, c)
	}
	return sinks, st
}

func (c *stackChild) Update(done, total int64) {
	c.s.mu.Lock()
	c.done, c.total = done, total
	c.s.mu.Unlock()
	c.s.render()
}

func (c *stackChild) Finish(string) {
	c.s.mu.Lock()
	if c.total == 0 {
		c.total = c.done
	}
	c.done = c.total
	c.s.mu.Unlock()
	c.s.render()
}

func (c *stackChild) Error(error) {
	c.s.mu.Lock()
	c.errd = true
	c.s.mu.Unlock()
	c.s.render()
}

// Footer returns the aggregate status line ("total <done>/<total> <speed>/s").
func (s *Stack) Footer() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var done, total int64
	var errs int
	for _, k := range s.kids {
		done += k.done
		total += k.total
		if k.errd {
			errs++
		}
	}
	elapsed := s.now().Sub(s.start).Seconds()
	var speed int64
	if elapsed > 0 {
		speed = int64(float64(done) / elapsed)
	}
	line := fmt.Sprintf("total %s/%s  %s/s", humanBytes(done), humanBytes(total), humanBytes(speed))
	if errs > 0 {
		line += fmt.Sprintf("  (%d failed)", errs)
	}
	return line
}

func (s *Stack) render() {
	if s.w == nil || !s.caps.TTY {
		return
	}
	footer := s.Footer()
	s.wmu.Lock()
	defer s.wmu.Unlock()
	now := s.now()
	if now.Sub(s.last) < 50*time.Millisecond {
		return // throttle repaints under many concurrent updates
	}
	s.last = now
	fmt.Fprintf(s.w, "\r%s\x1b[K", footer)
}

// Done draws a final footer and moves to a new line, so later output doesn't
// collide with the live aggregate line. No-op off a TTY.
func (s *Stack) Done() {
	if s.w == nil || !s.caps.TTY {
		return
	}
	footer := s.Footer()
	s.wmu.Lock()
	defer s.wmu.Unlock()
	fmt.Fprintf(s.w, "\r%s\x1b[K\n", footer)
}
