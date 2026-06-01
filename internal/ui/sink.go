package ui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

type sink struct {
	w      io.Writer
	theme  Theme
	caps   Capabilities
	now    func() time.Time
	name   string
	start  time.Time
	frame  int
	speeds []float64
	mu     sync.Mutex
}

// NewSink returns a themed progress.Sink. Exported for the CLI; newSink is the
// test seam taking an injectable clock.
func NewSink(w io.Writer, t Theme, c Capabilities, name string) progress.Sink {
	return newSink(w, t, c, time.Now, name)
}

func newSink(w io.Writer, t Theme, c Capabilities, now func() time.Time, name string) *sink {
	return &sink{w: w, theme: t, caps: c, now: now, name: name, start: now()}
}

func (s *sink) Update(done, total int64) {
	if !s.caps.TTY { // non-tty: stay silent until Finish
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	s.frame = (s.frame + 1) % len(g.Spinner)

	elapsed := s.now().Sub(s.start).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(done) / elapsed
	}
	s.speeds = append(s.speeds, speed)
	if len(s.speeds) > 40 {
		s.speeds = s.speeds[len(s.speeds)-40:]
	}

	pal := s.theme.Palette
	spin := paint(pal.Accent, g.Spinner[s.frame], s.caps)
	bar := paint(pal.Fill, renderBar(done, total, barWidth(s.caps.Width), g, s.caps), s.caps)
	pctStr := paint(pal.Accent, fmt.Sprintf("%d%%", pct(done, total)), s.caps)

	line := fmt.Sprintf("\r%s %s  [%s]  %s  %s/s", spin, s.name, bar, pctStr, humanBytes(int64(speed)))
	if s.caps.Unicode && len(s.speeds) > 1 {
		line += "  " + paint(pal.Dim, sparkline(s.speeds), s.caps)
	}
	fmt.Fprint(s.w, line)
}

func (s *sink) Finish(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	elapsed := s.now().Sub(s.start).Round(time.Second)
	ok := paint(s.theme.Palette.OK, g.OK, s.caps)
	// summary card: "{ok} {name}  {elapsed} {sep} {path}"
	fmt.Fprintf(s.w, "%s%s %s  %s %s %s\n", s.cr(), ok, s.name, elapsed, s.sep(), path)
}

func (s *sink) Error(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	fail := paint(s.theme.Palette.Fail, g.Fail, s.caps)
	fmt.Fprintf(s.w, "%s%s %s  error: %v\n", s.cr(), fail, s.name, err)
}

// cr returns a carriage return to overwrite the live line on a TTY, or "" for a
// non-tty (where Update stays silent, so there is no line to overwrite).
func (s *sink) cr() string {
	if s.caps.TTY {
		return "\r"
	}
	return ""
}

// sep returns the summary-card field separator: a middle dot when unicode is
// available, an ASCII dash otherwise (keeps --ascii output pure 7-bit).
func (s *sink) sep() string {
	if s.caps.Unicode {
		return "·"
	}
	return "-"
}

var _ progress.Sink = (*sink)(nil)
