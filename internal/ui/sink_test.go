package ui

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	t := time.Unix(0, 0)
	return func() time.Time { t = t.Add(time.Second); return t }
}

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestFitSparklineBudget(t *testing.T) {
	long := make([]float64, 40)
	// Plenty of width: capped at sparkMax (16), not all 40 samples.
	if got := fitSparkline(long, 200, 10, 30, 3, 8, 5); len(got) != 16 {
		t.Fatalf("wide terminal: len=%d want 16", len(got))
	}
	// Narrow terminal: the run shrinks to what's left after the rest of the line.
	got := fitSparkline(long, 80, 12, 26, 3, 8, 5)
	used := 1 + 1 + 12 + 3 + 26 + 3 + 3 + 2 + 8 + 2 + 6 + 5
	if want := 80 - used - 1; len(got) != want {
		t.Fatalf("narrow terminal: len=%d want %d", len(got), want)
	}
	// No room: empty (caller then skips the sparkline).
	if got := fitSparkline(long, 40, 20, 26, 3, 8, 5); len(got) != 0 {
		t.Fatalf("no room: len=%d want 0", len(got))
	}
}

// The matrix theme's live line (the only one with a sparkline) must never exceed
// the terminal width — a wrapped line breaks the \r redraw and floods stacked bars.
func TestMatrixLineNeverExceedsWidth(t *testing.T) {
	mtx, ok := ByName("matrix")
	if !ok {
		t.Fatal("no matrix theme")
	}
	// Realistic terminal widths: the sparkline must stay budgeted so the line
	// never wraps. (Sub-80 widths can still overflow on the fixed chrome alone —
	// a separate, pre-existing, all-theme concern, out of scope here.)
	for _, width := range []int{80, 100, 116, 160} {
		var buf bytes.Buffer
		caps := Capabilities{TTY: true, Color: true, Unicode: true, Width: width}
		s := newSink(&buf, mtx, caps, fixedClock(), "yt-dlp_linux", "")
		var last string
		for i := 1; i <= 60; i++ { // accumulate samples so the sparkline grows
			buf.Reset()
			s.Update(int64(i)*1_000_000, 100_000_000)
			last = buf.String()
		}
		line := last
		if i := strings.LastIndex(line, "\r"); i >= 0 {
			line = line[i+1:]
		}
		plain := stripANSI(line) // drops the trailing \x1b[K too
		if w := len([]rune(plain)); w > width {
			t.Fatalf("width %d: matrix line is %d cols: %q", width, w, plain)
		}
	}
}

func TestSinkASCIINoColor(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: false, Unicode: false, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(61, 100)
	s.Finish("./file.iso")
	out := buf.String()
	if !strings.Contains(out, "file.iso") || !strings.Contains(out, "61%") {
		t.Fatalf("missing name/percent: %q", out)
	}
	// The only escape allowed in no-color mode is the erase-to-EOL control;
	// strip it and assert there are no SGR (color) sequences left.
	clean := strings.ReplaceAll(out, "\x1b[K", "")
	if strings.Contains(clean, "\x1b[") {
		t.Fatalf("no-color sink emitted color ANSI: %q", out)
	}
	for i := 0; i < len(out); i++ {
		if out[i] >= 0x80 {
			t.Fatalf("ascii sink emitted non-ascii byte %#x in %q", out[i], out)
		}
	}
}

func TestSinkColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: true, Unicode: true, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(61, 100)
	if !strings.Contains(buf.String(), "\x1b[38") {
		t.Fatal("color sink should emit SGR color codes")
	}
}

func TestSinkSummaryCardHasSizeAndChecksum(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: true, Color: false, Unicode: false, Width: 60}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "sha256")
	s.Update(1048576, 1048576)
	s.Finish("/tmp/file.iso")
	out := buf.String()
	for _, want := range []string{"file.iso", "1.0MB", "sha256 ok", "/tmp/file.iso"} {
		if !strings.Contains(out, want) {
			t.Fatalf("summary card missing %q: %q", want, out)
		}
	}
}

func TestSinkNonTTYPlainSummary(t *testing.T) {
	var buf bytes.Buffer
	caps := Capabilities{TTY: false, Width: 80}
	s := newSink(&buf, Default(), caps, fixedClock(), "file.iso", "")
	s.Update(50, 100) // no redraws on non-tty
	s.Finish("./file.iso")
	out := buf.String()
	if strings.Count(out, "\n") != 1 || strings.Contains(out, "\r") {
		t.Fatalf("non-tty should print exactly one summary line: %q", out)
	}
}
