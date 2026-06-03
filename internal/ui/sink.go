package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/adityachaudhary99/yank/internal/progress"
)

// minRedraw throttles live redraws so a fast transfer doesn't spend its time
// repainting the terminal (which, on slow consoles, throttles the read loop).
const minRedraw = 50 * time.Millisecond

type sink struct {
	w      io.Writer
	theme  Theme
	caps   Capabilities
	now    func() time.Time
	name   string
	sum    string // checksum algo (e.g. "sha256"); shown as "<algo> ok" on finish
	start  time.Time
	last   time.Time // last redraw
	frame  int
	speeds []float64
	lastN  int64
	lastT  int64
	prevN  int64     // bytes at the last redraw (for instantaneous speed)
	prevT  time.Time // time of the last redraw
	mu     sync.Mutex
}

// NewSink returns a themed progress.Sink. name is the transfer label; sum is the
// checksum algorithm (or "") shown on the completion card. newSink is the test
// seam taking an injectable clock.
func NewSink(w io.Writer, t Theme, c Capabilities, name, sum string) progress.Sink {
	return newSink(w, t, c, time.Now, name, sum)
}

func newSink(w io.Writer, t Theme, c Capabilities, now func() time.Time, name, sum string) *sink {
	start := now()
	return &sink{w: w, theme: t, caps: c, now: now, name: name, sum: sum, start: start, last: start.Add(-time.Hour)}
}

func (s *sink) Update(done, total int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastN, s.lastT = done, total // always record, even when not drawing
	if !s.caps.TTY {               // non-tty: stay silent until Finish
		return
	}

	now := s.now()
	complete := total > 0 && done >= total
	if now.Sub(s.last) < minRedraw && !complete {
		return // throttle: skip this frame
	}
	s.last = now

	g := s.theme.Glyphs(s.caps)
	s.frame = (s.frame + 1) % len(g.Spinner)

	elapsed := now.Sub(s.start).Seconds()
	var speed float64 // cumulative average, for the readout + ETA (stable)
	if elapsed > 0 {
		speed = float64(done) / elapsed
	}
	// The sparkline samples instantaneous speed (Δbytes/Δt since the last
	// redraw) so it shows real variation instead of a flat average.
	var inst float64
	if !s.prevT.IsZero() {
		if dt := now.Sub(s.prevT).Seconds(); dt > 0 {
			inst = float64(done-s.prevN) / dt
		}
	}
	s.prevN, s.prevT = done, now
	s.speeds = append(s.speeds, inst)
	if len(s.speeds) > 40 {
		s.speeds = s.speeds[len(s.speeds)-40:]
	}

	pal := s.theme.Palette
	spin := paint(pal.Accent, g.Spinner[s.frame], s.caps)
	bar := s.bar(done, total, barWidth(s.caps.Width))
	pctStr := paint(pal.Accent, fmt.Sprintf("%d%%", pct(done, total)), s.caps)
	speedStr := humanBytes(int64(speed)) + "/s"

	line := fmt.Sprintf("\r%s %s  [%s]  %s  %s", spin, s.name, bar, pctStr, speedStr)
	if s.theme.Sparkline && s.caps.Unicode && len(s.speeds) > 1 {
		line += "  " + paint(pal.Dim, sparkline(s.speeds), s.caps)
	}
	line += "  eta " + etaStr(done, total, speed)
	fmt.Fprint(s.w, line+s.clear())
}

// bar renders a themed, colored progress bar: filled run + optional head + track.
func (s *sink) bar(done, total int64, width int) string {
	g := s.theme.Glyphs(s.caps)
	pal := s.theme.Palette
	filled := barCells(done, total, width)
	rest := width - filled
	head := ""
	if g.Head != "" && filled < width {
		head = g.Head
		rest--
	}
	if rest < 0 {
		rest = 0
	}
	return paint(pal.Fill, strings.Repeat(g.Fill, filled), s.caps) +
		paint(pal.Accent, head, s.caps) +
		paint(pal.Track, strings.Repeat(g.Track, rest), s.caps)
}

func (s *sink) Finish(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	ok := paint(s.theme.Palette.OK, g.OK, s.caps)
	size := s.lastT
	if s.lastN > size {
		size = s.lastN
	}
	elapsed := humanDur(s.now().Sub(s.start))

	card := fmt.Sprintf("%s %s  %s %s %s", ok, s.name, humanBytes(size), s.sep(), elapsed)
	if s.sum != "" {
		card += fmt.Sprintf(" %s %s ok", s.sep(), s.sum)
	}
	card += "  " + path
	fmt.Fprintf(s.w, "%s%s%s\n", s.cr(), card, s.clear())
}

func (s *sink) Error(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.theme.Glyphs(s.caps)
	fail := paint(s.theme.Palette.Fail, g.Fail, s.caps)
	fmt.Fprintf(s.w, "%s%s %s  error: %v%s\n", s.cr(), fail, s.name, err, s.clear())
}

// cr returns a carriage return to overwrite the live line on a TTY, or "" for a
// non-tty (where Update stays silent, so there is no line to overwrite).
func (s *sink) cr() string {
	if s.caps.TTY {
		return "\r"
	}
	return ""
}

// clear erases from the cursor to end of line, so a shorter line never leaves
// stale characters from a longer previous frame. TTY only.
func (s *sink) clear() string {
	if s.caps.TTY {
		return "\x1b[K"
	}
	return ""
}

// sep is the summary-card field separator: middle dot on unicode, ASCII dash
// otherwise (keeps --ascii output pure 7-bit).
func (s *sink) sep() string {
	if s.caps.Unicode {
		return "·"
	}
	return "-"
}

var _ progress.Sink = (*sink)(nil)
